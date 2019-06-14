package lnwallet

import (
	"github.com/btcsuite/btcutil"
	"github.com/monaarchives/btcwallet/wallet/txrules"
	"github.com/monasuite/lnd/input"
)

// DefaultDustLimit is used to calculate the dust HTLC amount which will be
// send to other node during funding process.
func DefaultDustLimit() btcutil.Amount {
	return txrules.GetDustThreshold(input.P2WSHSize, txrules.DefaultRelayFeePerKb)
}
