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

package akhqconfig

import (
	"fmt"
	akhqconfigv1 "github.com/Netcracker/qubership-kafka/operator/api/v1"
	"github.com/Netcracker/qubership-kafka/operator/util"
	"testing"
)

var r = &AkhqConfigReconciler{}
var suffixTopic1 = util.StringHash("topic1")
var suffixTopic2 = util.StringHash("topic2")
var suffixTopic3 = util.StringHash("topic3")
var suffixTopic4 = util.StringHash("topic4")

func areSlicesEqual(actual, expected []TopicMapping) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i := 0; i < len(expected); i++ {
		if actual[i] != expected[i] {
			return false
		}
	}
	return true
}

func areMapsEqual(actual, expected map[string]string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for key, value := range expected {
		actualValue, ok := actual[key]
		if !ok || value != actualValue {
			return false
		}
	}
	return true
}

func areSlicesEqualWithoutOrder(actual, expected []TopicMapping) bool {
	if len(actual) != len(expected) {
		return false
	}
	for _, value := range actual {
		isFound := false
		for i := 0; i < len(expected); i++ {
			if value == expected[i] {
				copy(expected[i:], expected[i+1:])
				expected[len(expected)-1] = TopicMapping{}
				expected = expected[:len(expected)-1]
				isFound = true
				break
			}
		}
		if !isFound {
			return false
		}
	}
	return true
}

func Test_deserializationConfigFullUpdate_addToEmpty(t *testing.T) {
	existingConfigs := make([]TopicMapping, 0, 1)
	crConfigs := map[string]akhqconfigv1.Config{
		"topic1": akhqconfigv1.Config{
			TopicRegex:           "topic1",
			MessageType:          "org.qubership.Employee",
			DescriptorFileBase64: "cjhsAkjdsa==",
		},
	}
	suffix := util.StringHash("topic1")
	actualConfigs, actualMapToUpdate := r.fullUpdateDeserializationConfig(existingConfigs, crConfigs, "akhq-kafka-service")
	expectedConfigs := []TopicMapping{
		{
			TopicRegex:       "topic1",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffix),
		},
	}
	expectedMapToUpdate := map[string]string{
		fmt.Sprintf("akhq-kafka-service-%s", suffix): "cjhsAkjdsa==",
	}

	if !areSlicesEqualWithoutOrder(actualConfigs, expectedConfigs) {
		t.Errorf("actualConfigs - %v are not equal expectedConfigs - %v", actualConfigs, expectedConfigs)
	}

	if !areMapsEqual(actualMapToUpdate, expectedMapToUpdate) {
		t.Errorf("actualMapToUpdate - %v is not equal expectedMapToUpdate - %v", actualMapToUpdate, expectedMapToUpdate)
	}
}

func Test_deserializationConfigFullUpdate_addToEnd(t *testing.T) {
	existingConfigs := []TopicMapping{
		{
			TopicRegex:       "topic1",
			Name:             "akhq-kafka-service-1",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic1),
		},
		{
			TopicRegex:       "topic2",
			Name:             "akhq-kafka-service-1",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic2),
		},
	}

	crConfigs := map[string]akhqconfigv1.Config{
		"topic3": {
			TopicRegex:           "topic3",
			MessageType:          "org.qubership.Employee",
			DescriptorFileBase64: "cjhsdfjrAkjdsa==",
		},
		"topic4": {
			TopicRegex:           "topic4",
			MessageType:          "org.qubership.Employee",
			DescriptorFileBase64: "cjhfjrAkjdsa==",
		},
	}

	actualConfigs, actualMapToUpdate := r.fullUpdateDeserializationConfig(existingConfigs, crConfigs, "akhq-kafka-service")

	expectedConfigs := []TopicMapping{
		{
			TopicRegex:       "topic1",
			Name:             "akhq-kafka-service-1",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic1),
		},
		{
			TopicRegex:       "topic2",
			Name:             "akhq-kafka-service-1",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic2),
		},
		{
			TopicRegex:       "topic3",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic3),
		},
		{
			TopicRegex:       "topic4",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic4),
		},
	}

	expectedMapToUpdate := map[string]string{
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic3): "cjhsdfjrAkjdsa==",
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic4): "cjhfjrAkjdsa==",
	}

	if !areSlicesEqualWithoutOrder(actualConfigs, expectedConfigs) {
		t.Errorf("actualConfigs - %v are not equal expectedConfigs - %v", actualConfigs, expectedConfigs)
	}

	if !areMapsEqual(actualMapToUpdate, expectedMapToUpdate) {
		t.Errorf("actualMapToUpdate - %v is not equal expectedMapToUpdate - %v", actualMapToUpdate, expectedMapToUpdate)
	}
}

