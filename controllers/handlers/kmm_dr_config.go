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
	"regexp"
	"strings"
)

const (
	arrowTopicsPattern                    = "^[\\w]*->[\\w]*\\.topics$"
	arrowBlackListPattern                 = "^[\\w]*->[\\w]*\\.topics\\.blacklist$"
	arrowEnabledPattern                   = "^[\\w]*->[\\w]*\\.enabled$"
	brokersFormattedString                = "%s.bootstrap.servers"
	clustersPropertyNotFoundError         = "can not find KMM property 'clusters'"
	incorrectClustersCountError           = "in disaster recovery mode exactly two clusters must be declared in 'clusters' property"
	sameClusterNameError                  = "declared KMM clusters can not be repeated"
	targetDcPropertyInvalidError          = "can not find KMM property 'target.dc' or property has incorrect value"
	bootstrapServersPropertyNotFoundError = "%s Kafka bootstrap servers are not set"
)

var stringProperties = []string{"target.dc", "enabled"}
var sliceProperties = []string{"clusters", "topics", "topics.blacklist"}
var arrowSlicePatterns = []string{arrowTopicsPattern, arrowBlackListPattern}
var arrowStringPatterns = []string{arrowEnabledPattern}

func getStringProperties(config string) map[string]string {
	properties := make(map[string]string)
	rows := strings.Split(config, "\n")
	for _, row := range rows {
		if row != "" && !strings.HasPrefix(row, "#") {
			if strings.Contains(row, "=") {
				pair := strings.SplitN(row, "=", 2)
				key := strings.Trim(pair[0], " ")
				value := strings.Trim(pair[1], " ")
				properties[key] = value
			}
		}
	}
	return properties
}

func NewKmmDrConfig(properties map[string]string) (*KmmDrConfig, error) {
	config := &KmmDrConfig{
		Properties:       properties,
		StringProperties: make(map[string]string),
		SliceProperties:  make(map[string][]string),
	}
	err := config.enrichProperties()
	if err != nil {
		return nil, err
	}
	return config, nil
}

type KmmDrConfig struct {
	Properties         map[string]string
	StringProperties   map[string]string
	SliceProperties    map[string][]string
	ArrowPattern       string
	TargetCluster      string
	SourceCluster      string
	TargetBrokers      []string
	SourceBrokers      []string
	ReplicationEnabled bool
}

func (kdc *KmmDrConfig) enrichProperties() error {
	kdc.enrichInitial()
	err := kdc.validateClusters()
	if err != nil {
		return err
	}
	if err = kdc.resolveBootstrapServers(); err != nil {
		return err
	}
	kdc.resolveReplicationEnable()
	return nil
}

func (kdc *KmmDrConfig) enrichInitial() {
	kdc.enrichStringProperties(stringProperties)
	kdc.enrichSliceProperties(sliceProperties)
	kdc.enrichArrowStringProperties(arrowStringPatterns)
	kdc.enrichArrowSliceProperties(arrowSlicePatterns)
}

func (kdc *KmmDrConfig) validateClusters() error {
	clusters, ok := kdc.SliceProperties["clusters"]
	if !ok {
		return errors.New(clustersPropertyNotFoundError)
	}
	if len(clusters) != 2 {
		return errors.New(incorrectClustersCountError)
	}
	if clusters[0] == clusters[1] {
		return errors.New(sameClusterNameError)
	}
	target, ok := kdc.StringProperties["target.dc"]
	if !ok {
		return errors.New(targetDcPropertyInvalidError)
	}
	kdc.TargetCluster = target
	for _, cluster := range clusters {
		if cluster != target {
			kdc.SourceCluster = cluster
		}
	}
	kdc.ArrowPattern = fmt.Sprintf("%s->%s", kdc.SourceCluster, kdc.TargetCluster)
	return nil
}

func (kdc *KmmDrConfig) enrichStringProperties(stringProperties []string) {
	for _, property := range stringProperties {
		if value, ok := kdc.Properties[property]; ok {
			if !strings.Contains(value, ",") {
				kdc.StringProperties[property] = value
			}
		}
	}
}

