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
	"github.com/Netcracker/qubership-kafka/api/kmm"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	sourceClusterName = "cluster1"
	targetClusterName = "cluster2"
)

func TestAddTransformationPropertiesWhenTransformationIsNil(t *testing.T) {
	var replicationConfig []string
	transformationConfigurator := NewTransformationConfigurator(nil, "")
	replicationConfig = transformationConfigurator.AddTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, false)
	assert.Empty(t, replicationConfig)
}

func TestAddTransformationPropertiesWhenTransformationContainsOnlyTransforms(t *testing.T) {
	var replicationConfig []string
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "test.*",
				},
			},
			{
				Name: "Filter",
				Type: "org.apache.kafka.connect.transforms.Filter",
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "")
	replicationConfig = transformationConfigurator.AddTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"cluster1->cluster2.transforms.HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.HeaderFilter.topics = test.*",
		"cluster1->cluster2.transforms.Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms = Filter,HeaderFilter",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestAddTransformationPropertiesWithNamePrefixWhenTransformationContainsOnlyTransforms(t *testing.T) {
	var replicationConfig []string
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "test.*",
				},
			},
			{
				Name: "Filter",
				Type: "org.apache.kafka.connect.transforms.Filter",
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "test_")
	replicationConfig = transformationConfigurator.AddTransformationProperties(replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.test_HeaderFilter.topics = test.*",
		"cluster1->cluster2.transforms.test_Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms = test_Filter,test_HeaderFilter",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestAddTransformationPropertiesWhenTransformationContainsTransformsAndPredicates(t *testing.T) {
	var replicationConfig []string
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name:      "HeaderFilter",
				Type:      "org.qubership.kafka.mirror.extension.HeaderFilter",
				Predicate: "HasHeaderKey",
				Negate:    boolPointer(true),
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
				},
			},
			{
				Name:      "Filter",
				Type:      "org.apache.kafka.connect.transforms.Filter",
				Predicate: "TopicNameMatches",
				Negate:    boolPointer(false),
			},
		},
		Predicates: []kmm.Predicate{
			{
				Name: "HasHeaderKey",
				Type: "org.apache.kafka.connect.transforms.predicates.HasHeaderKey",
				Params: map[string]string{
					"name": "heartbeat",
				},
			},
			{
				Name: "TopicNameMatches",
				Type: "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
				Params: map[string]string{
					"pattern": "test.*",
				},
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "")
	replicationConfig = transformationConfigurator.AddTransformationProperties(replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"cluster1->cluster2.transforms.HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.HeaderFilter.predicate = HasHeaderKey",
		"cluster1->cluster2.transforms.HeaderFilter.negate = true",
		"cluster1->cluster2.transforms.HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.Filter.predicate = TopicNameMatches",
		"cluster1->cluster2.transforms.Filter.negate = false",
		"cluster1->cluster2.predicates.HasHeaderKey.type = org.apache.kafka.connect.transforms.predicates.HasHeaderKey",
		"cluster1->cluster2.predicates.HasHeaderKey.name = heartbeat",
		"cluster1->cluster2.predicates.TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.pattern = test.*",
		"cluster1->cluster2.transforms = Filter,HeaderFilter",
		"cluster1->cluster2.predicates = HasHeaderKey,TopicNameMatches",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestAddTransformationPropertiesWithNamePrefixWhenTransformationContainsTransformsAndPredicates(t *testing.T) {
	var replicationConfig []string
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name:      "HeaderFilter",
				Type:      "org.qubership.kafka.mirror.extension.HeaderFilter",
				Predicate: "HasHeaderKey",
				Negate:    boolPointer(true),
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
				},
			},
			{
				Name:      "Filter",
				Type:      "org.apache.kafka.connect.transforms.Filter",
				Predicate: "TopicNameMatches",
				Negate:    boolPointer(false),
			},
		},
		Predicates: []kmm.Predicate{
			{
				Name: "HasHeaderKey",
				Type: "org.apache.kafka.connect.transforms.predicates.HasHeaderKey",
				Params: map[string]string{
					"name": "heartbeat",
				},
			},
			{
				Name: "TopicNameMatches",
				Type: "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
				Params: map[string]string{
					"pattern": "test.*",
				},
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "test_")
	replicationConfig = transformationConfigurator.AddTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.predicate = test_HasHeaderKey",
		"cluster1->cluster2.transforms.test_HeaderFilter.negate = true",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.test_Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.test_Filter.predicate = test_TopicNameMatches",
		"cluster1->cluster2.transforms.test_Filter.negate = false",
		"cluster1->cluster2.predicates.test_HasHeaderKey.type = org.apache.kafka.connect.transforms.predicates.HasHeaderKey",
		"cluster1->cluster2.predicates.test_HasHeaderKey.name = heartbeat",
		"cluster1->cluster2.predicates.test_TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.pattern = test.*",
		"cluster1->cluster2.transforms = test_Filter,test_HeaderFilter",
		"cluster1->cluster2.predicates = test_HasHeaderKey,test_TopicNameMatches",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestAddTransformationPropertiesWhenTransformationContainsParamsWithReplicationPrefix(t *testing.T) {
	var replicationConfig []string
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "${replication_prefix}test.*",
				},
			},
			{
				Name:      "Filter",
				Type:      "org.apache.kafka.connect.transforms.Filter",
				Predicate: "TopicNameMatches",
			},
		},
		Predicates: []kmm.Predicate{
			{
				Name: "TopicNameMatches",
				Type: "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
				Params: map[string]string{
					"pattern": "${replication_prefix}test.*",
				},
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "")
	replicationConfig = transformationConfigurator.AddTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"cluster1->cluster2.transforms.HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.HeaderFilter.topics = cluster1.test.*",
		"cluster1->cluster2.transforms.Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.Filter.predicate = TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.pattern = cluster1.test.*",
		"cluster1->cluster2.transforms = Filter,HeaderFilter",
		"cluster1->cluster2.predicates = TopicNameMatches",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestAddTransformationPropertiesWithNamePrefixWhenTransformationContainsParamsWithReplicationPrefix(t *testing.T) {
	var replicationConfig []string
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "${replication_prefix}test.*",
				},
			},
			{
				Name:      "Filter",
				Type:      "org.apache.kafka.connect.transforms.Filter",
				Predicate: "TopicNameMatches",
			},
		},
		Predicates: []kmm.Predicate{
			{
				Name: "TopicNameMatches",
				Type: "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
				Params: map[string]string{
					"pattern": "${replication_prefix}test.*",
				},
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "test_")
	replicationConfig = transformationConfigurator.AddTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.test_HeaderFilter.topics = cluster1.test.*",
		"cluster1->cluster2.transforms.test_Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.test_Filter.predicate = test_TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.pattern = cluster1.test.*",
		"cluster1->cluster2.transforms = test_Filter,test_HeaderFilter",
		"cluster1->cluster2.predicates = test_TopicNameMatches",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestAddTransformationPropertiesWhenTransformationContainsParamsWithReplicationPrefixAndIdentityReplicationIsEnabled(t *testing.T) {
	var replicationConfig []string
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "${replication_prefix}test.*",
				},
			},
			{
				Name:      "Filter",
				Type:      "org.apache.kafka.connect.transforms.Filter",
				Predicate: "TopicNameMatches",
			},
		},
		Predicates: []kmm.Predicate{
			{
				Name: "TopicNameMatches",
				Type: "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
				Params: map[string]string{
					"pattern": "${replication_prefix}test.*",
				},
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "")
	replicationConfig = transformationConfigurator.AddTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, true)
	expected := []string{
		"cluster1->cluster2.transforms.HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.HeaderFilter.topics = test.*",
		"cluster1->cluster2.transforms.Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.Filter.predicate = TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.pattern = test.*",
		"cluster1->cluster2.transforms = Filter,HeaderFilter",
		"cluster1->cluster2.predicates = TopicNameMatches",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestAddTransformationPropertiesWithNamePrefixWhenTransformationContainsParamsWithReplicationPrefixAndIdentityReplicationIsEnabled(t *testing.T) {
	var replicationConfig []string
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "${replication_prefix}test.*",
				},
			},
			{
				Name:      "Filter",
				Type:      "org.apache.kafka.connect.transforms.Filter",
				Predicate: "TopicNameMatches",
			},
		},
		Predicates: []kmm.Predicate{
			{
				Name: "TopicNameMatches",
				Type: "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
				Params: map[string]string{
					"pattern": "${replication_prefix}test.*",
				},
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "test_")
	replicationConfig = transformationConfigurator.AddTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, true)
	expected := []string{
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.test_HeaderFilter.topics = test.*",
		"cluster1->cluster2.transforms.test_Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.test_Filter.predicate = test_TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.pattern = test.*",
		"cluster1->cluster2.transforms = test_Filter,test_HeaderFilter",
		"cluster1->cluster2.predicates = test_TopicNameMatches",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestUpdateTransformationPropertiesWhenTransformationIsNil(t *testing.T) {
	replicationConfig := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.topics = test.*",
		"topics.blacklist = cluster1.test.*",
	}
	transformationConfigurator := NewTransformationConfigurator(nil, "")
	replicationConfig = transformationConfigurator.UpdateTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, false)

	expected := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.topics = test.*",
		"topics.blacklist = cluster1.test.*",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestUpdateTransformationPropertiesWhenTransformationContainsOnlyTransformsAndReplicationConfigDoesNotContainTransformationProperties(t *testing.T) {
	replicationConfig := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "test.*",
				},
			},
			{
				Name: "Filter",
				Type: "org.apache.kafka.connect.transforms.Filter",
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "")
	replicationConfig = transformationConfigurator.UpdateTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.HeaderFilter.topics = test.*",
		"cluster1->cluster2.transforms.Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms = Filter,HeaderFilter",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestUpdateTransformationPropertiesWhenTransformationContainsOnlyTransformsAndReplicationConfigContainsTransformationProperties(t *testing.T) {
	replicationConfig := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.HeaderFilter.filter.type = include",
		"cluster1->cluster2.transforms.HeaderFilter.headers = heartbeat",
		"cluster1->cluster2.transforms.HeaderFilter.topics = .*",
		"cluster1->cluster2.transforms.DropHeaders.type = org.apache.kafka.connect.transforms.DropHeaders",
		"cluster1->cluster2.transforms.DropHeaders.headers = heartbeat",
		"cluster1->cluster2.transforms = HeaderFilter,DropHeaders",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "test.*",
				},
			},
			{
				Name: "Filter",
				Type: "org.apache.kafka.connect.transforms.Filter",
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "")
	replicationConfig = transformationConfigurator.UpdateTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.HeaderFilter.topics = test.*",
		"cluster1->cluster2.transforms.Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms = Filter,HeaderFilter",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestUpdateTransformationPropertiesWithNamePrefixWhenTransformationContainsOnlyTransformsAndReplicationConfigDoesNotContainTransformationProperties(t *testing.T) {
	replicationConfig := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "test.*",
				},
			},
			{
				Name: "Filter",
				Type: "org.apache.kafka.connect.transforms.Filter",
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "test_")
	replicationConfig = transformationConfigurator.UpdateTransformationProperties(replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.test_HeaderFilter.topics = test.*",
		"cluster1->cluster2.transforms.test_Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms = test_Filter,test_HeaderFilter",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestUpdateTransformationPropertiesWithNamePrefixWhenTransformationContainsOnlyTransformsAndReplicationConfigContainsTransformationProperties(t *testing.T) {
	replicationConfig := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = include",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = heartbeat",
		"cluster1->cluster2.transforms.test_HeaderFilter.topics = .*",
		"cluster1->cluster2.transforms.test_DropHeaders.type = org.apache.kafka.connect.transforms.DropHeaders",
		"cluster1->cluster2.transforms.test_DropHeaders.headers = heartbeat",
		"cluster1->cluster2.transforms = test_HeaderFilter,test_DropHeaders",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "test.*",
				},
			},
			{
				Name: "Filter",
				Type: "org.apache.kafka.connect.transforms.Filter",
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "test_")
	replicationConfig = transformationConfigurator.UpdateTransformationProperties(replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.test_HeaderFilter.topics = test.*",
		"cluster1->cluster2.transforms.test_Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms = test_Filter,test_HeaderFilter",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestUpdateTransformationPropertiesWhenTransformationContainsTransformsAndPredicatesAndReplicationConfigDoesNotContainTransformationProperties(t *testing.T) {
	replicationConfig := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name:      "HeaderFilter",
				Type:      "org.qubership.kafka.mirror.extension.HeaderFilter",
				Predicate: "HasHeaderKey",
				Negate:    boolPointer(true),
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
				},
			},
			{
				Name:      "Filter",
				Type:      "org.apache.kafka.connect.transforms.Filter",
				Predicate: "TopicNameMatches",
				Negate:    boolPointer(false),
			},
		},
		Predicates: []kmm.Predicate{
			{
				Name: "HasHeaderKey",
				Type: "org.apache.kafka.connect.transforms.predicates.HasHeaderKey",
				Params: map[string]string{
					"name": "heartbeat",
				},
			},
			{
				Name: "TopicNameMatches",
				Type: "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
				Params: map[string]string{
					"pattern": "test.*",
				},
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "")
	replicationConfig = transformationConfigurator.UpdateTransformationProperties(replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.HeaderFilter.predicate = HasHeaderKey",
		"cluster1->cluster2.transforms.HeaderFilter.negate = true",
		"cluster1->cluster2.transforms.HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.Filter.predicate = TopicNameMatches",
		"cluster1->cluster2.transforms.Filter.negate = false",
		"cluster1->cluster2.predicates.HasHeaderKey.type = org.apache.kafka.connect.transforms.predicates.HasHeaderKey",
		"cluster1->cluster2.predicates.HasHeaderKey.name = heartbeat",
		"cluster1->cluster2.predicates.TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.pattern = test.*",
		"cluster1->cluster2.transforms = Filter,HeaderFilter",
		"cluster1->cluster2.predicates = HasHeaderKey,TopicNameMatches",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestUpdateTransformationPropertiesWithNamePrefixWhenTransformationContainsTransformsAndPredicatesAndReplicationConfigContainsTransformationProperties(t *testing.T) {
	replicationConfig := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = include",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = heartbeat",
		"cluster1->cluster2.transforms.test_HeaderFilter.topics = .*",
		"cluster1->cluster2.transforms.test_DropHeaders.type = org.apache.kafka.connect.transforms.DropHeaders",
		"cluster1->cluster2.transforms.test_DropHeaders.headers = heartbeat",
		"cluster1->cluster2.transforms.test_DropHeaders.predicate = test_TopicNameMatches",
		"cluster1->cluster2.transforms.test_DropHeaders.negate = true",
		"cluster1->cluster2.predicates.test_TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.pattern = .*",
		"cluster1->cluster2.transforms = test_HeaderFilter,test_DropHeaders",
		"cluster1->cluster2.predicates = test_TopicNameMatches",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name:      "HeaderFilter",
				Type:      "org.qubership.kafka.mirror.extension.HeaderFilter",
				Predicate: "HasHeaderKey",
				Negate:    boolPointer(true),
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
				},
			},
			{
				Name:      "Filter",
				Type:      "org.apache.kafka.connect.transforms.Filter",
				Predicate: "TopicNameMatches",
				Negate:    boolPointer(false),
			},
		},
		Predicates: []kmm.Predicate{
			{
				Name: "HasHeaderKey",
				Type: "org.apache.kafka.connect.transforms.predicates.HasHeaderKey",
				Params: map[string]string{
					"name": "heartbeat",
				},
			},
			{
				Name: "TopicNameMatches",
				Type: "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
				Params: map[string]string{
					"pattern": "test.*",
				},
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "test_")
	replicationConfig = transformationConfigurator.UpdateTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.predicate = test_HasHeaderKey",
		"cluster1->cluster2.transforms.test_HeaderFilter.negate = true",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.test_Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.test_Filter.predicate = test_TopicNameMatches",
		"cluster1->cluster2.transforms.test_Filter.negate = false",
		"cluster1->cluster2.predicates.test_HasHeaderKey.type = org.apache.kafka.connect.transforms.predicates.HasHeaderKey",
		"cluster1->cluster2.predicates.test_HasHeaderKey.name = heartbeat",
		"cluster1->cluster2.predicates.test_TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.pattern = test.*",
		"cluster1->cluster2.transforms = test_Filter,test_HeaderFilter",
		"cluster1->cluster2.predicates = test_HasHeaderKey,test_TopicNameMatches",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestUpdateTransformationPropertiesWhenTransformationContainsParamsWithReplicationPrefix(t *testing.T) {
	replicationConfig := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.HeaderFilter.filter.type = include",
		"cluster1->cluster2.transforms.HeaderFilter.headers = heartbeat",
		"cluster1->cluster2.transforms.HeaderFilter.topics = .*",
		"cluster1->cluster2.transforms.Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.Filter.predicate = TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.pattern = .*",
		"cluster1->cluster2.transforms = Filter,HeaderFilter",
		"cluster1->cluster2.predicates = TopicNameMatches",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "${replication_prefix}test.*",
				},
			},
			{
				Name:      "Filter",
				Type:      "org.apache.kafka.connect.transforms.Filter",
				Predicate: "TopicNameMatches",
			},
		},
		Predicates: []kmm.Predicate{
			{
				Name: "TopicNameMatches",
				Type: "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
				Params: map[string]string{
					"pattern": "${replication_prefix}test.*",
				},
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "")
	replicationConfig = transformationConfigurator.UpdateTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.HeaderFilter.topics = cluster1.test.*",
		"cluster1->cluster2.transforms.Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.Filter.predicate = TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.pattern = cluster1.test.*",
		"cluster1->cluster2.transforms = Filter,HeaderFilter",
		"cluster1->cluster2.predicates = TopicNameMatches",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestUpdateTransformationPropertiesWithNamePrefixWhenTransformationContainsParamsWithReplicationPrefix(t *testing.T) {
	replicationConfig := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = include",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = heartbeat",
		"cluster1->cluster2.transforms.test_HeaderFilter.topics = .*",
		"cluster1->cluster2.transforms.test_Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.test_Filter.predicate = TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.pattern = .*",
		"cluster1->cluster2.transforms = test_Filter,test_HeaderFilter",
		"cluster1->cluster2.predicates = test_TopicNameMatches",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "${replication_prefix}test.*",
				},
			},
			{
				Name:      "Filter",
				Type:      "org.apache.kafka.connect.transforms.Filter",
				Predicate: "TopicNameMatches",
			},
		},
		Predicates: []kmm.Predicate{
			{
				Name: "TopicNameMatches",
				Type: "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
				Params: map[string]string{
					"pattern": "${replication_prefix}test.*",
				},
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "test_")
	replicationConfig = transformationConfigurator.UpdateTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, false)
	expected := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.test_HeaderFilter.topics = cluster1.test.*",
		"cluster1->cluster2.transforms.test_Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.test_Filter.predicate = test_TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.pattern = cluster1.test.*",
		"cluster1->cluster2.transforms = test_Filter,test_HeaderFilter",
		"cluster1->cluster2.predicates = test_TopicNameMatches",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestUpdateTransformationPropertiesWhenTransformationContainsParamsWithReplicationPrefixAndIdentityReplicationIsEnabled(t *testing.T) {
	replicationConfig := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.HeaderFilter.filter.type = include",
		"cluster1->cluster2.transforms.HeaderFilter.headers = heartbeat",
		"cluster1->cluster2.transforms.HeaderFilter.topics = .*",
		"cluster1->cluster2.transforms.Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.Filter.predicate = TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.pattern = .*",
		"cluster1->cluster2.transforms = Filter,HeaderFilter",
		"cluster1->cluster2.predicates = TopicNameMatches",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "${replication_prefix}test.*",
				},
			},
			{
				Name:      "Filter",
				Type:      "org.apache.kafka.connect.transforms.Filter",
				Predicate: "TopicNameMatches",
			},
		},
		Predicates: []kmm.Predicate{
			{
				Name: "TopicNameMatches",
				Type: "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
				Params: map[string]string{
					"pattern": "${replication_prefix}test.*",
				},
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "")
	replicationConfig = transformationConfigurator.UpdateTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, true)
	expected := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.HeaderFilter.topics = test.*",
		"cluster1->cluster2.transforms.Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.Filter.predicate = TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.TopicNameMatches.pattern = test.*",
		"cluster1->cluster2.transforms = Filter,HeaderFilter",
		"cluster1->cluster2.predicates = TopicNameMatches",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	assert.Equal(t, expected, replicationConfig)
}

