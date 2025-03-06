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
	kafkaservice "github.com/Netcracker/qubership-kafka/api/v7"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const (
	statusFalse     = "False"
	statusTrue      = "True"
	typeInProgress  = "In progress"
	typeFailed      = "Failed"
	typeReady       = "Ready"
	typeSuccessful  = "Successful"
	waitingInterval = 10 * time.Second
)

func NewCondition(conditionStatus string, conditionType string, conditionReason string, conditionMessage string) kafkaservice.StatusCondition {
	return kafkaservice.StatusCondition{
		Type:    conditionType,
		Status:  conditionStatus,
		Reason:  conditionReason,
		Message: conditionMessage,
	}
}

func (r *KafkaServiceReconciler) updateConditions(condition kafkaservice.StatusCondition) error {
	return r.StatusUpdater.UpdateStatusWithRetry(func(instance *kafkaservice.KafkaService) {
		currentConditions := instance.Status.Conditions
		condition.LastTransitionTime = metav1.Now().String()
		currentConditions = addCondition(currentConditions, condition)
		instance.Status.Conditions = currentConditions
	})
}

func addCondition(currentConditions []kafkaservice.StatusCondition, condition kafkaservice.StatusCondition) []kafkaservice.StatusCondition {
	for i, currentCondition := range currentConditions {
		if currentCondition.Reason == condition.Reason {
			if currentCondition.Type != condition.Type ||
				currentCondition.Status != condition.Status ||
				currentCondition.Message != condition.Message {
				currentConditions[i] = condition
			}
			return currentConditions
		}
	}
	return append(currentConditions, condition)
}

func hasFailedConditions(status *kafkaservice.KafkaServiceStatus) bool {
	for _, condition := range status.Conditions {
		if condition.Type == typeFailed {
			return true
		}
	}
	return false
}

func (r *KafkaServiceReconciler) clearAllConditions() error {
	return r.StatusUpdater.UpdateStatusWithRetry(func(instance *kafkaservice.KafkaService) {
		instance.Status.Conditions = []kafkaservice.StatusCondition{}
	})
}
