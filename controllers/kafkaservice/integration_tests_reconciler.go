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
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

const integrationTestsConditionReason = "KafkaIntegrationTestsStatus"

type ReconcileIntegrationTests struct {
	reconciler *KafkaServiceReconciler
	cr         *kafkaservice.KafkaService
	logger     logr.Logger
}

func NewReconcileIntegrationTests(r *KafkaServiceReconciler, cr *kafkaservice.KafkaService, logger logr.Logger) ReconcileIntegrationTests {
	return ReconcileIntegrationTests{
		reconciler: r,
		cr:         cr,
		logger:     logger,
	}
}

func (r ReconcileIntegrationTests) Reconcile() error {
	return nil
}

func (r ReconcileIntegrationTests) Status() error {
	if !r.cr.Spec.IntegrationTests.WaitForResult {
		return nil
	}

	if err := r.reconciler.updateConditions(NewCondition(statusFalse,
		typeInProgress,
		integrationTestsConditionReason,
		"Start checking Kafka Integration Tests")); err != nil {
		return err
	}
	r.logger.Info("Start checking Kafka Integration Tests")
	err := wait.PollImmediate(waitingInterval, time.Duration(r.cr.Spec.IntegrationTests.Timeout)*time.Second, func() (done bool, err error) {
		labels := r.getKafkaIntegrationTestsLabels()
		return r.reconciler.AreDeploymentsReady(labels, r.cr.Namespace, r.logger), nil
	})
	if err != nil {
		return r.reconciler.updateConditions(NewCondition(statusFalse, typeFailed, integrationTestsConditionReason, "Kafka Integration Tests failed. See more details in integration test logs"))
	}
	return r.reconciler.updateConditions(NewCondition(statusTrue, typeReady, integrationTestsConditionReason, "Kafka Integration Tests performed successfully"))
}

func (r ReconcileIntegrationTests) getKafkaIntegrationTestsLabels() map[string]string {
	return map[string]string{
		"name": r.cr.Spec.IntegrationTests.ServiceName,
	}
}