func TestUpdateTransformationPropertiesWithNamePrefixWhenTransformationContainsParamsWithReplicationPrefixAndIdentityReplicationIsEnabled(t *testing.T) {
	replicationConfig := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = include",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = heartbeat",
		"cluster1->cluster2.transforms.test_HeaderFilter.topics = .*",
		"cluster1->cluster2.transforms.test_Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.test_Filter.predicate = test_TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.pattern = .*",
		"cluster1->cluster2.transforms = test_Filter,test_HeaderFilter",
		"cluster1->cluster2.predicates = test_TopicNameMatches",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	transformation := kmm.Transformation{
		Transforms: []kmm.Transform{
			{
				Name: "HeaderFilter",
				Type: "org.qubership.kafka.mirror.extension.HeaderFilter",
				Params: map[string]string{
					"filter.type": "exclude",
					"headers":     "messageType=heartbeat",
					"topics":      "${replication_prefix}test.*",
				},
			},
			{
				Name:      "Filter",
				Type:      "org.apache.kafka.connect.transforms.Filter",
				Predicate: "TopicNameMatches",
			},
		},
		Predicates: []kmm.Predicate{
			{
				Name: "TopicNameMatches",
				Type: "org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
				Params: map[string]string{
					"pattern": "${replication_prefix}test.*",
				},
			},
		},
	}
	transformationConfigurator := NewTransformationConfigurator(&transformation, "test_")
	replicationConfig = transformationConfigurator.UpdateTransformationProperties(
		replicationConfig, sourceClusterName, targetClusterName, true)
	expected := []string{
		"clusters = cluster1, cluster2",
		"cluster1->cluster2.enabled = true",
		"cluster1->cluster2.transforms.test_HeaderFilter.type = org.qubership.kafka.mirror.extension.HeaderFilter",
		"cluster1->cluster2.transforms.test_HeaderFilter.filter.type = exclude",
		"cluster1->cluster2.transforms.test_HeaderFilter.headers = messageType=heartbeat",
		"cluster1->cluster2.transforms.test_HeaderFilter.topics = test.*",
		"cluster1->cluster2.transforms.test_Filter.type = org.apache.kafka.connect.transforms.Filter",
		"cluster1->cluster2.transforms.test_Filter.predicate = test_TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.type = org.apache.kafka.connect.transforms.predicates.TopicNameMatches",
		"cluster1->cluster2.predicates.test_TopicNameMatches.pattern = test.*",
		"cluster1->cluster2.transforms = test_Filter,test_HeaderFilter",
		"cluster1->cluster2.predicates = test_TopicNameMatches",
		"cluster1->cluster2.topics = test.*",
		"cluster2->cluster1.enabled = true",
		"cluster2->cluster1.topics = test.*",
		"topics.blacklist = cluster1.test.*,cluster2.test.*",
	}
	assert.Equal(t, expected, replicationConfig)
}

func boolPointer(b bool) *bool {
	return &b
}
