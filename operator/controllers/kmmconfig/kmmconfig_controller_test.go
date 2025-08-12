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

package kmmconfig

import (
	kmmconfig "github.com/Netcracker/qubership-kafka/operator/api/v1"
	"github.com/Netcracker/qubership-kafka/operator/util"
	"strings"
	"testing"
)

func createTestInstance(topics string, sourceDc string, targetDc string) kmmconfig.KmmConfig {
	instance := kmmconfig.KmmConfig{}
	kmmTopics := kmmconfig.KmmTopics{}
	kmmTopics.Topics = topics
	kmmTopics.TargetDc = targetDc
	kmmTopics.SourceDc = sourceDc
	instance.Spec.KmmTopics = &kmmTopics
	return instance
}

func TestRecognizeTargetAndSourceDc(t *testing.T) {
	simpleInstance := createTestInstance("topic1", "dc2", "dc1")
	coupleProperties := make(map[string]interface{})
	coupleProperties["clusters"] = []string{"dc2", "dc1"}
	coupleProperties["target.dc"] = "dc1"
	expectedTargetDcForSimpleInstance := []string{"dc1"}
	expectedSourceDcForSimpleInstance := []string{"dc2"}

	twoSourceInstance := createTestInstance("topic1", "dc2,dc3", "dc1")
	twoSourceProperties := make(map[string]interface{})
	twoSourceProperties["clusters"] = []string{"dc2", "dc1", "dc3"}
	twoSourceProperties["target.dc"] = "dc1"
	expectedTargetDcForTwoSourceInstance := []string{"dc1"}
	expectedSourceDcForTwoSourceInstance := []string{"dc2", "dc3"}

	multipleInstance := createTestInstance("topic1", "dc2,dc3", "dc1,dc4")
	multipleProperties := make(map[string]interface{})
	multipleProperties["clusters"] = []string{"dc2", "dc1", "dc3", "dc4"}
	expectedTargetDcForMultipleInstance := []string{"dc1", "dc4"}
	expectedSourceDcForMultipleInstance := []string{"dc2", "dc3"}

	singleInstance := createTestInstance("topic1", "dc2", "dc1")
	withoutTargetProperties := make(map[string]interface{})
	withoutTargetProperties["clusters"] = []string{"dc2", "dc1"}
	expectedTargetDcForSingalInstance := []string{"dc1"}
	expectedSourceDcForSingalInstance := []string{"dc2"}

	errorsMap := make(map[string][]string)

	tests := []struct {
		cr               *kmmconfig.KmmConfig
		properties       map[string]interface{}
		errorsMap        *map[string][]string
		expectedTargetDc []string
		expectedSourceDc []string
	}{
		{&simpleInstance, coupleProperties, &errorsMap, expectedTargetDcForSimpleInstance, expectedSourceDcForSimpleInstance},
		{&twoSourceInstance, twoSourceProperties, &errorsMap, expectedTargetDcForTwoSourceInstance, expectedSourceDcForTwoSourceInstance},
		{&multipleInstance, multipleProperties, &errorsMap, expectedTargetDcForMultipleInstance, expectedSourceDcForMultipleInstance},
		{&singleInstance, withoutTargetProperties, &errorsMap, expectedTargetDcForSingalInstance, expectedSourceDcForSingalInstance},
	}

	for _, test := range tests {
		targetDc, sourceDc, _ := recognizeTargetAndSourceDc(test.cr, test.properties, test.errorsMap)
		if !util.EqualSlices(targetDc, test.expectedTargetDc) {
			t.Errorf("%s - unexpected targetDc list; expected - %s ", targetDc, test.expectedTargetDc)
		}
		if !util.EqualSlices(sourceDc, test.expectedSourceDc) {
			t.Errorf("%s - unexpected sourceDc list; expected - %s ", sourceDc, test.expectedSourceDc)
		}
	}
}