func Test_deserializationConfigFullUpdate_updateWithDeleting(t *testing.T) {
	existingConfigs := []TopicMapping{
		{
			TopicRegex:       "topic1",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic1),
		},
		{
			TopicRegex:       "topic2",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic2),
		},
	}

	crConfigs := map[string]akhqconfigv1.Config{
		"topic3": {
			TopicRegex:           "topic3",
			MessageType:          "org.qubership.Employee",
			DescriptorFileBase64: "cjhsdfjrAkjdsa==",
		},
		"topic4": {
			TopicRegex:           "topic4",
			MessageType:          "org.qubership.Employee",
			DescriptorFileBase64: "cjhfjrAkjdsa==",
		},
	}

	actualConfigs, actualMapToUpdate := r.fullUpdateDeserializationConfig(existingConfigs, crConfigs, "akhq-kafka-service")

	expectedConfigs := []TopicMapping{
		{
			TopicRegex:       "topic3",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic3),
		},
		{
			TopicRegex:       "topic4",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic4),
		},
	}

	expectedMapToUpdate := map[string]string{
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic1): "",
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic2): "",
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic3): "cjhsdfjrAkjdsa==",
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic4): "cjhfjrAkjdsa==",
	}

	if !areSlicesEqualWithoutOrder(actualConfigs, expectedConfigs) {
		t.Errorf("actualConfigs - %v are not equal expectedConfigs - %v", actualConfigs, expectedConfigs)
	}

	if !areMapsEqual(actualMapToUpdate, expectedMapToUpdate) {
		t.Errorf("actualMapToUpdate - %v is not equal expectedMapToUpdate - %v", actualMapToUpdate, expectedMapToUpdate)
	}
}

func Test_deserializationConfigFullUpdate_updateExistedOnly(t *testing.T) {
	existingConfigs := []TopicMapping{
		{
			TopicRegex:       "topic1",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic1),
		},
		{
			TopicRegex:       "topic2",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic2),
		},
	}

	crConfigs := map[string]akhqconfigv1.Config{
		"topic1": {
			TopicRegex:           "topic1",
			MessageType:          "org.qubership.Employee",
			DescriptorFileBase64: "cjhsdfjrAkjdsa==",
		},
		"topic2": {
			TopicRegex:           "topic2",
			MessageType:          "org.qubership.Employee",
			DescriptorFileBase64: "cjhfjrAkjdsa==",
		},
	}

	actualConfigs, actualMapToUpdate := r.fullUpdateDeserializationConfig(existingConfigs, crConfigs, "akhq-kafka-service")

	expectedConfigs := []TopicMapping{
		{
			TopicRegex:       "topic1",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic1),
		},
		{
			TopicRegex:       "topic2",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic2),
		},
	}

	expectedMapToUpdate := map[string]string{
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic1): "cjhsdfjrAkjdsa==",
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic2): "cjhfjrAkjdsa==",
	}

	if !areSlicesEqualWithoutOrder(actualConfigs, expectedConfigs) {
		t.Errorf("actualConfigs - %v are not equal expectedConfigs - %v", actualConfigs, expectedConfigs)
	}

	if !areMapsEqual(actualMapToUpdate, expectedMapToUpdate) {
		t.Errorf("actualMapToUpdate - %v is not equal expectedMapToUpdate - %v", actualMapToUpdate, expectedMapToUpdate)
	}
}

