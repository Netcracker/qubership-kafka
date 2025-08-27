package main

import (
	"context"
	"flag"
	"github.com/Netcracker/qubership-kafka/operator/cfg"
	"github.com/Netcracker/qubership-kafka/operator/workers"
	"os"
	"os/signal"
	"syscall"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/jessevdk/go-flags"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var setupLog = ctrl.Log.WithName("setup")

func main() {
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var appOpts cfg.Cfg
	if _, err := flags.Parse(&appOpts); err != nil {
		setupLog.Error(err, "unable to parse config")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	pool := workers.NewPool(ctx, appOpts, setupLog)

	if err := pool.Start(); err != nil {
		setupLog.Error(err, "failed to start worker pool")
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = pool.Wait()
	}()

	select {
	case <-ctx.Done():
		setupLog.Info("signal received; shutting down...")
	case <-done:
		setupLog.Info("all workers finished")
	}

	<-done
	setupLog.Info("operator exited")
}
