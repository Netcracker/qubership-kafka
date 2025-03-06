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

package kafka

import (
	"encoding/json"
	"fmt"
	kafka "github.com/Netcracker/qubership-kafka/api/v1"
	"github.com/Netcracker/qubership-kafka/controllers"
	"github.com/Netcracker/qubership-kafka/controllers/provider"
	"github.com/Netcracker/qubership-kafka/util"
	"github.com/go-logr/logr"
	consulApi "github.com/hashicorp/consul/api"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
	"strings"
	"time"
)

var consulConfigs = make(map[string]*consulApi.Config)
var areExternalAddressesUsed = false

const (
	aclNotFoundError string = "ACL not found"
	consulClientPort int    = 8500
	external         string = "external"
	kafkaPort        int    = 9092
)

type ReconcileKafkaDiscovery struct {
	cr                     *kafka.Kafka
	reconciler             *KafkaReconciler
	logger                 logr.Logger
	kafkaDiscoveryProvider provider.KafkaDiscoveryProvider
}

func NewReconcileKafkaDiscovery(r *KafkaReconciler, cr *kafka.Kafka, logger logr.Logger) ReconcileKafkaDiscovery {
	return ReconcileKafkaDiscovery{
		cr:                     cr,
		logger:                 logger,
		reconciler:             r,
		kafkaDiscoveryProvider: provider.NewKafkaDiscoveryProvider(cr, logger),
	}
}

func (r ReconcileKafkaDiscovery) Reconcile() error {
	discoveryConfigMap, err := r.reconciler.FindConfigMap(provider.DiscoveryConfigurationName, r.cr.Namespace, r.logger)
	if err != nil {
		if discoveryConfigMap, err = r.kafkaDiscoveryProvider.CreateDiscoveryConfigMapDef(); err != nil {
			return err
		}
		if err := controllerutil.SetControllerReference(r.cr, discoveryConfigMap, r.reconciler.Scheme); err != nil {
			return err
		}
		if err := r.reconciler.CreateConfigMapIfNotExist(discoveryConfigMap, r.logger); err != nil {
			return err
		}
	}

	configMapKey := fmt.Sprintf("%s.%s", discoveryConfigMap.Name, discoveryConfigMap.Namespace)
	if r.reconciler.ResourceVersions[configMapKey] != discoveryConfigMap.ResourceVersion {
		r.logger.Info("Discovery config map is updated")
		if err := r.registerAllKafkaInstances(); err != nil {
			return err
		}
		r.reconciler.ResourceVersions[configMapKey] = discoveryConfigMap.ResourceVersion
	} else {
		if err := r.registerKafkaInstanceIfDeploymentIsUpdated(); err != nil {
			return err
		}
	}
	return nil
}

func (r ReconcileKafkaDiscovery) Status() error {
	return nil
}

