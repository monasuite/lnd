package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lightninglabs/neutrino"
	"github.com/wakiyamap/lnd/chainntnfs"
	"github.com/wakiyamap/lnd/chainntnfs/bitcoindnotify"
	"github.com/wakiyamap/lnd/chainntnfs/btcdnotify"
	"github.com/wakiyamap/lnd/chainntnfs/neutrinonotify"
	"github.com/wakiyamap/lnd/channeldb"
	"github.com/wakiyamap/lnd/htlcswitch"
	"github.com/wakiyamap/lnd/keychain"
	"github.com/wakiyamap/lnd/lnwallet"
	"github.com/wakiyamap/lnd/lnwallet/btcwallet"
	"github.com/wakiyamap/lnd/lnwire"
	"github.com/wakiyamap/lnd/routing/chainview"
	"github.com/roasbeef/btcd/chaincfg/chainhash"
	"github.com/roasbeef/btcd/rpcclient"
	"github.com/roasbeef/btcutil"
	"github.com/roasbeef/btcwallet/chain"
	"github.com/roasbeef/btcwallet/walletdb"
)

const (
	defaultBitcoinMinHTLCMSat   = lnwire.MilliSatoshi(1000)
	defaultBitcoinBaseFeeMSat   = lnwire.MilliSatoshi(1000)
	defaultBitcoinFeeRate       = lnwire.MilliSatoshi(1)
	defaultBitcoinTimeLockDelta = 144
	defaultBitcoinStaticFeeRate = lnwallet.SatPerVByte(50)

	defaultMonacoinMinHTLCMSat   = lnwire.MilliSatoshi(1000)
	defaultMonacoinBaseFeeMSat   = lnwire.MilliSatoshi(1000)
	defaultMonacoinFeeRate       = lnwire.MilliSatoshi(1)
	defaultMonacoinTimeLockDelta = 960
	defaultMonacoinStaticFeeRate = lnwallet.SatPerVByte(200)
	defaultMonacoinDustLimit     = btcutil.Amount(54600)
)

// defaultBtcChannelConstraints is the default set of channel constraints that are
// meant to be used when initially funding a Bitcoin channel.
//
// TODO(halseth): make configurable at startup?
var defaultBtcChannelConstraints = channeldb.ChannelConstraints{
	DustLimit:        lnwallet.DefaultDustLimit(),
	MaxAcceptedHtlcs: lnwallet.MaxHTLCNumber / 2,
}

// defaultMonaChannelConstraints is the default set of channel constraints that are
// meant to be used when initially funding a Monacoin channel.
var defaultMonaChannelConstraints = channeldb.ChannelConstraints{
	DustLimit:        defaultMonacoinDustLimit,
	MaxAcceptedHtlcs: lnwallet.MaxHTLCNumber / 2,
}

// chainCode is an enum-like structure for keeping track of the chains
// currently supported within lnd.
type chainCode uint32

const (
	// bitcoinChain is Bitcoin's testnet chain.
	bitcoinChain chainCode = iota

	// monacoinChain is Monacoin's testnet chain.
	monacoinChain
)

// String returns a string representation of the target chainCode.
func (c chainCode) String() string {
	switch c {
	case bitcoinChain:
		return "bitcoin"
	case monacoinChain:
		return "monacoin"
	default:
		return "kekcoin"
	}
}

// chainControl couples the three primary interfaces lnd utilizes for a
// particular chain together. A single chainControl instance will exist for all
// the chains lnd is currently active on.
type chainControl struct {
	chainIO lnwallet.BlockChainIO

	feeEstimator lnwallet.FeeEstimator

	signer lnwallet.Signer

	msgSigner lnwallet.MessageSigner

	chainNotifier chainntnfs.ChainNotifier

	chainView chainview.FilteredChainView

	wallet *lnwallet.LightningWallet

	routingPolicy htlcswitch.ForwardingPolicy
}