func Test_deserializationConfigFullUpdate_updateAndInsert(t *testing.T) {
	existingConfigs := []TopicMapping{
		{
			TopicRegex:       "topic1",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic1),
		},
		{
			TopicRegex:       "topic2",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic2),
		},
		{
			TopicRegex:       "topic4",
			Name:             "akhq-kafka-service-1",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic4),
		},
	}

	crConfigs := map[string]akhqconfigv1.Config{
		"topic1": {
			TopicRegex:           "topic1",
			MessageType:          "org.qubership.Employee",
			DescriptorFileBase64: "cjhsdfjrAkjdsa==",
		},
		"topic2": {
			TopicRegex:           "topic2",
			MessageType:          "org.qubership.Employee",
			DescriptorFileBase64: "cjhfjrAkjdsa==",
		},
		"topic3": {
			TopicRegex:           "topic3",
			MessageType:          "org.qubership.Employee",
			DescriptorFileBase64: "fgtjhgswDsfjrAkjdsa==",
		},
	}

	actualConfigs, actualMapToUpdate := r.fullUpdateDeserializationConfig(existingConfigs, crConfigs, "akhq-kafka-service")

	expectedConfigs := []TopicMapping{
		{
			TopicRegex:       "topic1",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic1),
		},
		{
			TopicRegex:       "topic2",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic2),
		},
		{
			TopicRegex:       "topic3",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic3),
		},
		{
			TopicRegex:       "topic4",
			Name:             "akhq-kafka-service-1",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic4),
		},
	}

	expectedMapToUpdate := map[string]string{
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic1): "cjhsdfjrAkjdsa==",
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic2): "cjhfjrAkjdsa==",
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic3): "fgtjhgswDsfjrAkjdsa==",
	}

	if !areSlicesEqual(actualConfigs, expectedConfigs) {
		t.Errorf("actualConfigs - %v are not equal expectedConfigs - %v", actualConfigs, expectedConfigs)
	}

	if !areMapsEqual(actualMapToUpdate, expectedMapToUpdate) {
		t.Errorf("actualMapToUpdate - %v is not equal expectedMapToUpdate - %v", actualMapToUpdate, expectedMapToUpdate)
	}
}

func Test_deleteDeserializationConfig_crOnly(t *testing.T) {
	existingConfigs := []TopicMapping{
		{
			TopicRegex:       "topic1",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic1),
		},
		{
			TopicRegex:       "topic2",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic2),
		},
		{
			TopicRegex:       "topic4",
			Name:             "akhq-kafka-service-1",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic4),
		},
	}

	actualConfigs, actualMapToDelete := r.deleteDeserializationConfig(existingConfigs, "akhq-kafka-service")

	expectedConfigs := []TopicMapping{
		{
			TopicRegex:       "topic4",
			Name:             "akhq-kafka-service-1",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic4),
		},
	}

	expectedMapToDelete := map[string]string{
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic1): "",
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic2): "",
	}

	if !areSlicesEqual(actualConfigs, expectedConfigs) {
		t.Errorf("actualConfigs - %v are not equal expectedConfigs - %v", actualConfigs, expectedConfigs)
	}

	if !areMapsEqual(actualMapToDelete, expectedMapToDelete) {
		t.Errorf("actualMapToUpdate - %v is not equal expectedMapToUpdate - %v", actualMapToDelete, expectedMapToDelete)
	}
}

func Test_deleteDeserializationConfig_crAndGarbage(t *testing.T) {
	existingConfigs := []TopicMapping{
		{
			TopicRegex:       "topic1",
			Name:             "akhq-kafka-service",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic1),
		},
		{
			TopicRegex:           "topic2",
			Name:                 "akhq-kafka-service",
			KeyMessageType:       "",
			ValueMessageType:     "org.qubership.Employee",
			DescriptorFileBase64: "jdshfGGGjjhxxs==",
		},
		{
			TopicRegex:       "topic4",
			Name:             "akhq-kafka-service-1",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic4),
		},
	}

	actualConfigs, actualMapToDelete := r.deleteDeserializationConfig(existingConfigs, "akhq-kafka-service")

	expectedConfigs := []TopicMapping{
		{
			TopicRegex:       "topic4",
			Name:             "akhq-kafka-service-1",
			KeyMessageType:   "",
			ValueMessageType: "org.qubership.Employee",
			DescriptorFile:   fmt.Sprintf("descs/akhq-kafka-service-%s", suffixTopic4),
		},
	}

	expectedMapToDelete := map[string]string{
		fmt.Sprintf("akhq-kafka-service-%s", suffixTopic1): "",
	}

	if !areSlicesEqual(actualConfigs, expectedConfigs) {
		t.Errorf("actualConfigs - %v are not equal expectedConfigs - %v", actualConfigs, expectedConfigs)
	}

	if !areMapsEqual(actualMapToDelete, expectedMapToDelete) {
		t.Errorf("actualMapToUpdate - %v is not equal expectedMapToUpdate - %v", actualMapToDelete, expectedMapToDelete)
	}
}

