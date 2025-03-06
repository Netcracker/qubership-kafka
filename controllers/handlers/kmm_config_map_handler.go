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

package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/Netcracker/qubership-kafka/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KmmConfigError struct {
	Message string
	Err     error
}

func (kce *KmmConfigError) Error() string {
	kce.Err = errors.New("kafka mirror maker config is invalid")
	return fmt.Sprintf("%s-%s", kce.Err, kce.Message)
}

type KmmConfigHandler struct {
	client client.Client
	config *KmmDrConfig
	data   map[string][]string
}

func NewKmmConfigHandler(client client.Client) (*KmmConfigHandler, error) {
	handler := &KmmConfigHandler{
		client: client,
	}
	if err := handler.enrichKmmHandler(); err != nil {
		return nil, err
	}
	return handler, nil
}

func (kch *KmmConfigHandler) enrichKmmHandler() error {
	cmContent, err := kch.getKmmConfigMapContent()
	if err != nil {
		return nil
	}
	properties := getStringProperties(cmContent)
	config, err := NewKmmDrConfig(properties)
	if err != nil {
		return nil
	}
	kch.config = config
	kch.data = make(map[string][]string)
	targetCreds, sourceCreds, err := kch.getKafkaCredentials(kch.config.TargetCluster, kch.config.SourceCluster)
	if err != nil {
		return err
	}
	kch.data["targetCredentials"] = targetCreds
	kch.data["sourceCredentials"] = sourceCreds
	return nil
}

func (kch *KmmConfigHandler) getKmmConfigMapContent() (string, error) {
	name := util.GetKmmConfigMapName()
	namespace := os.Getenv("OPERATOR_NAMESPACE")
	kmmConfigMap := &corev1.ConfigMap{}
	err := kch.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, kmmConfigMap)
	if err != nil {
		return "", err
	}
	return kmmConfigMap.Data["config"], nil
}

func (kch *KmmConfigHandler) getKafkaCredentials(targetCluster string, sourceCluster string) ([]string, []string, error) {
	secretName := util.GetKmmSecretName()
	namespace := os.Getenv("OPERATOR_NAMESPACE")
	kmmSecret := &corev1.Secret{}
	err := kch.client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: namespace}, kmmSecret)
	if err != nil {
		return nil, nil, err
	}
	userPattern := "-kafka-username"
	passwordPattern := "-kafka-password"
	var targetUser, targetPassword, sourceUser, sourcePassword string
	targetUserKey := fmt.Sprintf("%s%s", targetCluster, userPattern)
	targetPasswordKey := fmt.Sprintf("%s%s", targetCluster, passwordPattern)
	sourceUserKey := fmt.Sprintf("%s%s", sourceCluster, userPattern)
	sourcePasswordKey := fmt.Sprintf("%s%s", sourceCluster, passwordPattern)
	if value, ok := kmmSecret.Data[targetUserKey]; !ok {
		return nil, nil, fmt.Errorf("can not find username for Kafka cluster with key - %s", targetUserKey)
	} else {
		targetUser = string(value)
	}
	if value, ok := kmmSecret.Data[sourceUserKey]; !ok {
		return nil, nil, fmt.Errorf("can not find username for Kafka cluster with key - %s", targetUserKey)
	} else {
		sourceUser = string(value)
	}
	if value, ok := kmmSecret.Data[targetPasswordKey]; !ok {
		return nil, nil, fmt.Errorf("can not find password for Kafka cluster with key - %s", targetUserKey)
	} else {
		targetPassword = string(value)
	}
	if value, ok := kmmSecret.Data[sourcePasswordKey]; !ok {
		return nil, nil, fmt.Errorf("can not find password for Kafka cluster with key - %s", targetUserKey)
	} else {
		sourcePassword = string(value)
	}
	return []string{targetUser, targetPassword}, []string{sourceUser, sourcePassword}, nil
}

func (kch *KmmConfigHandler) GetKafkaUsernamePasswords() ([]string, []string) {
	return kch.data["targetCredentials"], kch.data["sourceCredentials"]
}

func (kch *KmmConfigHandler) GetBrokers() ([]string, []string) {
	return kch.config.TargetBrokers, kch.config.SourceBrokers
}

func (kch *KmmConfigHandler) GetTargetCluster() string {
	return kch.config.TargetCluster
}

func (kch *KmmConfigHandler) GetSourceCluster() string {
	return kch.config.SourceCluster
}

func (kch *KmmConfigHandler) IsReplicationEnabled() bool {
	return kch.config.ReplicationEnabled
}

func (kch *KmmConfigHandler) GetRegularExpressions() ([]string, []string) {
	return kch.config.GetRegularExpressions()
}