func TestErrorsRecognizeTargetAndSourceDc(t *testing.T) {
	incorrectSourceInstance := createTestInstance("topic1", "dc3", "dc1")
	properties := make(map[string]interface{})
	properties["clusters"] = []string{"dc2", "dc1"}
	expectedSourceErrorMap := map[string][]string{"sourceDc": {"dc3 - 'clusters' property does not contain current source dc"}, "targetDc": {}}

	incorrectTargetInstance := createTestInstance("topic1", "dc2", "dc3")
	expectedTargetErrorMap := map[string][]string{"targetDc": {"dc3 - neither 'target.dc' property nor 'clusters' property contains current target dc"}, "sourceDc": {}}

	withTargetProperties := make(map[string]interface{})
	withTargetProperties["clusters"] = []string{"dc2", "dc1"}
	withTargetProperties["target.dc"] = "dc1"
	expectedSpecialTargetErrorMap := map[string][]string{"targetDc": {"dc3 - target.dc property does not contain target dc"}, "sourceDc": {}}

	tests := []struct {
		cr               *kmmconfig.KmmConfig
		properties       map[string]interface{}
		expectedErrorMap map[string][]string
	}{
		{&incorrectSourceInstance, properties, expectedSourceErrorMap},
		{&incorrectTargetInstance, properties, expectedTargetErrorMap},
		{&incorrectTargetInstance, withTargetProperties, expectedSpecialTargetErrorMap},
	}

	for _, test := range tests {
		errorsMap := map[string][]string{"sourceDc": {}, "targetDc": {}}
		_, _, err := recognizeTargetAndSourceDc(test.cr, test.properties, &errorsMap)
		if err == nil {
			t.Errorf("Method must return error!")
		} else {
			if !util.EqualComplexMaps(errorsMap, test.expectedErrorMap) {
				t.Errorf("%s - unexpected errorsMap; expected - %s ", errorsMap, test.expectedErrorMap)
			}
		}
	}
}

func TestGetUpdateContent(t *testing.T) {
	topics := "topic2,topic3"

	singleTargetDc := []string{"dc1"}
	singleSourceDc := []string{"dc2"}
	singleProperties := make(map[string]interface{})
	singleProperties["dc2->dc1.enabled"] = "false"
	singleProperties["dc2->dc1.topics"] = "topic1"
	expectedSingleTurnOnReplicationPairs := []string{"dc2->dc1.enabled"}
	expectedSingleUpdateTopicLines := map[string]string{"dc2->dc1.topics": "topic1,topic2,topic3"}

	withoutTopicsProperties := make(map[string]interface{})
	withoutTopicsProperties["dc2->dc1.enabled"] = "false"
	newLinesSlice := []string{"dc2->dc1.topics = topic2,topic3"}
	notEmptyExpectedNewLines := strings.Join(newLinesSlice, "\n")

	multiSourceDc := []string{"dc2", "dc3"}
	multiNewLinesSlice := []string{"dc2->dc1.topics = topic2,topic3", "dc3->dc1.enabled = true", "dc3->dc1.topics = topic2,topic3"}
	multiExpectedNewLines := strings.Join(multiNewLinesSlice, "\n")

	tests := []struct {
		targetDc                       []string
		sourceDc                       []string
		properties                     map[string]interface{}
		topics                         string
		expectedTurnOnReplicationPairs []string
		expectedUpdateTopicLines       map[string]string
		expectedNewLines               string
	}{
		{singleTargetDc, singleSourceDc, singleProperties, topics, expectedSingleTurnOnReplicationPairs, expectedSingleUpdateTopicLines, ""},
		{singleTargetDc, singleSourceDc, withoutTopicsProperties, topics, expectedSingleTurnOnReplicationPairs, nil, notEmptyExpectedNewLines},
		{singleTargetDc, multiSourceDc, withoutTopicsProperties, topics, expectedSingleTurnOnReplicationPairs, nil, multiExpectedNewLines},
	}

	for _, test := range tests {
		turnOnReplicationPairs, updateTopicLines, newLines := getUpdatedContent(test.targetDc, test.sourceDc, test.properties, test.topics)
		if !util.EqualSlices(turnOnReplicationPairs, test.expectedTurnOnReplicationPairs) {
			t.Errorf("%s - unexpected turnOnReplicationPairs list; expected - %s ", turnOnReplicationPairs, test.expectedTurnOnReplicationPairs)
		}
		if !util.EqualPlainMaps(updateTopicLines, test.expectedUpdateTopicLines) {
			t.Errorf("%s - unexpected updateTopicLines list; expected - %s ", updateTopicLines, test.expectedUpdateTopicLines)
		}
		if newLines != test.expectedNewLines {
			t.Errorf("%s - unexpected newLines list; expected - %s ", newLines, test.expectedNewLines)
		}
	}
}