func Test_validate(t *testing.T) {
	tests := []struct {
		Configs             []akhqconfigv1.Config
		ExpectedBool        bool
		ExpectedDescription string
	}{
		{Configs: []akhqconfigv1.Config{
			{
				TopicRegex:           "topic1",
				MessageType:          "org.qubership.Message",
				DescriptorFileBase64: "CuMCCghteS5wcm90bxIIdHV0b3JpYWwi/AEKBlBlcnNvbhISCgRuYW1lGAEgASgJUgRuYW1lEg4KAmlkGAIgASgFUgJpZBIUCgVlbWFpbBgDIAEoCVIFZW1haWwSNAoGcGhvbmVzGAQgAygLMhwudHV0b3JpYWwuUGVyc29uLlBob25lTnVtYmVyUgZwaG9uZXMaVQoLUGhvbmVOdW1iZXISFgoGbnVtYmVyGAEgASgJUgZudW1iZXISLgoEdHlwZRgCIAEoDjIaLnR1dG9yaWFsLlBlcnNvbi5QaG9uZVR5cGVSBHR5cGUiKwoJUGhvbmVUeXBlEgoKBk1PQklMRRAAEggKBEhPTUUQARIICgRXT1JLEAIiNwoLQWRkcmVzc0Jvb2sSKAoGcGVvcGxlGAEgAygLMhAudHV0b3JpYWwuUGVyc29uUgZwZW9wbGVCDVoLLi9yZXNvdXJjZXNiBnByb3RvMw==",
			},
		},
			ExpectedBool:        true,
			ExpectedDescription: "",
		},

		{Configs: []akhqconfigv1.Config{
			{
				TopicRegex:           "topic1",
				MessageType:          "org.qubership.Message",
				DescriptorFileBase64: "jfhdghgrger",
			},
		},
			ExpectedBool:        false,
			ExpectedDescription: fmt.Sprintf(decodingValidationError, "topic1"),
		},

		{Configs: []akhqconfigv1.Config{
			{
				TopicRegex:           "topic1",
				MessageType:          "org.qubership.Message",
				DescriptorFileBase64: "CuMCCghteS5wcm90bxIIdHV0b3JpYWwi/AEKBlBlcnNvbhISCgRuYW1lGAEgASgJUgRuYW1lEg4KAmlkGAIgASgFUgJpZBIUCgVlbWFpbBgDIAEoCVIFZW1haWwSNAoGcGhvbmVzGAQgAygLMhwudHV0b3JpYWwuUGVyc29uLlBob25lTnVtYmVyUgZwaG9uZXMaVQoLUGhvbmVOdW1iZXISFgoGbnVtYmVyGAEgASgJUgZudW1iZXISLgoEdHlwZRgCIAEoDjIaLnR1dG9yaWFsLlBlcnNvbi5QaG9uZVR5cGVSBHR5cGUiKwoJUGhvbmVUeXBlEgoKBk1PQklMRRAAEggKBEhPTUUQARIICgRXT1JLEAIiNwoLQWRkcmVzc0Jvb2sSKAoGcGVvcGxlGAEgAygLMhAudHV0b3JpYWwuUGVyc29uUgZwZW9wbGVCDVoLLi9yZXNvdXJjZXNiBnByb3RvMw==",
			},
			{
				TopicRegex:           "topic1",
				MessageType:          "org.qubership.Employee",
				DescriptorFileBase64: "CuMCCghteS5wcm90bxIIdHV0b3JpYWwi/AEKBlBlcnNvbhISCgRuYW1lGAEgASgJUgRuYW1lEg4KAmlkGAIgASgFUgJpZBIUCgVlbWFpbBgDIAEoCVIFZW1haWwSNAoGcGhvbmVzGAQgAygLMhwudHV0b3JpYWwuUGVyc29uLlBob25lTnVtYmVyUgZwaG9uZXMaVQoLUGhvbmVOdW1iZXISFgoGbnVtYmVyGAEgASgJUgZudW1iZXISLgoEdHlwZRgCIAEoDjIaLnR1dG9yaWFsLlBlcnNvbi5QaG9uZVR5cGVSBHR5cGUiKwoJUGhvbmVUeXBlEgoKBk1PQklMRRAAEggKBEhPTUUQARIICgRXT1JLEAIiNwoLQWRkcmVzc0Jvb2sSKAoGcGVvcGxlGAEgAygLMhAudHV0b3JpYWwuUGVyc29uUgZwZW9wbGVCDVoLLi9yZXNvdXJjZXNiBnByb3RvMw==",
			},
		},
			ExpectedBool:        false,
			ExpectedDescription: fmt.Sprintf(duplicateValidationError, "topic1"),
		},
	}

	for _, test := range tests {
		boolRes, description := r.validateCrConfig(test.Configs)
		if test.ExpectedBool != boolRes || test.ExpectedDescription != description {
			t.Errorf("given boolRes - [%v], given description - [%v]; expected boolRes - [%v], expected description - [%v]",
				boolRes, description, test.ExpectedBool, test.ExpectedDescription)
		}
	}
}
