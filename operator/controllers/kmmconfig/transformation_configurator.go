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
	"fmt"
	"github.com/Netcracker/qubership-kafka/operator/api/kmm"
	"sort"
	"strings"
)

const (
	ReplicationPolicyClassConfig = "replication.policy.class"
	IdentityReplicationPolicy    = "io.strimzi.kafka.connect.mirror.IdentityReplicationPolicy"
)

type transformationDefinition struct {
	replicationFlow      string
	transformationConfig []string
	transformNames       []string
	predicateNames       []string
}

type intWrapper struct {
	value int
}

type TransformationConfigurator struct {
	transformation           *kmm.Transformation
	transformationNamePrefix string
}

func NewTransformationConfigurator(transformation *kmm.Transformation, transformationNamePrefix string) TransformationConfigurator {
	return TransformationConfigurator{
		transformation:           transformation,
		transformationNamePrefix: transformationNamePrefix,
	}
}

func (tc TransformationConfigurator) AddTransformationProperties(
	replicationConfig []string, sourceClusterName, targetClusterName string, identityReplicationEnabled bool) []string {
	transformationDefinition := tc.getTransformationDefinition(sourceClusterName, targetClusterName, identityReplicationEnabled)
	if transformationDefinition != nil {
		replicationConfig = enrichWithTransformationProperties(replicationConfig, *transformationDefinition, nil, nil)
	}
	return replicationConfig
}

func enrichWithTransformationProperties(
	replicationConfig []string, transformationDefinition transformationDefinition, extraTransformNames []string, extraPredicateNames []string) []string {
	if len(transformationDefinition.transformationConfig) == 0 {
		return replicationConfig
	}
	replicationConfig = append(replicationConfig, transformationDefinition.transformationConfig...)
	transformNames := append([]string(nil), transformationDefinition.transformNames...)
	if extraTransformNames != nil {
		transformNames = append(transformNames, extraTransformNames...)
	}
	predicateNames := append([]string(nil), transformationDefinition.predicateNames...)
	if extraPredicateNames != nil {
		predicateNames = append(predicateNames, extraPredicateNames...)
	}
	if len(transformNames) > 0 {
		sort.Strings(transformNames)
		replicationConfig = append(replicationConfig,
			fmt.Sprintf("%s.transforms = %s", transformationDefinition.replicationFlow, strings.Join(transformNames, ",")))
	}
	if len(predicateNames) > 0 {
		sort.Strings(predicateNames)
		replicationConfig = append(replicationConfig,
			fmt.Sprintf("%s.predicates = %s", transformationDefinition.replicationFlow, strings.Join(predicateNames, ",")))
	}
	return replicationConfig
}

func (tc TransformationConfigurator) getTransformationDefinition(
	sourceClusterName, targetClusterName string, identityReplicationEnabled bool) *transformationDefinition {
	if tc.transformation == nil || len(tc.transformation.Transforms) == 0 {
		return nil
	}
	var transformationConfig []string
	var transformNames []string
	var predicateNames []string
	replicationFlow := NewReplicationFlow(sourceClusterName, targetClusterName)
	replicationPrefix := sourceClusterName + "."
	if identityReplicationEnabled {
		replicationPrefix = ""
	}
	for _, transform := range tc.transformation.Transforms {
		transformName := tc.transformationName(transform.Name)
		transformNames = append(transformNames, transformName)
		transformPrefix := fmt.Sprintf("%s.transforms.%s", replicationFlow, transformName)
		transformationConfig = append(transformationConfig,
			fmt.Sprintf("%s.type = %s", transformPrefix, transform.Type))
		if transform.Predicate != "" {
			predicateName := tc.transformationName(transform.Predicate)
			transformationConfig = append(transformationConfig,
				fmt.Sprintf("%s.predicate = %s", transformPrefix, predicateName))
		}
		if transform.Negate != nil {
			transformationConfig = append(transformationConfig,
				fmt.Sprintf("%s.negate = %v", transformPrefix, *transform.Negate))
		}
		transformationConfig = addParams(
			transformationConfig, transform.Params, transformPrefix, replicationPrefix)
	}
	for _, predicate := range tc.transformation.Predicates {
		predicateName := tc.transformationName(predicate.Name)
		predicateNames = append(predicateNames, predicateName)
		predicatePrefix := fmt.Sprintf("%s.predicates.%s", replicationFlow, predicateName)
		transformationConfig = append(transformationConfig,
			fmt.Sprintf("%s.type = %s", predicatePrefix, predicate.Type))
		transformationConfig = addParams(
			transformationConfig, predicate.Params, predicatePrefix, replicationPrefix)
	}
	return &transformationDefinition{replicationFlow, transformationConfig, transformNames, predicateNames}
}