func (r *ReconcileKafkaDiscovery) registerAllKafkaInstances() error {
	r.logger.Info("Register Kafka service")
	kafkaDeployments, err := r.reconciler.FindKafkaDeployments(r.cr)
	if err != nil {
		return err
	}
	for _, deployment := range kafkaDeployments.Items {
		if err := r.registerKafkaInstance(deployment); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileKafkaDiscovery) registerKafkaInstanceIfDeploymentIsUpdated() error {
	deployments, err := r.reconciler.FindKafkaDeployments(r.cr)
	if err != nil {
		return err
	}
	for _, deployment := range deployments.Items {
		deploymentKey := fmt.Sprintf("%s.%s", deployment.Name, deployment.Namespace)
		if r.reconciler.ResourceVersions[deploymentKey] != deployment.ResourceVersion {
			if err = r.registerKafkaInstance(deployment); err != nil {
				return err
			}
			r.reconciler.ResourceVersions[deploymentKey] = deployment.ResourceVersion
		}
	}
	return nil
}

func (r *ReconcileKafkaDiscovery) registerKafkaInstance(deployment appsv1.Deployment) error {
	kafkaPod, err := r.findAvailablePodByDeployment(&deployment)
	// It is important not to register Kafka instance if an error occurs or the Kafka deployment does not have available pod.
	if err != nil || kafkaPod == nil {
		return err
	}
	consulConfig := consulConfigs[deployment.Name]
	if consulConfig == nil {
		hostIP := kafkaPod.Status.HostIP
		if consulConfig, err = r.createConsulConfig(hostIP); err != nil {
			return err
		}
		consulConfigs[deployment.Name] = consulConfig
	}

	consulClient, err := getConsulClient(consulConfig, r.logger)
	if err != nil {
		return err
	}

	configMap, err := r.reconciler.FindConfigMap(provider.DiscoveryConfigurationName, r.cr.Namespace, r.logger)
	if err != nil {
		return err
	}
	tags, meta := r.getTagsAndMetaFromConfigMap(configMap)

	if err = r.registerInternalKafkaAddress(deployment.Name, tags, meta, consulClient); err != nil {
		return err
	}
	if err = r.registerExternalKafkaAddress(deployment, tags, meta, consulClient); err != nil {
		return err
	}
	return nil
}

func (r *ReconcileKafkaDiscovery) registerInternalKafkaAddress(serviceName string, tags []string, meta map[string]string, consulClient *consulApi.Client) error {
	registrationIPAddress, err := r.reconciler.GetServiceIp(r.cr.Namespace, serviceName)
	if err != nil {
		return err
	}
	checkEndpoint := fmt.Sprintf("%s.%s:%d", serviceName, r.cr.Namespace, kafkaPort)

	return r.registerKafkaAddress(serviceName, registrationIPAddress, kafkaPort, checkEndpoint, tags, meta, consulClient)
}

func (r *ReconcileKafkaDiscovery) registerExternalKafkaAddress(deployment appsv1.Deployment, tags []string, meta map[string]string,
	consulClient *consulApi.Client) error {
	externalKafkaIp := r.reconciler.GetDeploymentParameter(deployment, "EXTERNAL_HOST_NAME")
	externalKafkaPort := r.reconciler.GetDeploymentParameter(deployment, "EXTERNAL_PORT")
	if externalKafkaIp != "" && externalKafkaPort != "" {
		areExternalAddressesUsed = true
		serviceName := fmt.Sprintf("%s-%s", external, deployment.Name)
		registrationPort, err := strconv.Atoi(externalKafkaPort)
		if err != nil {
			return err
		}
		checkEndpoint := fmt.Sprintf("%s:%s", externalKafkaIp, externalKafkaPort)
		if !util.Contains(external, tags) {
			tags = append(tags, external)
		}

		return r.registerKafkaAddress(serviceName, externalKafkaIp, registrationPort, checkEndpoint, tags, meta, consulClient)
	}
	return nil
}

func (r *ReconcileKafkaDiscovery) registerKafkaAddress(registrationId string, registrationIPAddress string, registrationPort int,
	checkEndpoint string, tags []string, meta map[string]string, consulClient *consulApi.Client) error {
	err := r.kafkaDiscoveryProvider.RegisterKafkaAddress(registrationId, registrationIPAddress,
		registrationPort, checkEndpoint, tags, meta, consulClient)
	if err != nil && strings.Contains(err.Error(), aclNotFoundError) {
		CleanDiscoveryData(r.logger)
	}
	return err
}

func (r *ReconcileKafkaDiscovery) createConsulConfig(ipAddress string) (*consulApi.Config, error) {
	consulConfig := consulApi.DefaultConfig()
	consulConfig.Address = fmt.Sprintf("%s:%d", ipAddress, consulClientPort)
	if r.cr.Spec.ConsulAclEnabled {
		if err := r.updateConsulConfigWithToken(consulConfig); err != nil {
			return nil, err
		}
	}
	return consulConfig, nil
}

func (r *ReconcileKafkaDiscovery) updateConsulConfigWithToken(consulConfig *consulApi.Config) error {
	kubeConfig := r.reconciler.GetOperatorClusterConfig()
	loginParams := &consulApi.ACLLoginParams{
		AuthMethod:  r.cr.Spec.ConsulAuthMethod,
		BearerToken: kubeConfig.BearerToken,
	}
	consulClient, err := consulApi.NewClient(consulConfig)
	if err != nil {
		return err
	}
	aclToken, _, err := consulClient.ACL().Login(loginParams, &consulApi.WriteOptions{})
	if err != nil {
		r.logger.Error(err, "Error occurred during Consul login")
		return err
	}
	consulConfig.Token = aclToken.SecretID
	time.Sleep(1 * time.Second)
	return nil
}

func (r *ReconcileKafkaDiscovery) findAvailablePodByDeployment(deployment *appsv1.Deployment) (*corev1.Pod, error) {
	labels := deployment.Spec.Selector.MatchLabels
	pods, err := r.reconciler.FindPodList(r.cr.Namespace, labels)
	if err != nil {
		return nil, err
	}
	return controllers.GetFirstAvailablePod(pods), nil
}

func (r *ReconcileKafkaDiscovery) getTagsAndMetaFromConfigMap(configMap *corev1.ConfigMap) ([]string, map[string]string) {
	var tags []string
	_ = json.Unmarshal([]byte(configMap.Data[provider.ConsulDiscoveryTags]), &tags)
	var meta map[string]string
	_ = json.Unmarshal([]byte(configMap.Data[provider.ConsulDiscoveryMeta]), &meta)
	return tags, meta
}

func getConsulClient(consulConfig *consulApi.Config, logger logr.Logger) (*consulApi.Client, error) {
	client, err := consulApi.NewClient(consulConfig)
	if err != nil {
		logger.Error(err, "Can not create Consul client")
		return nil, err
	}
	return client, nil
}

func CleanDiscoveryData(logger logr.Logger) {
	logger.Info("Clean discovery data")
	if consulConfigs == nil {
		return
	}
	for name, consulConfig := range consulConfigs {
		if consulConfig != nil {
			consulClient, err := getConsulClient(consulConfig, logger)
			if err != nil {
				return
			}
			if err := deregisterServices(name, consulClient); err != nil {
				if !strings.Contains(err.Error(), aclNotFoundError) {
					logger.Error(err, "Error occurred during services deregistration")
					continue
				}
			}
			logoutConsulToken(consulConfig.Token, consulClient, logger)
		}
		delete(consulConfigs, name)
	}
}

func deregisterServices(serviceName string, consulClient *consulApi.Client) error {
	if err := consulClient.Agent().ServiceDeregister(serviceName); err != nil {
		return err
	}
	if areExternalAddressesUsed {
		externalServiceName := fmt.Sprintf("%s-%s", external, serviceName)
		if err := consulClient.Agent().ServiceDeregister(externalServiceName); err != nil {
			return err
		}
	}
	return nil
}

func logoutConsulToken(token string, consulClient *consulApi.Client, logger logr.Logger) {
	if token != "" {
		_, err := consulClient.ACL().Logout(&consulApi.WriteOptions{Token: token})
		if err != nil {
			logger.Error(err, "Error occurred during Consul logout")
		}
	}
}
