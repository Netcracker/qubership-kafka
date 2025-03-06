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

package kafkaservice

import (
	"context"
	kafkaservice "git.netcracker.com/PROD.Platform.Streaming/kafka-service/kafka-service-operator/api/v7"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusUpdater struct {
	client    client.Client
	name      string
	namespace string
}

func NewStatusUpdater(client client.Client, cr *kafkaservice.KafkaService) StatusUpdater {
	return StatusUpdater{
		client:    client,
		name:      cr.Name,
		namespace: cr.Namespace,
	}
}

func (su StatusUpdater) UpdateStatusWithRetry(statusUpdateFunc func(*kafkaservice.KafkaService)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance, err := su.reloadCR()
		if err != nil {
			return err
		}
		statusUpdateFunc(instance)
		return su.client.Status().Update(context.TODO(), instance)
	})
}

func (su StatusUpdater) GetStatus() (*kafkaservice.KafkaServiceStatus, error) {
	instance, err := su.reloadCR()
	if err != nil {
		return nil, err
	}
	return &instance.Status, nil
}

func (su StatusUpdater) reloadCR() (*kafkaservice.KafkaService, error) {
	instance := &kafkaservice.KafkaService{}
	err := su.client.Get(context.TODO(),
		types.NamespacedName{Name: su.name, Namespace: su.namespace}, instance)
	return instance, err
}