func (tc TransformationConfigurator) UpdateTransformationProperties(
	replicationConfig []string, sourceClusterName, targetClusterName string, identityReplicationEnabled bool) []string {
	transformationDefinition := tc.getTransformationDefinition(sourceClusterName, targetClusterName, identityReplicationEnabled)
	if transformationDefinition == nil {
		return replicationConfig
	}
	var before []string
	var after []string
	var extraTransformNames []string
	var extraPredicateNames []string
	enabledProperty := fmt.Sprintf("%s.enabled = true", transformationDefinition.replicationFlow)
	transformsPropertyPrefix := fmt.Sprintf("%s.transforms = ", transformationDefinition.replicationFlow)
	predicatesPropertyPrefix := fmt.Sprintf("%s.predicates = ", transformationDefinition.replicationFlow)
	transformPrefix := fmt.Sprintf("%s.transforms.%s", transformationDefinition.replicationFlow, tc.transformationNamePrefix)
	predicatePrefix := fmt.Sprintf("%s.predicates.%s", transformationDefinition.replicationFlow, tc.transformationNamePrefix)
	var enabledPropertyPosition *intWrapper
	for i, line := range replicationConfig {
		if strings.HasPrefix(line, transformationDefinition.replicationFlow) {
			if line == enabledProperty {
				enabledPropertyPosition = &intWrapper{i}
			} else if strings.HasPrefix(line, transformsPropertyPrefix) {
				names := tc.getFilteredNames(line, transformsPropertyPrefix)
				extraTransformNames = append(extraTransformNames, names...)
				continue
			} else if strings.HasPrefix(line, predicatesPropertyPrefix) {
				names := tc.getFilteredNames(line, predicatesPropertyPrefix)
				extraPredicateNames = append(extraPredicateNames, names...)
				continue
			} else if strings.HasPrefix(line, transformPrefix) || strings.HasPrefix(line, predicatePrefix) {
				continue
			}
		}
		if enabledPropertyPosition != nil && i > enabledPropertyPosition.value {
			after = append(after, line)
		} else {
			before = append(before, line)
		}
	}
	var result []string
	result = append(result, before...)
	if enabledPropertyPosition != nil {
		result = enrichWithTransformationProperties(result, *transformationDefinition, extraTransformNames, extraPredicateNames)
	}
	result = append(result, after...)
	return result
}

func (tc TransformationConfigurator) getFilteredNames(line string, propertyPrefix string) []string {
	var names []string
	for _, value := range strings.Split(line[len(propertyPrefix):], ",") {
		name := strings.TrimSpace(value)
		if !strings.HasPrefix(name, tc.transformationNamePrefix) {
			names = append(names, name)
		}
	}
	return names
}

func (tc TransformationConfigurator) transformationName(name string) string {
	return tc.transformationNamePrefix + name
}

func addParams(
	transformationConfig []string, params map[string]string, paramPrefix, replicationPrefix string) []string {
	if len(params) > 0 {
		var keys []string
		for key := range params {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := strings.ReplaceAll(params[key], "${replication_prefix}", replicationPrefix)
			transformationConfig = append(transformationConfig, fmt.Sprintf("%s.%s = %s", paramPrefix, key, value))
		}
	}
	return transformationConfig
}

func NewReplicationFlow(sourceClusterName, targetClusterName string) string {
	return fmt.Sprintf("%s->%s", sourceClusterName, targetClusterName)
}
