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
	"github.com/Netcracker/qubership-kafka/operator/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReplicationAuditor_mustBeTopicReplicated(t *testing.T) {
	testTopic := "topic-1"

	tests := []struct {
		topic          string
		allowRegExp    []string
		blockRegExp    []string
		expectedResult bool
	}{
		{testTopic, nil, nil, true},
		{testTopic, []string{"topic-1"}, nil, true},
		{testTopic, nil, []string{"topic-1"}, false},
		{testTopic, []string{".*"}, nil, true},
		{testTopic, []string{"topic-1", "topic-2"}, []string{"dc2\\..*"}, true},
		{testTopic, []string{"topic-1"}, []string{"topic-1"}, false},
		{testTopic, []string{""}, []string{""}, false}, //TODO: check behaviour
	}
	for _, test := range tests {
		actualResult := mustBeTopicReplicated(test.topic, test.allowRegExp, test.blockRegExp)
		if actualResult != test.expectedResult {
			t.Errorf("expected result is %t, but actual result is %t. "+
				"Allow regular expression is %s, block regular expression is %s",
				test.expectedResult, actualResult, test.allowRegExp, test.blockRegExp)
		}
	}
}

func TestReplicationAuditor_flushOut(t *testing.T) {
	bigetActiveMap := map[string]int64{
		"topic-1-0": 2,
		"topic-2-0": 3,
		"topic-2-1": 1,
		"topic-3-0": 2,
	}
	standbyMap := map[string]int64{
		"topic-1-0": 2,
		"topic-2-0": 3,
		"topic-2-1": 1,
	}
	lessActiveMap := map[string]int64{
		"topic-1-0": 2,
		"topic-2-0": 3,
	}
	doesNotEqualActiveMap := map[string]int64{
		"topic-1-0": 1,
		"topic-2-0": 2,
		"topic-2-1": 1,
	}
	differenceMap := map[string]int64{
		"topic-3-0": 2,
	}
	emptyMap := make(map[string]int64)

	tests := []struct {
		activeMap      map[string]int64
		standbyMap     map[string]int64
		expectedResult map[string]int64
	}{
		{bigetActiveMap, standbyMap, differenceMap},
		{lessActiveMap, standbyMap, emptyMap},
		{doesNotEqualActiveMap, standbyMap, emptyMap},
		{standbyMap, standbyMap, emptyMap},
	}
	for _, test := range tests {
		actualResult := flushOut(test.activeMap, test.standbyMap)
		if !util.EqualIntMaps(actualResult, test.expectedResult) {
			t.Errorf("expected result is %v but %v given", test.expectedResult, actualResult)
		}
	}
}

func TestReplicationAuditor_matchTopic(t *testing.T) {
	value, err := matchTopic("test_topic.*", "test_topic")
	assert.Nil(t, err)
	assert.True(t, value)

	value, err = matchTopic("test_topic.*", "test_topic_123")
	assert.Nil(t, err)
	assert.True(t, value)

	value, err = matchTopic(".*test_topic.*", "other_test_topic_1")
	assert.Nil(t, err)
	assert.True(t, value)

	value, err = matchTopic("test_topic.*1", "test_topic_123")
	assert.Nil(t, err)
	assert.False(t, value)

	value, err = matchTopic("test_topic.*", "other_test_topic")
	assert.Nil(t, err)
	assert.False(t, value)
}
