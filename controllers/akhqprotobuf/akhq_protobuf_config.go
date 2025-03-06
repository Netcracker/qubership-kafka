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

package akhqprotobuf

import (
	"context"
	"reflect"

	kafkaservice "github.com/Netcracker/qubership-kafka/api/v7"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	protobufConfigurationCMName = "akhq-protobuf-configuration"
	protobufConfig              = `deserialization:
 protobuf:
   descriptors-folder: "/app/config"
   topics-mapping: []
`
)

func getConfigMap(configMapName, namespace string, client client.Client) (*v1.ConfigMap, error) {
	foundCM := &v1.ConfigMap{}
	if err := client.Get(context.TODO(),
		types.NamespacedName{Name: configMapName, Namespace: namespace}, foundCM); err != nil {
		return nil, err
	}
	return foundCM, nil
}

func CreateProtobufConfigMapIfNotExist(namespace string, client client.Client, labels map[string]string,
	controller *kafkaservice.KafkaService, scheme *runtime.Scheme) (*corev1.ConfigMap, error) {

	needToCreate := false
	needToUpdate := false

	protobufConfigMap, err := getConfigMap(protobufConfigurationCMName, namespace, client)
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		protobufConfigMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      protobufConfigurationCMName,
				Namespace: namespace,
			},
			Data: map[string]string{"config": protobufConfig},
		}
		needToCreate = true
	}

	if !reflect.DeepEqual(protobufConfigMap.Labels, labels) {
		protobufConfigMap.Labels = labels
		needToUpdate = true
	}

	if controller != nil && scheme != nil {
		if err = controllerutil.SetControllerReference(controller, protobufConfigMap, scheme); err != nil {
			return &corev1.ConfigMap{}, err
		}
		needToUpdate = true
	}

	if needToCreate {
		if err = client.Create(context.TODO(), protobufConfigMap); err != nil {
			return nil, err
		}
	} else if needToUpdate {
		if err = client.Update(context.TODO(), protobufConfigMap); err != nil {
			return nil, err
		}
	}

	return protobufConfigMap, nil

}
