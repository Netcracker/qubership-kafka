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

package kafkauser

import (
	"context"
	kafka "github.com/Netcracker/qubership-kafka/operator/api/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CustomResourceUpdater struct {
	client    client.Client
	name      string
	namespace string
}

func NewCustomResourceUpdater(client client.Client, cr *kafka.KafkaUser) CustomResourceUpdater {
	return CustomResourceUpdater{
		client:    client,
		name:      cr.Name,
		namespace: cr.Namespace,
	}
}

func (cru CustomResourceUpdater) UpdateWithRetry(updateFunc func(*kafka.KafkaUser)) error {
	return cru.updateWithRetry(updateFunc, cru.client)
}

func (cru CustomResourceUpdater) UpdateStatusWithRetry(statusUpdateFunc func(*kafka.KafkaUser)) error {
	return cru.updateWithRetry(statusUpdateFunc, cru.client.Status())
}

func (cru CustomResourceUpdater) updateWithRetry(updateFunc func(*kafka.KafkaUser), writer client.StatusWriter) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance, err := cru.GetCustomResource()
		if err != nil {
			return err
		}
		updateFunc(instance)
		return writer.Update(context.TODO(), instance)
	})
}

func (cru CustomResourceUpdater) GetCustomResource() (*kafka.KafkaUser, error) {
	instance := &kafka.KafkaUser{}
	err := cru.client.Get(context.TODO(),
		types.NamespacedName{Name: cru.name, Namespace: cru.namespace}, instance)
	return instance, err
}
