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
	"errors"
	"fmt"
	"github.com/Netcracker/qubership-kafka/util"
	"testing"
)

func makeDefaultTestProperties() map[string]string {
	properties := make(map[string]string)
	properties["clusters"] = "dc1,dc2"
	properties["target.dc"] = "dc1"
	properties["dc1.bootstrap.servers"] = "kafka-1:9092,kafka-2:9092,kafka-3:9092"
	properties["dc2.bootstrap.servers"] = "kafka-1.kafka-service:9092"
	properties["dc2->dc1.enabled"] = "true"
	properties["dc2->dc1.topics"] = "topic-7"
	properties["topics.blacklist"] = "dc2\\.*,dc1\\.*"
	return properties
}

func errorEquals(actualError error, expectedError error) bool {
	if expectedError == nil {
		return actualError == nil
	}
	return expectedError.Error() == actualError.Error()
}

func TestKmmDrConfig_resolveReplicationEnable(t *testing.T) {
	emptyEnabledProperties := makeDefaultTestProperties()
	delete(emptyEnabledProperties, "dc2->dc1.enabled")

	globalDisabledLocalEnabledProperties := makeDefaultTestProperties()
	globalDisabledLocalEnabledProperties["enabled"] = "false"

	globalEnabledLocalDisabledProperties := makeDefaultTestProperties()
	globalEnabledLocalDisabledProperties["enabled"] = "true"
	globalEnabledLocalDisabledProperties["dc2->dc1.enabled"] = "false"

	globalDisabledOnlyProperties := makeDefaultTestProperties()
	delete(globalDisabledOnlyProperties, "dc2->dc1.enabled")
	globalDisabledOnlyProperties["enabled"] = "false"

	tests := []struct {
		properties      map[string]string
		expectedEnabled bool
	}{
		{emptyEnabledProperties, true},
		{globalDisabledLocalEnabledProperties, true},
		{globalEnabledLocalDisabledProperties, false},
		{globalDisabledOnlyProperties, false},
	}
	for _, test := range tests {
		kmmDrConfig, err := NewKmmDrConfig(test.properties)
		if err != nil {
			t.Errorf("Unexpected error, %v", err)
		} else {
			if kmmDrConfig.ReplicationEnabled != test.expectedEnabled {
				t.Errorf("enabling must be %t ut %t given", test.expectedEnabled, kmmDrConfig.ReplicationEnabled)
			}
		}
	}
}

func TestKmmDrConfig_validateClusters(t *testing.T) {
	validProperties := makeDefaultTestProperties()
	var validExpectedError error

	propertiesWithoutClusters := makeDefaultTestProperties()
	delete(propertiesWithoutClusters, "clusters")
	errorWithoutClusters := errors.New(clustersPropertyNotFoundError)

	propertiesWithThreeClusters := makeDefaultTestProperties()
	propertiesWithThreeClusters["clusters"] = "dc1, dc2,dc3"
	errorWithThreeClusters := errors.New(incorrectClustersCountError)

	propertiesWithRepeatedClusters := makeDefaultTestProperties()
	propertiesWithRepeatedClusters["clusters"] = "dc1,dc1"
	errorWithRepeatedClusters := errors.New(sameClusterNameError)

	propertiesWithoutTargetDc := makeDefaultTestProperties()
	delete(propertiesWithoutTargetDc, "target.dc")
	errorTargetDc := errors.New(targetDcPropertyInvalidError)

	propertiesWithIncorrectTargetDc := makeDefaultTestProperties()
	propertiesWithIncorrectTargetDc["target.dc"] = "dc1,dc2"

	tests := []struct {
		properties    map[string]string
		expectedError error
	}{
		{validProperties, validExpectedError},
		{propertiesWithoutClusters, errorWithoutClusters},
		{propertiesWithThreeClusters, errorWithThreeClusters},
		{propertiesWithRepeatedClusters, errorWithRepeatedClusters},
		{propertiesWithoutTargetDc, errorTargetDc},
		{propertiesWithIncorrectTargetDc, errorTargetDc},
	}
	for _, test := range tests {
		_, err := NewKmmDrConfig(test.properties)
		if !errorEquals(err, test.expectedError) {
			t.Errorf("Unexpected error, %v", err)
		}
	}
}

