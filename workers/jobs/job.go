package jobs

import (
	"context"
	"fmt"
	qubershiporgv1 "github.com/Netcracker/qubership-kafka/api/v1"
	qubershiporgv7 "github.com/Netcracker/qubership-kafka/api/v7"
	"github.com/Netcracker/qubership-kafka/cfg"
	"github.com/Netcracker/qubership-kafka/util"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	sigsScheme "sigs.k8s.io/controller-runtime/pkg/scheme"
	"strconv"
	"strings"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(qubershiporgv1.AddToScheme(scheme))
	utilruntime.Must(qubershiporgv7.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

type Exec func() error

type Job interface {
	Build(ctx context.Context, opts cfg.Cfg, apiGroup string, logger logr.Logger) (Exec, error)
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

func duplicateAddr(addr string) (string, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return fmt.Sprintf("%s:%d", addr, 8081), nil
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", parts[0], port+10), nil
}

func mainApiGroup() string {
	if value, ok := os.LookupEnv("API_GROUP"); ok {
		return value
	}
	return "qubership.org"
}

func duplicateScheme(apiGroup string) (*runtime.Scheme, error) {
	dblScheme := runtime.NewScheme()
	err := clientgoscheme.AddToScheme(dblScheme)
	if err != nil {
		return nil, err
	}
	additionalGroupVersion := schema.GroupVersion{Group: apiGroup, Version: "v1"}
	additionalSchemeBuilder := &sigsScheme.Builder{GroupVersion: additionalGroupVersion}
	additionalSchemeBuilder.Register(&qubershiporgv1.AkhqConfig{}, &qubershiporgv1.AkhqConfigList{})
	additionalSchemeBuilder.Register(&qubershiporgv1.Kafka{}, &qubershiporgv1.KafkaList{})
	additionalSchemeBuilder.Register(&qubershiporgv1.KafkaUser{}, &qubershiporgv1.KafkaUserList{})
	additionalSchemeBuilder.Register(&qubershiporgv1.KmmConfig{}, &qubershiporgv1.KmmConfigList{})
	err = additionalSchemeBuilder.AddToScheme(dblScheme)
	if err != nil {
		return nil, err
	}
	secondaryGroupVersionV7 := schema.GroupVersion{Group: apiGroup, Version: "7"}
	secondarySchemeBuilderV7 := &sigsScheme.Builder{GroupVersion: secondaryGroupVersionV7}
	secondarySchemeBuilderV7.Register(&qubershiporgv7.KafkaService{}, &qubershiporgv7.KafkaServiceList{})
	err = secondarySchemeBuilderV7.AddToScheme(dblScheme)
	return dblScheme, err
}