// newChainControlFromConfig attempts to create a chainControl instance
// according to the parameters in the passed lnd configuration. Currently two
// branches of chainControl instances exist: one backed by a running btcd
// full-node, and the other backed by a running neutrino light client instance.
func newChainControlFromConfig(cfg *config, chanDB *channeldb.DB,
	privateWalletPw, publicWalletPw []byte) (*chainControl, func(), error) {

	// Set the RPC config from the "home" chain. Multi-chain isn't yet
	// active, so we'll restrict usage to a particular chain for now.
	homeChainConfig := cfg.Bitcoin
	if registeredChains.PrimaryChain() == monacoinChain {
		homeChainConfig = cfg.Monacoin
	}
	ltndLog.Infof("Primary chain is set to: %v",
		registeredChains.PrimaryChain())

	cc := &chainControl{}

	switch registeredChains.PrimaryChain() {
	case bitcoinChain:
		cc.routingPolicy = htlcswitch.ForwardingPolicy{
			MinHTLC:       cfg.Bitcoin.MinHTLC,
			BaseFee:       cfg.Bitcoin.BaseFee,
			FeeRate:       cfg.Bitcoin.FeeRate,
			TimeLockDelta: cfg.Bitcoin.TimeLockDelta,
		}
		cc.feeEstimator = lnwallet.StaticFeeEstimator{
			FeeRate: defaultBitcoinStaticFeeRate,
		}
	case monacoinChain:
		cc.routingPolicy = htlcswitch.ForwardingPolicy{
			MinHTLC:       cfg.Monacoin.MinHTLC,
			BaseFee:       cfg.Monacoin.BaseFee,
			FeeRate:       cfg.Monacoin.FeeRate,
			TimeLockDelta: cfg.Monacoin.TimeLockDelta,
		}
		cc.feeEstimator = lnwallet.StaticFeeEstimator{
			FeeRate: defaultMonacoinStaticFeeRate,
		}
	default:
		return nil, nil, fmt.Errorf("Default routing policy for "+
			"chain %v is unknown", registeredChains.PrimaryChain())
	}

	walletConfig := &btcwallet.Config{
		PrivatePass:  privateWalletPw,
		PublicPass:   publicWalletPw,
		DataDir:      homeChainConfig.ChainDir,
		NetParams:    activeNetParams.Params,
		FeeEstimator: cc.feeEstimator,
		CoinType:     activeNetParams.CoinType,
	}

	var (
		err          error
		cleanUp      func()
		btcdConn     *chain.RPCClient
		bitcoindConn *chain.BitcoindClient
	)

	// If spv mode is active, then we'll be using a distinct set of
	// chainControl interfaces that interface directly with the p2p network
	// of the selected chain.
	switch homeChainConfig.Node {
	case "neutrino":
		// First we'll open the database file for neutrino, creating
		// the database if needed. We append the normalized network name
		// here to match the behavior of btcwallet.
		neutrinoDbPath := filepath.Join(homeChainConfig.ChainDir,
			normalizeNetwork(activeNetParams.Name))

		// Ensure that the neutrino db path exists.
		if err := os.MkdirAll(neutrinoDbPath, 0700); err != nil {
			return nil, nil, err
		}

		dbName := filepath.Join(neutrinoDbPath, "neutrino.db")
		nodeDatabase, err := walletdb.Create("bdb", dbName)
		if err != nil {
			return nil, nil, err
		}

		// With the database open, we can now create an instance of the
		// neutrino light client. We pass in relevant configuration
		// parameters required.
		config := neutrino.Config{
			DataDir:      neutrinoDbPath,
			Database:     nodeDatabase,
			ChainParams:  *activeNetParams.Params,
			AddPeers:     cfg.NeutrinoMode.AddPeers,
			ConnectPeers: cfg.NeutrinoMode.ConnectPeers,
			Dialer: func(addr net.Addr) (net.Conn, error) {
				return cfg.net.Dial(addr.Network(), addr.String())
			},
			NameResolver: func(host string) ([]net.IP, error) {
				addrs, err := cfg.net.LookupHost(host)
				if err != nil {
					return nil, err
				}

				ips := make([]net.IP, 0, len(addrs))
				for _, strIP := range addrs {
					ip := net.ParseIP(strIP)
					if ip == nil {
						continue
					}

					ips = append(ips, ip)
				}

				return ips, nil
			},
		}
		neutrino.WaitForMoreCFHeaders = time.Second * 1
		neutrino.MaxPeers = 8
		neutrino.BanDuration = 5 * time.Second
		svc, err := neutrino.NewChainService(config)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to create neutrino: %v", err)
		}
		svc.Start()

		// Next we'll create the instances of the ChainNotifier and
		// FilteredChainView interface which is backed by the neutrino
		// light client.
		cc.chainNotifier, err = neutrinonotify.New(svc)
		if err != nil {
			return nil, nil, err
		}
		cc.chainView, err = chainview.NewCfFilteredChainView(svc)
		if err != nil {
			return nil, nil, err
		}

		// Finally, we'll set the chain source for btcwallet, and
		// create our clean up function which simply closes the
		// database.
		walletConfig.ChainSource = chain.NewNeutrinoClient(svc)
		cleanUp = func() {
			svc.Stop()
			nodeDatabase.Close()
		}
	case "bitcoind", "monacoind":
		var bitcoindMode *bitcoindConfig
		switch {
		case cfg.Bitcoin.Active:
			bitcoindMode = cfg.BitcoindMode
		case cfg.Monacoin.Active:
			bitcoindMode = cfg.MonacoindMode
		}
		// Otherwise, we'll be speaking directly via RPC and ZMQ to a
		// bitcoind node. If the specified host for the btcd/monad RPC
		// server already has a port specified, then we use that
		// directly. Otherwise, we assume the default port according to
		// the selected chain parameters.
		var bitcoindHost string
		if strings.Contains(bitcoindMode.RPCHost, ":") {
			bitcoindHost = bitcoindMode.RPCHost
		} else {
			// The RPC ports specified in chainparams.go assume
			// btcd, which picks a different port so that btcwallet
			// can use the same RPC port as bitcoind. We convert
			// this back to the btcwallet/bitcoind port.
			rpcPort, err := strconv.Atoi(activeNetParams.rpcPort)
			if err != nil {
				return nil, nil, err
			}
			rpcPort -= 2
			bitcoindHost = fmt.Sprintf("%v:%d",
				bitcoindMode.RPCHost, rpcPort)
			if cfg.Bitcoin.Active && cfg.Bitcoin.RegTest {
				conn, err := net.Dial("tcp", bitcoindHost)
				if err != nil || conn == nil {
					rpcPort = 18443
					bitcoindHost = fmt.Sprintf("%v:%d",
						bitcoindMode.RPCHost,
						rpcPort)
				} else {
					conn.Close()
				}
			}
		}

		bitcoindUser := bitcoindMode.RPCUser
		bitcoindPass := bitcoindMode.RPCPass
		rpcConfig := &rpcclient.ConnConfig{
			Host:                 bitcoindHost,
			User:                 bitcoindUser,
			Pass:                 bitcoindPass,
			DisableConnectOnNew:  true,
			DisableAutoReconnect: false,
			DisableTLS:           true,
			HTTPPostMode:         true,
		}
		cc.chainNotifier, err = bitcoindnotify.New(rpcConfig,
			bitcoindMode.ZMQPath, *activeNetParams.Params)
		if err != nil {
			return nil, nil, err
		}

		// Next, we'll create an instance of the bitcoind chain view to
		// be used within the routing layer.
		cc.chainView, err = chainview.NewBitcoindFilteredChainView(
			*rpcConfig, bitcoindMode.ZMQPath,
			*activeNetParams.Params)
		if err != nil {
			srvrLog.Errorf("unable to create chain view: %v", err)
			return nil, nil, err
		}

		// Create a special rpc+ZMQ client for bitcoind which will be
		// used by the wallet for notifications, calls, etc.
		bitcoindConn, err = chain.NewBitcoindClient(
			activeNetParams.Params, bitcoindHost, bitcoindUser,
			bitcoindPass, bitcoindMode.ZMQPath,
			time.Millisecond*100)
		if err != nil {
			return nil, nil, err
		}

		walletConfig.ChainSource = bitcoindConn

		// If we're not in regtest mode, then we'll attempt to use a
		// proper fee estimator for testnet.
		if cfg.Bitcoin.Active && !cfg.Bitcoin.RegTest {
			ltndLog.Infof("Initializing bitcoind backed fee estimator")

			// Finally, we'll re-initialize the fee estimator, as
			// if we're using bitcoind as a backend, then we can
			// use live fee estimates, rather than a statically
			// coded value.
			fallBackFeeRate := lnwallet.SatPerVByte(25)
			cc.feeEstimator, err = lnwallet.NewBitcoindFeeEstimator(
				*rpcConfig, fallBackFeeRate,
			)
			if err != nil {
				return nil, nil, err
			}
			if err := cc.feeEstimator.Start(); err != nil {
				return nil, nil, err
			}
		} else if cfg.Monacoin.Active {
			ltndLog.Infof("Initializing monacoind backed fee estimator")

			// Finally, we'll re-initialize the fee estimator, as
			// if we're using monacoind as a backend, then we can
			// use live fee estimates, rather than a statically
			// coded value.
			fallBackFeeRate := lnwallet.SatPerVByte(25)
			cc.feeEstimator, err = lnwallet.NewBitcoindFeeEstimator(
				*rpcConfig, fallBackFeeRate,
			)
			if err != nil {
				return nil, nil, err
			}
			if err := cc.feeEstimator.Start(); err != nil {
				return nil, nil, err
			}
		}
	case "btcd", "monad":
		// Otherwise, we'll be speaking directly via RPC to a node.
		//
		// So first we'll load btcd/monad's TLS cert for the RPC
		// connection. If a raw cert was specified in the config, then
		// we'll set that directly. Otherwise, we attempt to read the
		// cert from the path specified in the config.
		var btcdMode *btcdConfig
		switch {
		case cfg.Bitcoin.Active:
			btcdMode = cfg.BtcdMode
		case cfg.Monacoin.Active:
			btcdMode = cfg.MonadMode
		}
		var rpcCert []byte
		if btcdMode.RawRPCCert != "" {
			rpcCert, err = hex.DecodeString(btcdMode.RawRPCCert)
			if err != nil {
				return nil, nil, err
			}
		} else {
			certFile, err := os.Open(btcdMode.RPCCert)
			if err != nil {
				return nil, nil, err
			}
			rpcCert, err = ioutil.ReadAll(certFile)
			if err != nil {
				return nil, nil, err
			}
			if err := certFile.Close(); err != nil {
				return nil, nil, err
			}
		}

		// If the specified host for the btcd/monad RPC server already
		// has a port specified, then we use that directly. Otherwise,
		// we assume the default port according to the selected chain
		// parameters.
		var btcdHost string
		if strings.Contains(btcdMode.RPCHost, ":") {
			btcdHost = btcdMode.RPCHost
		} else {
			btcdHost = fmt.Sprintf("%v:%v", btcdMode.RPCHost,
				activeNetParams.rpcPort)
		}

		btcdUser := btcdMode.RPCUser
		btcdPass := btcdMode.RPCPass
		rpcConfig := &rpcclient.ConnConfig{
			Host:                 btcdHost,
			Endpoint:             "ws",
			User:                 btcdUser,
			Pass:                 btcdPass,
			Certificates:         rpcCert,
			DisableTLS:           false,
			DisableConnectOnNew:  true,
			DisableAutoReconnect: false,
		}
		cc.chainNotifier, err = btcdnotify.New(rpcConfig)
		if err != nil {
			return nil, nil, err
		}

		// Finally, we'll create an instance of the default chain view to be
		// used within the routing layer.
		cc.chainView, err = chainview.NewBtcdFilteredChainView(*rpcConfig)
		if err != nil {
			srvrLog.Errorf("unable to create chain view: %v", err)
			return nil, nil, err
		}

		// Create a special websockets rpc client for btcd which will be used
		// by the wallet for notifications, calls, etc.
		chainRPC, err := chain.NewRPCClient(activeNetParams.Params, btcdHost,
			btcdUser, btcdPass, rpcCert, false, 20)
		if err != nil {
			return nil, nil, err
		}

		walletConfig.ChainSource = chainRPC
		btcdConn = chainRPC

		// If we're not in simnet or regtest mode, then we'll attempt
		// to use a proper fee estimator for testnet.
		if !cfg.Bitcoin.SimNet && !cfg.Monacoin.SimNet &&
			!cfg.Bitcoin.RegTest && !cfg.Monacoin.RegTest {

			ltndLog.Infof("Initializing btcd backed fee estimator")

			// Finally, we'll re-initialize the fee estimator, as
			// if we're using btcd as a backend, then we can use
			// live fee estimates, rather than a statically coded
			// value.
			fallBackFeeRate := lnwallet.SatPerVByte(25)
			cc.feeEstimator, err = lnwallet.NewBtcdFeeEstimator(
				*rpcConfig, fallBackFeeRate,
			)
			if err != nil {
				return nil, nil, err
			}
			if err := cc.feeEstimator.Start(); err != nil {
				return nil, nil, err
			}
		}
	default:
		return nil, nil, fmt.Errorf("unknown node type: %s",
			homeChainConfig.Node)
	}

	wc, err := btcwallet.New(*walletConfig)
	if err != nil {
		fmt.Printf("unable to create wallet controller: %v\n", err)
		return nil, nil, err
	}

	cc.msgSigner = wc
	cc.signer = wc
	cc.chainIO = wc

	// Select the default channel constraints for the primary chain.
	channelConstraints := defaultBtcChannelConstraints
	if registeredChains.PrimaryChain() == monacoinChain {
		channelConstraints = defaultMonaChannelConstraints
	}

	keyRing := keychain.NewBtcWalletKeyRing(
		wc.InternalWallet(), activeNetParams.CoinType,
	)

	// Create, and start the lnwallet, which handles the core payment
	// channel logic, and exposes control via proxy state machines.
	walletCfg := lnwallet.Config{
		Database:           chanDB,
		Notifier:           cc.chainNotifier,
		WalletController:   wc,
		Signer:             cc.signer,
		FeeEstimator:       cc.feeEstimator,
		SecretKeyRing:      keyRing,
		ChainIO:            cc.chainIO,
		DefaultConstraints: channelConstraints,
		NetParams:          *activeNetParams.Params,
	}
	wallet, err := lnwallet.NewLightningWallet(walletCfg)
	if err != nil {
		fmt.Printf("unable to create wallet: %v\n", err)
		return nil, nil, err
	}
	if err := wallet.Startup(); err != nil {
		fmt.Printf("unable to start wallet: %v\n", err)
		return nil, nil, err
	}

	ltndLog.Info("LightningWallet opened")

	cc.wallet = wallet

	// As a final check, if we're using the RPC backend, we'll ensure that
	// the btcd node has the txindex set. Atm, this is required in order to
	// properly perform historical confirmation+spend dispatches.
	if homeChainConfig.Node != "neutrino" {
		// In order to check to see if we have the txindex up to date
		// and active, we'll try to fetch the first transaction in the
		// latest block via the index. If this doesn't succeed, then we
		// know it isn't active (or just not yet up to date).
		bestHash, _, err := cc.chainIO.GetBestBlock()
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get current "+
				"best hash: %v", err)
		}
		bestBlock, err := cc.chainIO.GetBlock(bestHash)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get current "+
				"block hash: %v", err)
		}

		firstTxHash := bestBlock.Transactions[0].TxHash()
		switch homeChainConfig.Node {
		case "btcd":
			_, err = btcdConn.GetRawTransaction(&firstTxHash)
		case "bitcoind":
			_, err = bitcoindConn.GetRawTransactionVerbose(&firstTxHash)
		}
		if err != nil {
			// If the node doesn't have the txindex set, then we'll
			// halt startup, as we can't proceed in this state.
			return nil, nil, fmt.Errorf("%s detected to not "+
				"have --txindex active, cannot proceed",
				homeChainConfig.Node)
		}
	}

	return cc, cleanUp, nil
}