func TestKmmDrConfig_getBrokers(t *testing.T) {
	properties := makeDefaultTestProperties()
	expectedTarget := []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"}
	expectedSource := []string{"kafka-1.kafka-service:9092"}
	var expectedErr error

	propertiesWithoutSourceBrokers := makeDefaultTestProperties()
	delete(propertiesWithoutSourceBrokers, "dc2.bootstrap.servers")
	expectedErrSourceBrokers := errors.New("source Kafka bootstrap servers are not set")

	propertiesWithoutTargetBrokers := makeDefaultTestProperties()
	delete(propertiesWithoutTargetBrokers, "dc1.bootstrap.servers")
	expectedErrTargetBrokers := errors.New("target Kafka bootstrap servers are not set")

	propertiesWithIncorrectSourceBrokers := makeDefaultTestProperties()
	delete(propertiesWithIncorrectSourceBrokers, "dc2.bootstrap.servers")
	propertiesWithIncorrectSourceBrokers["dc3.bootstrap.servers"] = "kafka-1.kafka-service:9092"

	propertiesWithIncorrectTargetBrokers := makeDefaultTestProperties()
	delete(propertiesWithIncorrectTargetBrokers, "dc1.bootstrap.servers")
	propertiesWithIncorrectTargetBrokers["dc3.bootstrap.servers"] = "kafka-1:9092,kafka-2:9092,kafka-3:9092"

	tests := []struct {
		properties            map[string]string
		expectedError         error
		expectedTargetBrokers []string
		expectedSourceBrokers []string
	}{
		{properties, expectedErr, expectedTarget, expectedSource},
		{propertiesWithoutSourceBrokers, expectedErrSourceBrokers, nil, nil},
		{propertiesWithoutTargetBrokers, expectedErrTargetBrokers, nil, nil},
		{propertiesWithIncorrectSourceBrokers, expectedErrSourceBrokers, nil, nil},
		{propertiesWithIncorrectTargetBrokers, expectedErrTargetBrokers, nil, nil},
	}
	for _, test := range tests {
		kmmDrConfig, err := NewKmmDrConfig(test.properties)
		if !errorEquals(err, test.expectedError) {
			t.Errorf("Unexpected error, %v", err)
		}
		if err == nil {
			target := kmmDrConfig.TargetBrokers
			source := kmmDrConfig.SourceBrokers
			if !util.EqualSlices(target, test.expectedTargetBrokers) {
				t.Errorf("target error")
			}
			if !util.EqualSlices(source, test.expectedSourceBrokers) {
				t.Errorf("source error")
			}
		}
	}
}

func TestKmmDrConfig_GetRegularExpressions(t *testing.T) {
	defaultProperties := makeDefaultTestProperties()
	expectedDefaultAllow := []string{"topic-7"}
	expectedDefaultBlock := []string{"dc2\\.*", "dc1\\.*"}
	var expectedError error

	replicationDisabledProperties := makeDefaultTestProperties()
	replicationDisabledProperties["dc2->dc1.enabled"] = "false"

	globalAndLocalProperties := makeDefaultTestProperties()
	globalAndLocalProperties["topics"] = "topic-8"
	globalAndLocalProperties["dc2->dc1.topics.blacklist"] = "topic-9,topic-10"
	expectedGlobalAndLocalAllow := []string{"topic-7"}
	expectedGlobalAndLocalBlock := []string{"topic-9", "topic-10"}

	globalOnlyProperties := makeDefaultTestProperties()
	globalOnlyProperties["topics"] = "topic-8"
	delete(globalOnlyProperties, "dc2->dc1.topics")
	expectedGlobalOnlyAllow := []string{"topic-8"}

	emptyProperties := makeDefaultTestProperties()
	delete(emptyProperties, "dc2->dc1.topics")
	delete(emptyProperties, "topics.blacklist")

	tests := []struct {
		properties          map[string]string
		expectedError       error
		expectedAllowRegExp []string
		expectedBlockRegExp []string
	}{
		{defaultProperties, expectedError, expectedDefaultAllow, expectedDefaultBlock},
		{replicationDisabledProperties, nil, expectedDefaultAllow, expectedDefaultBlock},
		{globalAndLocalProperties, nil, expectedGlobalAndLocalAllow, expectedGlobalAndLocalBlock},
		{globalOnlyProperties, nil, expectedGlobalOnlyAllow, expectedDefaultBlock},
		{emptyProperties, nil, nil, nil},
	}
	for _, test := range tests {
		kmmDrConfig, err := NewKmmDrConfig(test.properties)
		if err != nil {
			t.Errorf("Unexpected error, %v", err)
		} else {
			allow, block := kmmDrConfig.GetRegularExpressions()
			if allow == nil {
				fmt.Println(block)
			}
			if !util.EqualSlicesNil(allow, test.expectedAllowRegExp) {
				t.Errorf("unexpected allow expressions. Expected: %s, but given: %s", test.expectedAllowRegExp, allow)
			}
			if !util.EqualSlicesNil(block, test.expectedBlockRegExp) {
				t.Errorf("unexpected block expressions. Expected: %s, but given: %s", test.expectedBlockRegExp, block)
			}
		}
	}
}
