// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"github.com/Netcracker/qubership-kafka/cfg"
	"github.com/Netcracker/qubership-kafka/workers"
	"os"
	"os/signal"
	"syscall"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/jessevdk/go-flags"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var appOpts cfg.Cfg
	_, err := flags.Parse(&appOpts)
	if err != nil {
		setupLog.Error(err, "unable to parse config")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	pool := workers.NewPool(ctx, appOpts, setupLog)
	go func() {
		err = pool.Start()
		if err != nil {
			setupLog.Error(err, "failed to start worker pool")
			stop()
			return
		}
	}()

	<-ctx.Done()
	err = pool.Wait()
	if err != nil {
		setupLog.Error(err, "worker pool encountered an error with waiting jobs")
	}
	setupLog.Info("shutting down operator")
}