var (
	// bitcoinTestnetGenesis is the genesis hash of Bitcoin's testnet
	// chain.
	bitcoinTestnetGenesis = chainhash.Hash([chainhash.HashSize]byte{
		0x43, 0x49, 0x7f, 0xd7, 0xf8, 0x26, 0x95, 0x71,
		0x08, 0xf4, 0xa3, 0x0f, 0xd9, 0xce, 0xc3, 0xae,
		0xba, 0x79, 0x97, 0x20, 0x84, 0xe9, 0x0e, 0xad,
		0x01, 0xea, 0x33, 0x09, 0x00, 0x00, 0x00, 0x00,
	})

	// bitcoinMainnetGenesis is the genesis hash of Bitcoin's main chain.
	bitcoinMainnetGenesis = chainhash.Hash([chainhash.HashSize]byte{
		0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
		0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
		0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
		0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00,
	})

	// monacoinTestnetGenesis is the genesis hash of Monacoin's testnet4
	// chain.
	monacoinTestnetGenesis = chainhash.Hash([chainhash.HashSize]byte{
		0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59,
		0xca, 0xaf, 0x12, 0x60, 0x43, 0xeb, 0x5b, 0xbf,
		0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f,
		0xc7, 0xb2, 0xb7, 0x3c, 0xf1, 0x88, 0x91, 0x0f,
	})

	// monacoinMainnetGenesis is the genesis hash of Monacoin's main chain.
	monacoinMainnetGenesis = chainhash.Hash([chainhash.HashSize]byte{
		0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
		0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
		0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
		0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00,
	})

	// chainMap is a simple index that maps a chain's genesis hash to the
	// chainCode enum for that chain.
	chainMap = map[chainhash.Hash]chainCode{
		bitcoinTestnetGenesis:  bitcoinChain,
		monacoinTestnetGenesis: monacoinChain,

		bitcoinMainnetGenesis:  bitcoinChain,
		monacoinMainnetGenesis: monacoinChain,
	}

	// chainDNSSeeds is a map of a chain's hash to the set of DNS seeds
	// that will be use to bootstrap peers upon first startup.
	//
	// The first item in the array is the primary host we'll use to attempt
	// the SRV lookup we require. If we're unable to receive a response
	// over UDP, then we'll fall back to manual TCP resolution. The second
	// item in the array is a special A record that we'll query in order to
	// receive the IP address of the current authoritative DNS server for
	// the network seed.
	//
	// TODO(roasbeef): extend and collapse these and chainparams.go into
	// struct like chaincfg.Params
	chainDNSSeeds = map[chainhash.Hash][][2]string{
		bitcoinMainnetGenesis: {
			{
				"nodes.lightning.directory",
				"soa.nodes.lightning.directory",
			},
		},

		bitcoinTestnetGenesis: {
			{
				"test.nodes.lightning.directory",
				"soa.nodes.lightning.directory",
			},
		},

		monacoinMainnetGenesis: {
			{
				"ns1.lndnode.tamami-foundation.org",
				"lndnode.tamami-foundation.org",
			},
		},
	}
)

