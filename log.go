package lnd

import (
	"context"

	"github.com/btcsuite/btcd/connmgr"
	"github.com/btcsuite/btclog"
	sphinx "github.com/lightningnetwork/lightning-onion"
	"github.com/monasuite/lnd/autopilot"
	"github.com/monasuite/lnd/build"
	"github.com/monasuite/lnd/chainntnfs"
	"github.com/monasuite/lnd/chanbackup"
	"github.com/monasuite/lnd/chanfitness"
	"github.com/monasuite/lnd/channeldb"
	"github.com/monasuite/lnd/channelnotifier"
	"github.com/monasuite/lnd/contractcourt"
	"github.com/monasuite/lnd/discovery"
	"github.com/monasuite/lnd/htlcswitch"
	"github.com/monasuite/lnd/invoices"
	"github.com/monasuite/lnd/lnrpc/autopilotrpc"
	"github.com/monasuite/lnd/lnrpc/chainrpc"
	"github.com/monasuite/lnd/lnrpc/invoicesrpc"
	"github.com/monasuite/lnd/lnrpc/routerrpc"
	"github.com/monasuite/lnd/lnrpc/signrpc"
	"github.com/monasuite/lnd/lnrpc/verrpc"
	"github.com/monasuite/lnd/lnrpc/walletrpc"
	"github.com/monasuite/lnd/lnwallet"
	"github.com/monasuite/lnd/lnwallet/chanfunding"
	"github.com/monasuite/lnd/monitoring"
	"github.com/monasuite/lnd/netann"
	"github.com/monasuite/lnd/peernotifier"
	"github.com/monasuite/lnd/routing"
	"github.com/monasuite/lnd/routing/localchans"
	"github.com/monasuite/lnd/signal"
	"github.com/monasuite/lnd/sweep"
	"github.com/monasuite/lnd/watchtower"
	"github.com/monasuite/lnd/watchtower/wtclient"
	"github.com/monasuite/neutrino"
	"google.golang.org/grpc"
)

// Loggers per subsystem.  A single backend logger is created and all subsystem
// loggers created from it will write to the backend.  When adding new
// subsystems, add the subsystem logger variable here and to the
// subsystemLoggers map.
//
// Loggers can not be used before the log rotator has been initialized with a
// log file.  This must be performed early during application startup by
// calling logWriter.InitLogRotator.
var (
	logWriter = build.NewRotatingLogWriter()

	// Loggers that need to be accessible from the lnd package can be placed
	// here. Loggers that are only used in sub modules can be added directly
	// by using the addSubLogger method.
	ltndLog = build.NewSubLogger("LTND", logWriter.GenSubLogger)
	peerLog = build.NewSubLogger("PEER", logWriter.GenSubLogger)
	rpcsLog = build.NewSubLogger("RPCS", logWriter.GenSubLogger)
	srvrLog = build.NewSubLogger("SRVR", logWriter.GenSubLogger)
	fndgLog = build.NewSubLogger("FNDG", logWriter.GenSubLogger)
	utxnLog = build.NewSubLogger("UTXN", logWriter.GenSubLogger)
	brarLog = build.NewSubLogger("BRAR", logWriter.GenSubLogger)
	atplLog = build.NewSubLogger("ATPL", logWriter.GenSubLogger)
)

// Initialize package-global logger variables.
func init() {
	setSubLogger("LTND", ltndLog, signal.UseLogger)
	setSubLogger("ATPL", atplLog, autopilot.UseLogger)
	setSubLogger("PEER", peerLog)
	setSubLogger("RPCS", rpcsLog)
	setSubLogger("SRVR", srvrLog)
	setSubLogger("FNDG", fndgLog)
	setSubLogger("UTXN", utxnLog)
	setSubLogger("BRAR", brarLog)

	addSubLogger("LNWL", lnwallet.UseLogger)
	addSubLogger("DISC", discovery.UseLogger)
	addSubLogger("NTFN", chainntnfs.UseLogger)
	addSubLogger("CHDB", channeldb.UseLogger)
	addSubLogger("HSWC", htlcswitch.UseLogger)
	addSubLogger("CMGR", connmgr.UseLogger)
	addSubLogger("BTCN", neutrino.UseLogger)
	addSubLogger("CNCT", contractcourt.UseLogger)
	addSubLogger("SPHX", sphinx.UseLogger)
	addSubLogger("SWPR", sweep.UseLogger)
	addSubLogger("SGNR", signrpc.UseLogger)
	addSubLogger("WLKT", walletrpc.UseLogger)
	addSubLogger("ARPC", autopilotrpc.UseLogger)
	addSubLogger("INVC", invoices.UseLogger)
	addSubLogger("NANN", netann.UseLogger)
	addSubLogger("WTWR", watchtower.UseLogger)
	addSubLogger("NTFR", chainrpc.UseLogger)
	addSubLogger("IRPC", invoicesrpc.UseLogger)
	addSubLogger("CHNF", channelnotifier.UseLogger)
	addSubLogger("CHBU", chanbackup.UseLogger)
	addSubLogger("PROM", monitoring.UseLogger)
	addSubLogger("WTCL", wtclient.UseLogger)
	addSubLogger("PRNF", peernotifier.UseLogger)
	addSubLogger("CHFD", chanfunding.UseLogger)

	addSubLogger(routing.Subsystem, routing.UseLogger, localchans.UseLogger)
	addSubLogger(routerrpc.Subsystem, routerrpc.UseLogger)
	addSubLogger(chanfitness.Subsystem, chanfitness.UseLogger)
	addSubLogger(verrpc.Subsystem, verrpc.UseLogger)
}

// addSubLogger is a helper method to conveniently create and register the
// logger of one or more sub systems.
func addSubLogger(subsystem string, useLoggers ...func(btclog.Logger)) {
	// Create and register just a single logger to prevent them from
	// overwriting each other internally.
	logger := build.NewSubLogger(subsystem, logWriter.GenSubLogger)
	setSubLogger(subsystem, logger, useLoggers...)
}

// setSubLogger is a helper method to conveniently register the logger of a sub
// system.
func setSubLogger(subsystem string, logger btclog.Logger,
	useLoggers ...func(btclog.Logger)) {

	logWriter.RegisterSubLogger(subsystem, logger)
	for _, useLogger := range useLoggers {
		useLogger(logger)
	}
}

// logClosure is used to provide a closure over expensive logging operations so
// don't have to be performed when the logging level doesn't warrant it.
type logClosure func() string

// String invokes the underlying function and returns the result.
func (c logClosure) String() string {
	return c()
}

// newLogClosure returns a new closure over a function that returns a string
// which itself provides a Stringer interface so that it can be used with the
// logging system.
func newLogClosure(c func() string) logClosure {
	return logClosure(c)
}

// errorLogUnaryServerInterceptor is a simple UnaryServerInterceptor that will
// automatically log any errors that occur when serving a client's unary
// request.
func errorLogUnaryServerInterceptor(logger btclog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {

		resp, err := handler(ctx, req)
		if err != nil {
			// TODO(roasbeef): also log request details?
			logger.Errorf("[%v]: %v", info.FullMethod, err)
		}

		return resp, err
	}
}

// errorLogStreamServerInterceptor is a simple StreamServerInterceptor that
// will log any errors that occur while processing a client or server streaming
// RPC.
func errorLogStreamServerInterceptor(logger btclog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream,
		info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

		err := handler(srv, ss)
		if err != nil {
			logger.Errorf("[%v]: %v", info.FullMethod, err)
		}

		return err
	}
}
