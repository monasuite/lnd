package chainreg

import (
	"github.com/btcsuite/btcd/chaincfg"
	bitcoinCfg "github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	bitcoinWire "github.com/btcsuite/btcd/wire"
	"github.com/monasuite/lnd/keychain"
	monacoinCfg "github.com/monasuite/monad/chaincfg"
	monacoinWire "github.com/monasuite/monad/wire"
)

// BitcoinNetParams couples the p2p parameters of a network with the
// corresponding RPC port of a daemon running on the particular network.
type BitcoinNetParams struct {
	*bitcoinCfg.Params
	RPCPort  string
	CoinType uint32
}

// MonacoinNetParams couples the p2p parameters of a network with the
// corresponding RPC port of a daemon running on the particular network.
type MonacoinNetParams struct {
	*monacoinCfg.Params
	RPCPort  string
	CoinType uint32
}

// BitcoinTestNetParams contains parameters specific to the 3rd version of the
// test network.
var BitcoinTestNetParams = BitcoinNetParams{
	Params:   &bitcoinCfg.TestNet3Params,
	RPCPort:  "18334",
	CoinType: keychain.CoinTypeTestnet,
}

// BitcoinMainNetParams contains parameters specific to the current Bitcoin
// mainnet.
var BitcoinMainNetParams = BitcoinNetParams{
	Params:   &bitcoinCfg.MainNetParams,
	RPCPort:  "8334",
	CoinType: keychain.CoinTypeBitcoin,
}

// BitcoinSimNetParams contains parameters specific to the simulation test
// network.
var BitcoinSimNetParams = BitcoinNetParams{
	Params:   &bitcoinCfg.SimNetParams,
	RPCPort:  "18556",
	CoinType: keychain.CoinTypeTestnet,
}

// MonacoinSimNetParams contains parameters specific to the simulation test
// network.
var MonacoinSimNetParams = MonacoinNetParams{
	Params:   &monacoinCfg.SimNetParams,
	RPCPort:  "18556",
	CoinType: keychain.CoinTypeTestnet,
}

// MonacoinTestNetParams contains parameters specific to the 4th version of the
// test network.
var MonacoinTestNetParams = MonacoinNetParams{
	Params:   &monacoinCfg.TestNet4Params,
	RPCPort:  "19400",
	CoinType: keychain.CoinTypeTestnet,
}

// MonacoinMainNetParams contains the parameters specific to the current
// Monacoin mainnet.
var MonacoinMainNetParams = MonacoinNetParams{
	Params:   &monacoinCfg.MainNetParams,
	RPCPort:  "9400",
	CoinType: keychain.CoinTypeMonacoin,
}

// MonacoinRegTestNetParams contains parameters specific to a local monacoin
// regtest network.
var MonacoinRegTestNetParams = MonacoinNetParams{
	Params:   &monacoinCfg.RegressionNetParams,
	RPCPort:  "18334",
	CoinType: keychain.CoinTypeTestnet,
}

// BitcoinRegTestNetParams contains parameters specific to a local bitcoin
// regtest network.
var BitcoinRegTestNetParams = BitcoinNetParams{
	Params:   &bitcoinCfg.RegressionNetParams,
	RPCPort:  "18334",
	CoinType: keychain.CoinTypeTestnet,
}

