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
	"flag"
	"fmt"
	"github.com/Netcracker/qubership-kafka/controllers/kafkauser"
	"os"
	"strconv"
	"strings"

	"github.com/Netcracker/qubership-kafka/controllers"

	"github.com/Netcracker/qubership-kafka/controllers/akhqconfig"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/Netcracker/qubership-kafka/controllers/kafka"
	"github.com/Netcracker/qubership-kafka/controllers/kafkaservice"
	"github.com/Netcracker/qubership-kafka/controllers/kmmconfig"
	"github.com/Netcracker/qubership-kafka/util"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	qubershiporgv1 "github.com/Netcracker/qubership-kafka/api/v1"
	qubershiporgv7 "github.com/Netcracker/qubership-kafka/api/v7"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	watchAkhqCollecNamespaceEnv                         = "WATCH_AKHQ_COLLECT_NAMESPACE"
	kmmEnabledEnv                                       = "KMM_ENABLED"
	watchKafkaUsersCollectNamespaceEnv                  = "WATCH_KAFKA_USERS_COLLECT_NAMESPACE"
	kafkaUserSecretCreatingEnabledEnv                   = "KAFKA_USER_SECRET_CREATING_ENABLED"
	kafkaUserConfiguratorReconciliationPeriodSecondsEnv = "KAFKA_USER_CONFIGURATOR_RECONCILE_PERIOD_SECONDS"
	kafkaBootstrapServersEnv                            = "BOOTSTRAP_SERVERS"
	kafkaSecretEnv                                      = "KAFKA_SECRET"
	kafkaSaslMechanismEnv                               = "KAFKA_SASL_MECHANISM"
	kafkaSslEnabledEnv                                  = "KAFKA_SSL_ENABLED"
	kafkaSslSecretEnv                                   = "KAFKA_SSL_SECRET"
)

func logKmmConfigMgrEndGoroutine() {
	setupLog.Info("KmmConfig manager goroutine has been finished")
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(qubershiporgv1.AddToScheme(scheme))
	utilruntime.Must(qubershiporgv7.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8082", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ownNamespace := os.Getenv("OPERATOR_NAMESPACE")
	mode := os.Getenv("OPERATOR_MODE")
	apiGroup := os.Getenv("API_GROUP")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		Namespace:               ownNamespace,
		MetricsBindAddress:      metricsAddr,
		Port:                    9443,
		HealthProbeBindAddress:  probeAddr,
		LeaderElection:          enableLeaderElection,
		LeaderElectionNamespace: ownNamespace,
		LeaderElectionID:        fmt.Sprintf("%s.%s.%s", mode, ownNamespace, apiGroup),
	})
	if err != nil {
		setupLog.Error(err, fmt.Sprintf("unable to start %s manager", mode))
		os.Exit(1)
	}
	ssh := ctrl.SetupSignalHandler()
	if mode == "kafka" {
		if err = (&kafka.KafkaReconciler{
			Reconciler: controllers.Reconciler{
				Client:           mgr.GetClient(),
				Scheme:           mgr.GetScheme(),
				ResourceVersions: map[string]string{},
				ResourceHashes:   map[string]string{},
			},
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Kafka")
			os.Exit(1)
		}
	} else {
		if err = (&kafkaservice.KafkaServiceReconciler{
			Reconciler: controllers.Reconciler{
				Client:           mgr.GetClient(),
				Scheme:           mgr.GetScheme(),
				ResourceVersions: map[string]string{},
				ResourceHashes:   map[string]string{},
			},
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "KafkaService")
			os.Exit(1)
		}

		kmmEnabled, err := strconv.ParseBool(os.Getenv(kmmEnabledEnv))
		if err != nil {
			setupLog.Error(err, "unable to start kmm manager")
			kmmEnabled = false
		}
		if kmmEnabled {
			kmmMgr := createKmmConfigMgr(enableLeaderElection, ownNamespace, apiGroup)
			go func() {
				defer logKmmConfigMgrEndGoroutine()
				setupLog.Info("starting KmmConfig manager")
				if err := kmmMgr.Start(ssh); err != nil {
					setupLog.Error(err, "problem running KmmConfig manager")
					os.Exit(1)
				}
			}()
		}

		if akhqNamespaces, ok := os.LookupEnv(watchAkhqCollecNamespaceEnv); ok {
			akhqMgr := createAkhqConfigMgr(akhqNamespaces, ownNamespace, enableLeaderElection, apiGroup)
			go func() {
				defer func() {
					setupLog.Info("AkhqConfig manager goroutine has been finished")
				}()
				setupLog.Info("starting AkhqConfig manager")
				if err := akhqMgr.Start(ssh); err != nil {
					setupLog.Error(err, "problem running AkhqConfig manager")
					os.Exit(1)
				}
			}()
		}

		if kafkaUsersNamespaces, ok := os.LookupEnv(watchKafkaUsersCollectNamespaceEnv); ok {
			kafkaUserMgr := createKafkaUsersMgr(kafkaUsersNamespaces, ownNamespace, enableLeaderElection, apiGroup)
			go func() {
				defer func() {
					setupLog.Info("KafkaUser manager goroutine has been finished")
				}()
				setupLog.Info("starting KafkaUser manager")
				if err := kafkaUserMgr.Start(ssh); err != nil {
					setupLog.Error(err, "problem running KafkaUser manager")
					os.Exit(1)
				}
			}()
		}
	}

	//+kubebuilder:scaffold:builder

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Data is cleared after an error occurs or a stop signal is received by the controller manager.
	defer kafka.CleanData(setupLog)

	setupLog.Info(fmt.Sprintf("starting %s manager", mode))
	if err := mgr.Start(ssh); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func createKmmConfigMgr(enableLeaderElection bool, ownNamespace string, apiGroup string) manager.Manager {
	namespace, err := getWatchNamespace()
	if err != nil {
		setupLog.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	kmmMgrOptions := ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      "0",
		Port:                    9543,
		HealthProbeBindAddress:  "0",
		LeaderElection:          enableLeaderElection,
		LeaderElectionNamespace: ownNamespace,
		LeaderElectionID:        fmt.Sprintf("kmmconfig.%s.%s", ownNamespace, apiGroup),
	}
	configureManagerNamespaces(&kmmMgrOptions, namespace, ownNamespace)

	kmmMgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), kmmMgrOptions)
	if err != nil {
		setupLog.Error(err, "unable to start KmmConfig manager")
		os.Exit(1)
	}

	if err = (&kmmconfig.KmmConfigReconciler{
		Client: kmmMgr.GetClient(),
		Scheme: kmmMgr.GetScheme(),
	}).SetupWithManager(kmmMgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KmmConfig")
		os.Exit(1)
	}

	if err := kmmMgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := kmmMgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
	return kmmMgr
}

