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
	"github.com/Netcracker/qubership-kafka/util"
	"os"
	"testing"
)

var cmContent string

func Setup() {
	cmContent = "# mm2.properties\nclusters = dc2, dc1\nreplication.factor = 3\nconfig.storage.replication.factor = 3\noffset.storage.replication.factor = 3\nstatus.storage.replication.factor = 3\nheartbeats.topic.replication.factor = 3\ncheckpoints.topic.replication.factor = 3\noffset-syncs.topic.replication.factor = 3\nrefresh.topics.interval.seconds = 5\nrefresh.groups.interval.seconds = 5\nsync.topic.acls.enabled = false\n\n# configure [dc2] cluster\ndc2.bootstrap.servers = kafka-1.kafka-service:9092,kafka-2.kafka-service:9092,kafka-3.kafka-service:9092\n\n# configure [dc1] cluster\ndc1.bootstrap.servers = kafka-1.kafka-cluster:9092,kafka-2.kafka-cluster:9092,kafka-3.kafka-cluster:9092\n\ntarget.dc=dc2\nreplication.policy.class = io.strimzi.kafka.connect.mirror.IdentityReplicationPolicy\nemit.checkpoints.enabled = true\nemit.checkpoints.interval.seconds = 5\nsync.group.offsets.enabled = true\nsync.group.offsets.interval.seconds = 5\nenabled = false\ntopics = topic-71,topic-72\n# configure a specific source->target replication flow\ndc1->dc2.enabled = true\ntopics.blacklist = dc2\\.*,dc1\\.*"
	os.Setenv("OPERATOR_NAME", "kafka-service-operator")
}

func createEmptyKmmSecret() map[string]string {
	kmmSecretContent := make(map[string]string)
	kmmSecretContent["dc1-kafka-username"] = ""
	kmmSecretContent["dc2-kafka-username"] = ""
	kmmSecretContent["dc1-kafka-password"] = ""
	kmmSecretContent["dc2-kafka-password"] = ""
	return kmmSecretContent
}

func TestKmmConfigHandler_GetKafkaUsernamePasswords(t *testing.T) {
	Setup()
	kmmSecretContent := make(map[string]string)
	kmmSecretContent["dc1-kafka-username"] = "client"
	kmmSecretContent["dc2-kafka-username"] = "client1"
	kmmSecretContent["dc1-kafka-password"] = "password"
	kmmSecretContent["dc2-kafka-password"] = "password1"
	expectedTarget := []string{"client1", "password1"}
	expectedSource := []string{"client", "password"}
	client := TestClient{
		ConfigMapContent: cmContent,
		SecretData:       kmmSecretContent,
	}

	emptySecret := createEmptyKmmSecret()
	expectedEmptyTarget := []string{"", ""}
	expectedEmptySource := []string{"", ""}

	emptyClient := TestClient{
		ConfigMapContent: cmContent,
		SecretData:       emptySecret,
	}

	tests := []struct {
		client         TestClient
		expectedError  error
		expectedTarget []string
		expectedSource []string
	}{
		{emptyClient, nil, expectedEmptyTarget, expectedEmptySource},
		{client, nil, expectedTarget, expectedSource},
	}
	for _, test := range tests {
		handler, err := NewKmmConfigHandler(test.client)
		if !errorEquals(err, test.expectedError) {
			t.Errorf("unexpected error %v", err)
		} else {
			target, source := handler.GetKafkaUsernamePasswords()
			if !util.EqualSlices(target, test.expectedTarget) {
				t.Errorf("unexpected target cluster credentials. Expected - %s, but given - %s",
					expectedTarget, target)
			}
			if !util.EqualSlices(source, test.expectedSource) {
				t.Errorf("unexpected source cluster credentials. Expected - %s, but given - %s",
					expectedSource, source)
			}
		}
	}
}

func TestKmmConfigHandler_GetBrokers(t *testing.T) {
	Setup()
	kmmSecretContent := createEmptyKmmSecret()
	client := TestClient{
		ConfigMapContent: cmContent,
		SecretData:       kmmSecretContent,
	}
	handler, err := NewKmmConfigHandler(client)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	target, source := handler.GetBrokers()
	expectedTarget := []string{"kafka-1.kafka-service:9092", "kafka-2.kafka-service:9092", "kafka-3.kafka-service:9092"}
	expectedSource := []string{"kafka-1.kafka-cluster:9092", "kafka-2.kafka-cluster:9092", "kafka-3.kafka-cluster:9092"}
	if !util.EqualSlices(target, expectedTarget) {
		t.Errorf("unexpected target cluster brokers. Expected - %s, but given - %s",
			expectedTarget, target)
	}
	if !util.EqualSlices(source, expectedSource) {
		t.Errorf("unexpected source cluster brokers. Expected - %s, but given - %s",
			expectedSource, source)
	}
}

func TestKmmConfigHandler_GetRegularExpressions(t *testing.T) {
	Setup()
	kmmSecretContent := createEmptyKmmSecret()
	client := TestClient{
		ConfigMapContent: cmContent,
		SecretData:       kmmSecretContent,
	}
	handler, err := NewKmmConfigHandler(client)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	allow, block := handler.GetRegularExpressions()
	expectedAllow := []string{"topic-71", "topic-72"}
	expectedBlock := []string{"dc2\\.*", "dc1\\.*"}
	if !util.EqualSlices(allow, expectedAllow) {
		t.Errorf("unexpected allow regular expressions. Expected - %s, but given - %s",
			expectedAllow, allow)
	}
	if !util.EqualSlices(block, expectedBlock) {
		t.Errorf("unexpected block regular expressions. Expected - %s, but given - %s",
			expectedBlock, block)
	}
}

func TestKmmConfigHandler_IsReplicationEnabled(t *testing.T) {
	Setup()
	kmmSecretContent := createEmptyKmmSecret()
	client := TestClient{
		ConfigMapContent: cmContent,
		SecretData:       kmmSecretContent,
	}
	handler, err := NewKmmConfigHandler(client)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	replicationEnabled := handler.IsReplicationEnabled()
	expectedReplicationEnabled := true
	if replicationEnabled != expectedReplicationEnabled {
		t.Errorf("unexpected replicationEnabled property. Expected - %t, but given - %t",
			expectedReplicationEnabled, replicationEnabled)
	}
}