func (kdc *KmmDrConfig) enrichSliceProperties(sliceProperties []string) {
	for _, property := range sliceProperties {
		if value, ok := kdc.Properties[property]; ok {
			if value == "" {
				kdc.SliceProperties[property] = []string{}
			} else {
				list := strings.Split(value, ",")
				for i, member := range list {
					list[i] = strings.Trim(member, " ")
				}
				kdc.SliceProperties[property] = list
			}
		}
	}
}

func (kdc *KmmDrConfig) enrichArrowStringProperties(stringPatterns []string) {
	for key, property := range kdc.Properties {
		for _, pattern := range stringPatterns {
			if matched, _ := regexp.MatchString(pattern, key); matched {
				if !strings.Contains(property, ",") {
					kdc.StringProperties[key] = property
				}
			}
		}
	}
}

func (kdc *KmmDrConfig) enrichArrowSliceProperties(slicePatterns []string) {
	for key, property := range kdc.Properties {
		for _, pattern := range slicePatterns {
			if matched, _ := regexp.MatchString(pattern, key); matched {
				if property == "" {
					kdc.SliceProperties[key] = []string{}
				} else {
					list := strings.Split(property, ",")
					for i, member := range list {
						list[i] = strings.Trim(member, " ")
					}
					kdc.SliceProperties[key] = list
				}
			}
		}
	}
}

func (kdc *KmmDrConfig) resolveBootstrapServers() error {
	target := fmt.Sprintf(brokersFormattedString, kdc.TargetCluster)
	source := fmt.Sprintf(brokersFormattedString, kdc.SourceCluster)
	brokerKeys := []string{target, source}
	kdc.enrichSliceProperties(brokerKeys)
	targetBrokers, ok := kdc.SliceProperties[target]
	if !ok || len(targetBrokers) == 0 {
		return fmt.Errorf(bootstrapServersPropertyNotFoundError, "target")
	}
	sourceBrokers, ok := kdc.SliceProperties[source]
	if !ok || len(sourceBrokers) == 0 {
		return fmt.Errorf(bootstrapServersPropertyNotFoundError, "source")
	}
	kdc.TargetBrokers = targetBrokers
	kdc.SourceBrokers = sourceBrokers
	return nil
}

func (kdc *KmmDrConfig) resolveReplicationEnable() {
	value := kdc.resolveGlobalLocalStringValue("enabled")
	if value == nil {
		kdc.ReplicationEnabled = true
	} else {
		kdc.ReplicationEnabled = strings.ToLower(value.(string)) == "true"
	}
}

func (kdc *KmmDrConfig) resolveGlobalLocalStringValue(key string) interface{} {
	globalValue, globalOk := kdc.StringProperties[key]
	localValue, localOk := kdc.StringProperties[fmt.Sprintf("%s.%s", kdc.ArrowPattern, key)]
	if !globalOk && !localOk {
		return nil
	} else {
		if localOk {
			return localValue
		} else {
			return globalValue
		}
	}
}

func (kdc *KmmDrConfig) resolveGlobalLocalSliceValue(key string) interface{} {
	globalValue, globalOk := kdc.SliceProperties[key]
	localValue, localOk := kdc.SliceProperties[fmt.Sprintf("%s.%s", kdc.ArrowPattern, key)]
	if !globalOk && !localOk {
		return nil
	} else {
		if localOk {
			return localValue
		} else {
			return globalValue
		}
	}
}

func (kdc *KmmDrConfig) getAllowRegExps() []string {
	value := kdc.resolveGlobalLocalSliceValue("topics")
	if value == nil {
		return nil
	} else {
		//TODO: what about empty slice
		return value.([]string)
	}
}

func (kdc *KmmDrConfig) getBlockRegExps() []string {
	value := kdc.resolveGlobalLocalSliceValue("topics.blacklist")
	if value == nil {
		return nil
	} else {
		//TODO: what about empty slice
		return value.([]string)
	}
}

func (kdc *KmmDrConfig) GetRegularExpressions() ([]string, []string) {
	return kdc.getAllowRegExps(), kdc.getBlockRegExps()
}
