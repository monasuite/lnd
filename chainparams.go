package lnd

import (
	"github.com/btcsuite/btcd/chaincfg"
	bitcoinCfg "github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	bitcoinWire "github.com/btcsuite/btcd/wire"
	"github.com/monasuite/lnd/keychain"
	monacoinCfg "github.com/monasuite/monad/chaincfg"
	monacoinWire "github.com/monasuite/monad/wire"
)

// activeNetParams is a pointer to the parameters specific to the currently
// active bitcoin network.
var activeNetParams = bitcoinTestNetParams

// bitcoinNetParams couples the p2p parameters of a network with the
// corresponding RPC port of a daemon running on the particular network.
type bitcoinNetParams struct {
	*bitcoinCfg.Params
	rpcPort  string
	CoinType uint32
}

// monacoinNetParams couples the p2p parameters of a network with the
// corresponding RPC port of a daemon running on the particular network.
type monacoinNetParams struct {
	*monacoinCfg.Params
	rpcPort  string
	CoinType uint32
}

// bitcoinTestNetParams contains parameters specific to the 3rd version of the
// test network.
var bitcoinTestNetParams = bitcoinNetParams{
	Params:   &bitcoinCfg.TestNet3Params,
	rpcPort:  "18334",
	CoinType: keychain.CoinTypeTestnet,
}

// bitcoinMainNetParams contains parameters specific to the current Bitcoin
// mainnet.
var bitcoinMainNetParams = bitcoinNetParams{
	Params:   &bitcoinCfg.MainNetParams,
	rpcPort:  "8334",
	CoinType: keychain.CoinTypeBitcoin,
}

// bitcoinSimNetParams contains parameters specific to the simulation test
// network.
var bitcoinSimNetParams = bitcoinNetParams{
	Params:   &bitcoinCfg.SimNetParams,
	rpcPort:  "18556",
	CoinType: keychain.CoinTypeTestnet,
}

// monacoinSimNetParams contains parameters specific to the simulation test
// network.
var monacoinSimNetParams = monacoinNetParams{
	Params:   &monacoinCfg.SimNetParams,
	rpcPort:  "18556",
	CoinType: keychain.CoinTypeTestnet,
}

// monacoinTestNetParams contains parameters specific to the 4th version of the
// test network.
var monacoinTestNetParams = monacoinNetParams{
	Params:   &monacoinCfg.TestNet4Params,
	rpcPort:  "19400",
	CoinType: keychain.CoinTypeTestnet,
}

// monacoinMainNetParams contains the parameters specific to the current
// Monacoin mainnet.
var monacoinMainNetParams = monacoinNetParams{
	Params:   &monacoinCfg.MainNetParams,
	rpcPort:  "9400",
	CoinType: keychain.CoinTypeMonacoin,
}

// monacoinRegTestNetParams contains parameters specific to a local monacoin
// regtest network.
var monacoinRegTestNetParams = monacoinNetParams{
	Params:   &monacoinCfg.RegressionNetParams,
	rpcPort:  "18334",
	CoinType: keychain.CoinTypeTestnet,
}

// bitcoinRegTestNetParams contains parameters specific to a local bitcoin
// regtest network.
var bitcoinRegTestNetParams = bitcoinNetParams{
	Params:   &bitcoinCfg.RegressionNetParams,
	rpcPort:  "18334",
	CoinType: keychain.CoinTypeTestnet,
}

// applyMonacoinParams applies the relevant chain configuration parameters that
// differ for monacoin to the chain parameters typed for btcsuite derivation.
// This function is used in place of using something like interface{} to
// abstract over _which_ chain (or fork) the parameters are for.
func applyMonacoinParams(params *bitcoinNetParams, monacoinParams *monacoinNetParams) {
	params.Name = monacoinParams.Name
	params.Net = bitcoinWire.BitcoinNet(monacoinParams.Net)
	params.DefaultPort = monacoinParams.DefaultPort
	params.CoinbaseMaturity = monacoinParams.CoinbaseMaturity

	copy(params.GenesisHash[:], monacoinParams.GenesisHash[:])

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
	params.rpcPort = monacoinParams.rpcPort
	params.CoinType = monacoinParams.CoinType
}

// isTestnet tests if the given params correspond to a testnet
// parameter configuration.
func isTestnet(params *bitcoinNetParams) bool {
	switch params.Params.Net {
	case bitcoinWire.TestNet3, bitcoinWire.BitcoinNet(monacoinWire.TestNet4):
		return true
	default:
		return false
	}
}
