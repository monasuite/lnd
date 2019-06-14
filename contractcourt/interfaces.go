package contractcourt

import (
	"github.com/monasuite/lnd/channeldb"
	"github.com/monasuite/lnd/invoices"
	"github.com/monasuite/lnd/lntypes"
	"github.com/monasuite/lnd/lnwire"
)

// Registry is an interface which represents the invoice registry.
type Registry interface {
	// LookupInvoice attempts to look up an invoice according to its 32
	// byte payment hash. This method should also reutrn the min final CLTV
	// delta for this invoice. We'll use this to ensure that the HTLC
	// extended to us gives us enough time to settle as we prescribe.
	LookupInvoice(lntypes.Hash) (channeldb.Invoice, uint32, error)

	// NotifyExitHopHtlc attempts to mark an invoice as settled. If the
	// invoice is a debug invoice, then this method is a noop as debug
	// invoices are never fully settled. The return value describes how the
	// htlc should be resolved. If the htlc cannot be resolved immediately,
	// the resolution is sent on the passed in hodlChan later.
	NotifyExitHopHtlc(payHash lntypes.Hash, paidAmount lnwire.MilliSatoshi,
		expiry uint32, currentHeight int32,
		hodlChan chan<- interface{}) (*invoices.HodlEvent, error)

	// HodlUnsubscribeAll unsubscribes from all hodl events.
	HodlUnsubscribeAll(subscriber chan<- interface{})
}