func createAkhqConfigMgr(namespace string, ownNamespace string, enableLeaderElection bool, apiGroup string) manager.Manager {
	akhqMgrOptions := ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      "0",
		Port:                    9542,
		HealthProbeBindAddress:  "0",
		LeaderElection:          enableLeaderElection,
		LeaderElectionNamespace: ownNamespace,
		LeaderElectionID:        fmt.Sprintf("akhqconfig.%s.%s", ownNamespace, apiGroup),
	}
	configureManagerNamespaces(&akhqMgrOptions, namespace, ownNamespace)

	akhqMgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), akhqMgrOptions)
	if err != nil {
		setupLog.Error(err, "unable to start AkhqConfig manager")
		os.Exit(1)
	}

	if err = (&akhqconfig.AkhqConfigReconciler{
		Client:    akhqMgr.GetClient(),
		Scheme:    akhqMgr.GetScheme(),
		Namespace: ownNamespace,
	}).SetupWithManager(akhqMgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AkhqConfig")
		os.Exit(1)
	}

	if err := akhqMgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := akhqMgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
	return akhqMgr
}

func createKafkaUsersMgr(namespace string, ownNamespace string, enableLeaderElection bool, apiGroup string) manager.Manager {
	kafkaUsersMgrOptions := ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      "0",
		Port:                    9544,
		HealthProbeBindAddress:  "0",
		LeaderElection:          enableLeaderElection,
		LeaderElectionNamespace: ownNamespace,
		LeaderElectionID:        fmt.Sprintf("kafkausers.%s.%s", ownNamespace, apiGroup),
	}
	configureManagerNamespaces(&kafkaUsersMgrOptions, namespace, ownNamespace)

	kafkaUserMgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), kafkaUsersMgrOptions)
	if err != nil {
		setupLog.Error(err, "unable to start Kafka Users manager")
		os.Exit(1)
	}

	reconciliationPeriod, err := strconv.Atoi(os.Getenv(kafkaUserConfiguratorReconciliationPeriodSecondsEnv))
	if err != nil {
		setupLog.Error(err, "unable to start Connector manager")
		os.Exit(1)
	}

	secretCreatingEnabled, err := strconv.ParseBool(os.Getenv(kafkaUserSecretCreatingEnabledEnv))
	if err != nil {
		setupLog.Error(err, "unable to start Connector manager")
		os.Exit(1)
	}
	kafkaSslEnabled, err := strconv.ParseBool(os.Getenv(kafkaSslEnabledEnv))
	if err != nil {
		setupLog.Error(err, "unable to start Connector manager")
		os.Exit(1)
	}
	if err = (&kafkauser.KafkaUserReconciler{
		BootstrapServers:      os.Getenv(kafkaBootstrapServersEnv),
		Client:                kafkaUserMgr.GetClient(),
		SecretCreatingEnabled: secretCreatingEnabled,
		Namespace:             ownNamespace,
		ReconciliationPeriod:  reconciliationPeriod,
		Scheme:                kafkaUserMgr.GetScheme(),
		ResourceHashes:        map[string]string{},
		KafkaSecret:           os.Getenv(kafkaSecretEnv),
		KafkaSaslMechanism:    os.Getenv(kafkaSaslMechanismEnv),
		KafkaSslEnabled:       kafkaSslEnabled,
		KafkaSslSecret:        os.Getenv(kafkaSslSecretEnv),
	}).SetupWithManager(kafkaUserMgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KafkaUsers")
		os.Exit(1)
	}

	if err := kafkaUserMgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := kafkaUserMgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
	return kafkaUserMgr
}

// getWatchNamespace returns the Namespace the operator should be watching for changes
func getWatchNamespace() (string, error) {
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	var watchNamespaceEnvVar = "WATCH_NAMESPACE"

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

func configureManagerNamespaces(configMgrOptions *ctrl.Options, namespace string, ownNamespace string) {
	if namespace == "" || namespace == ownNamespace {
		configMgrOptions.Namespace = namespace
	} else {
		namespaces := strings.Split(namespace, ",")
		if !util.Contains(ownNamespace, namespaces) {
			namespaces = append(namespaces, ownNamespace)
		}
		configMgrOptions.NewCache = cache.MultiNamespacedCacheBuilder(namespaces)
	}
}
