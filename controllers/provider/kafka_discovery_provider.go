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

package provider

import (
	"encoding/json"
	"fmt"
	kafkaservice "github.com/Netcracker/qubership-kafka/api/v1"
	"github.com/Netcracker/qubership-kafka/util"
	"github.com/go-logr/logr"
	consulApi "github.com/hashicorp/consul/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConsulDeregisterCriticalServiceAfter string = "100s"
	ConsulDiscoveryMeta                  string = "meta"
	ConsulDiscoveryTags                  string = "tags"
	ConsulHealthCheckInterval            string = "10s"
	ConsulHealthCheckTimeout             string = "1s"
	DiscoveryConfigurationName           string = "kafka-service-discovery-data"
)

type KafkaDiscoveryProvider struct {
	cr     *kafkaservice.Kafka
	logger logr.Logger
	spec   kafkaservice.KafkaSpec
}

func NewKafkaDiscoveryProvider(cr *kafkaservice.Kafka, logger logr.Logger) KafkaDiscoveryProvider {
	return KafkaDiscoveryProvider{cr: cr, spec: cr.Spec, logger: logger}
}

func (kdp KafkaDiscoveryProvider) CreateDiscoveryConfigMapDef() (*corev1.ConfigMap, error) {
	tagsString, err := json.Marshal(kdp.spec.KafkaDiscoveryTags)
	if err != nil {
		kdp.logger.Error(err, "Can't read tags from discovery config")
		return nil, err
	}
	metaString, err := json.Marshal(kdp.spec.KafkaDiscoveryMeta)
	if err != nil {
		kdp.logger.Error(err, "Can't read meta from discovery config")
		return nil, err
	}
	labels := make(map[string]string)
	labels["component"] = "kafka-service"
	labels = util.JoinMaps(labels, kdp.cr.Spec.DefaultLabels)
	discoveryConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DiscoveryConfigurationName,
			Namespace: kdp.cr.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			ConsulDiscoveryTags: string(tagsString),
			ConsulDiscoveryMeta: string(metaString),
		},
	}
	return discoveryConfig, nil
}

func (kdp KafkaDiscoveryProvider) RegisterKafkaAddress(registrationId string, registrationIPAddress string, registrationPort int,
	checkEndpoint string, tags []string, meta map[string]string, consulClient *consulApi.Client) error {

	serviceCheck := &consulApi.AgentServiceCheck{
		Name:                           "tcp-check",
		Interval:                       ConsulHealthCheckInterval,
		Timeout:                        ConsulHealthCheckTimeout,
		TCP:                            checkEndpoint,
		DeregisterCriticalServiceAfter: ConsulDeregisterCriticalServiceAfter,
	}

	serviceDefinition := &consulApi.AgentServiceRegistration{
		Name:              kdp.spec.RegisteredServiceName,
		ID:                registrationId,
		Address:           registrationIPAddress,
		Port:              registrationPort,
		Tags:              tags,
		Meta:              meta,
		EnableTagOverride: true,
		Check:             serviceCheck,
	}

	err := consulClient.Agent().ServiceRegisterOpts(serviceDefinition, consulApi.ServiceRegisterOpts{ReplaceExistingChecks: true})
	if err != nil {
		kdp.logger.Error(err, "Error during Consul registration")
		return err
	}
	kdp.logger.Info(fmt.Sprintf("Kafka address with <%s> ID (ip: %s) is registered in Consul", registrationId, registrationIPAddress))
	return nil
}