// applyMonacoinParams applies the relevant chain configuration parameters that
// differ for monacoin to the chain parameters typed for btcsuite derivation.
// This function is used in place of using something like interface{} to
// abstract over _which_ chain (or fork) the parameters are for.
func applyMonacoinParams(params *BitcoinNetParams,
	monacoinParams *MonacoinNetParams) {

	params.Name = monacoinParams.Name
	params.Net = bitcoinWire.BitcoinNet(monacoinParams.Net)
	params.DefaultPort = monacoinParams.DefaultPort
	params.CoinbaseMaturity = monacoinParams.CoinbaseMaturity

	copy(params.GenesisHash[:], monacoinParams.GenesisHash[:])
	copy(params.GenesisBlock.Header.MerkleRoot[:],
		monacoinParams.GenesisBlock.Header.MerkleRoot[:])
	params.GenesisBlock.Header.Version =
		monacoinParams.GenesisBlock.Header.Version
	params.GenesisBlock.Header.Timestamp =
		monacoinParams.GenesisBlock.Header.Timestamp
	params.GenesisBlock.Header.Bits =
		monacoinParams.GenesisBlock.Header.Bits
	params.GenesisBlock.Header.Nonce =
		monacoinParams.GenesisBlock.Header.Nonce
	params.GenesisBlock.Transactions[0].Version =
		monacoinParams.GenesisBlock.Transactions[0].Version
	params.GenesisBlock.Transactions[0].LockTime =
		monacoinParams.GenesisBlock.Transactions[0].LockTime
	params.GenesisBlock.Transactions[0].TxIn[0].Sequence =
		monacoinParams.GenesisBlock.Transactions[0].TxIn[0].Sequence
	params.GenesisBlock.Transactions[0].TxIn[0].PreviousOutPoint.Index =
		monacoinParams.GenesisBlock.Transactions[0].TxIn[0].PreviousOutPoint.Index
	copy(params.GenesisBlock.Transactions[0].TxIn[0].SignatureScript,
		monacoinParams.GenesisBlock.Transactions[0].TxIn[0].SignatureScript)
	copy(params.GenesisBlock.Transactions[0].TxOut[0].PkScript,
		monacoinParams.GenesisBlock.Transactions[0].TxOut[0].PkScript)
	params.GenesisBlock.Transactions[0].TxOut[0].Value =
		monacoinParams.GenesisBlock.Transactions[0].TxOut[0].Value
	params.GenesisBlock.Transactions[0].TxIn[0].PreviousOutPoint.Hash =
		chainhash.Hash{}
	params.PowLimitBits = monacoinParams.PowLimitBits
	params.PowLimit = monacoinParams.PowLimit

	dnsSeeds := make([]chaincfg.DNSSeed, len(monacoinParams.DNSSeeds))
	for i := 0; i < len(monacoinParams.DNSSeeds); i++ {
		dnsSeeds[i] = chaincfg.DNSSeed{
			Host:         monacoinParams.DNSSeeds[i].Host,
			HasFiltering: monacoinParams.DNSSeeds[i].HasFiltering,
		}
	}
	params.DNSSeeds = dnsSeeds

	// Address encoding magics
	params.PubKeyHashAddrID = monacoinParams.PubKeyHashAddrID
	params.ScriptHashAddrID = monacoinParams.ScriptHashAddrID
	params.PrivateKeyID = monacoinParams.PrivateKeyID
	params.WitnessPubKeyHashAddrID = monacoinParams.WitnessPubKeyHashAddrID
	params.WitnessScriptHashAddrID = monacoinParams.WitnessScriptHashAddrID
	params.Bech32HRPSegwit = monacoinParams.Bech32HRPSegwit

	copy(params.HDPrivateKeyID[:], monacoinParams.HDPrivateKeyID[:])
	copy(params.HDPublicKeyID[:], monacoinParams.HDPublicKeyID[:])

	params.HDCoinType = monacoinParams.HDCoinType

	checkPoints := make([]chaincfg.Checkpoint, len(monacoinParams.Checkpoints))
	for i := 0; i < len(monacoinParams.Checkpoints); i++ {
		var chainHash chainhash.Hash
		copy(chainHash[:], monacoinParams.Checkpoints[i].Hash[:])

		checkPoints[i] = chaincfg.Checkpoint{
			Height: monacoinParams.Checkpoints[i].Height,
			Hash:   &chainHash,
		}
	}
	params.Checkpoints = checkPoints

	params.RPCPort = monacoinParams.RPCPort
	params.CoinType = monacoinParams.CoinType
}

// IsTestnet tests if the givern params correspond to a testnet
// parameter configuration.
func IsTestnet(params *BitcoinNetParams) bool {
	switch params.Params.Net {
	case bitcoinWire.TestNet3, bitcoinWire.BitcoinNet(monacoinWire.TestNet4):
		return true
	default:
		return false
	}
}