// chainRegistry keeps track of the current chains
type chainRegistry struct {
	sync.RWMutex

	activeChains map[chainCode]*chainControl
	netParams    map[chainCode]*bitcoinNetParams

	primaryChain chainCode
}

// newChainRegistry creates a new chainRegistry.
func newChainRegistry() *chainRegistry {
	return &chainRegistry{
		activeChains: make(map[chainCode]*chainControl),
		netParams:    make(map[chainCode]*bitcoinNetParams),
	}
}

// RegisterChain assigns an active chainControl instance to a target chain
// identified by its chainCode.
func (c *chainRegistry) RegisterChain(newChain chainCode, cc *chainControl) {
	c.Lock()
	c.activeChains[newChain] = cc
	c.Unlock()
}

// LookupChain attempts to lookup an active chainControl instance for the
// target chain.
func (c *chainRegistry) LookupChain(targetChain chainCode) (*chainControl, bool) {
	c.RLock()
	cc, ok := c.activeChains[targetChain]
	c.RUnlock()
	return cc, ok
}

// LookupChainByHash attempts to look up an active chainControl which
// corresponds to the passed genesis hash.
func (c *chainRegistry) LookupChainByHash(chainHash chainhash.Hash) (*chainControl, bool) {
	c.RLock()
	defer c.RUnlock()

	targetChain, ok := chainMap[chainHash]
	if !ok {
		return nil, ok
	}

	cc, ok := c.activeChains[targetChain]
	return cc, ok
}

// RegisterPrimaryChain sets a target chain as the "home chain" for lnd.
func (c *chainRegistry) RegisterPrimaryChain(cc chainCode) {
	c.Lock()
	defer c.Unlock()

	c.primaryChain = cc
}

// PrimaryChain returns the primary chain for this running lnd instance. The
// primary chain is considered the "home base" while the other registered
// chains are treated as secondary chains.
func (c *chainRegistry) PrimaryChain() chainCode {
	c.RLock()
	defer c.RUnlock()

	return c.primaryChain
}

// ActiveChains returns the total number of active chains.
func (c *chainRegistry) ActiveChains() []chainCode {
	c.RLock()
	defer c.RUnlock()

	chains := make([]chainCode, 0, len(c.activeChains))
	for activeChain := range c.activeChains {
		chains = append(chains, activeChain)
	}

	return chains
}

// NumActiveChains returns the total number of active chains.
func (c *chainRegistry) NumActiveChains() uint32 {
	c.RLock()
	defer c.RUnlock()

	return uint32(len(c.activeChains))
}
