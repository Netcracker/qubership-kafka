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

package provider

import (
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

var noopLogger = logr.Discard()

func envVarMap(vars []corev1.EnvVar) map[string]string {
	m := make(map[string]string, len(vars))
	for _, v := range vars {
		m[v.Name] = v.Value
	}
	return m
}

// configMapToEnvVars tests

func TestConfigMapToEnvVars_NilMap(t *testing.T) {
	result := configMapToEnvVars(nil, "CONF_KAFKA_", nil, noopLogger)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil config, got %v", result)
	}
}

func TestConfigMapToEnvVars_EmptyMap(t *testing.T) {
	result := configMapToEnvVars(map[string]string{}, "CONF_KAFKA_", nil, noopLogger)
	if len(result) != 0 {
		t.Errorf("expected empty result for empty config, got %v", result)
	}
}

func TestConfigMapToEnvVars_EncodingRule(t *testing.T) {
	config := map[string]string{
		"num.io.threads": "16",
	}
	result := configMapToEnvVars(config, "CONF_KAFKA_", nil, noopLogger)
	m := envVarMap(result)
	if v, ok := m["CONF_KAFKA_NUM_IO_THREADS"]; !ok || v != "16" {
		t.Errorf("expected CONF_KAFKA_NUM_IO_THREADS=16, got %v", m)
	}
}

func TestConfigMapToEnvVars_MultipleKeys(t *testing.T) {
	config := map[string]string{
		"num.io.threads":                  "16",
		"group.initial.rebalance.delay.ms": "3000",
	}
	result := configMapToEnvVars(config, "CONF_KAFKA_", nil, noopLogger)
	m := envVarMap(result)
	if v, ok := m["CONF_KAFKA_NUM_IO_THREADS"]; !ok || v != "16" {
		t.Errorf("expected CONF_KAFKA_NUM_IO_THREADS=16, got %v", m)
	}
	if v, ok := m["CONF_KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS"]; !ok || v != "3000" {
		t.Errorf("expected CONF_KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=3000, got %v", m)
	}
}

func TestConfigMapToEnvVars_ConflictIsSkipped(t *testing.T) {
	config := map[string]string{
		"num.io.threads": "8",
	}
	existingEnvVars := []string{"CONF_KAFKA_NUM_IO_THREADS=32"}
	result := configMapToEnvVars(config, "CONF_KAFKA_", existingEnvVars, noopLogger)
	if len(result) != 0 {
		t.Errorf("expected conflicting config entry to be skipped, got %v", result)
	}
}

func TestConfigMapToEnvVars_NonConflictingBothIncluded(t *testing.T) {
	config := map[string]string{
		"group.initial.rebalance.delay.ms": "3000",
	}
	existingEnvVars := []string{"CONF_KAFKA_NUM_IO_THREADS=16"}
	result := configMapToEnvVars(config, "CONF_KAFKA_", existingEnvVars, noopLogger)
	m := envVarMap(result)
	if v, ok := m["CONF_KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS"]; !ok || v != "3000" {
		t.Errorf("expected CONF_KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=3000, got %v", m)
	}
}

func TestConfigMapToEnvVars_EmptyKeySkipped(t *testing.T) {
	config := map[string]string{
		"":               "value",
		"num.io.threads": "16",
	}
	result := configMapToEnvVars(config, "CONF_KAFKA_", nil, noopLogger)
	m := envVarMap(result)
	if _, ok := m[""]; ok {
		t.Error("empty key should not produce an env var")
	}
	if v, ok := m["CONF_KAFKA_NUM_IO_THREADS"]; !ok || v != "16" {
		t.Errorf("valid key should still be encoded, got %v", m)
	}
}

func TestConfigMapToEnvVars_MirrorMakerPrefix(t *testing.T) {
	config := map[string]string{
		"refresh.topics.interval.seconds": "30",
	}
	result := configMapToEnvVars(config, "CONF_", nil, noopLogger)
	m := envVarMap(result)
	if v, ok := m["CONF_REFRESH_TOPICS_INTERVAL_SECONDS"]; !ok || v != "30" {
		t.Errorf("expected CONF_REFRESH_TOPICS_INTERVAL_SECONDS=30, got %v", m)
	}
}

// Kafka provider integration tests: configMapToEnvVars + buildEnvs

func TestKafkaConfig_EnvVarsAppearedOnPod(t *testing.T) {
	config := map[string]string{
		"num.io.threads":                  "16",
		"group.initial.rebalance.delay.ms": "3000",
	}
	configEnvVars := configMapToEnvVars(config, "CONF_KAFKA_", nil, noopLogger)
	allEnvVars := buildEnvs(configEnvVars, nil, noopLogger)
	m := envVarMap(allEnvVars)
	if v, ok := m["CONF_KAFKA_NUM_IO_THREADS"]; !ok || v != "16" {
		t.Errorf("expected CONF_KAFKA_NUM_IO_THREADS=16, got %v", m)
	}
	if v, ok := m["CONF_KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS"]; !ok || v != "3000" {
		t.Errorf("expected CONF_KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=3000, got %v", m)
	}
}

func TestKafkaConfig_EnvironmentVariablesWins(t *testing.T) {
	config := map[string]string{
		"num.io.threads": "8",
	}
	environmentVariables := []string{"CONF_KAFKA_NUM_IO_THREADS=32"}
	configEnvVars := configMapToEnvVars(config, "CONF_KAFKA_", environmentVariables, noopLogger)
	allEnvVars := buildEnvs(configEnvVars, environmentVariables, noopLogger)
	m := envVarMap(allEnvVars)
	if v, ok := m["CONF_KAFKA_NUM_IO_THREADS"]; !ok || v != "32" {
		t.Errorf("environmentVariables should win; expected CONF_KAFKA_NUM_IO_THREADS=32, got %v", m)
	}
}

// MirrorMaker provider integration tests

func TestMirrorMakerConfig_EnvVarsAppearedOnPod(t *testing.T) {
	config := map[string]string{
		"refresh.topics.interval.seconds": "30",
		"sync.group.offsets.enabled":      "true",
	}
	configEnvVars := configMapToEnvVars(config, "CONF_", nil, noopLogger)
	allEnvVars := buildEnvs(configEnvVars, nil, noopLogger)
	m := envVarMap(allEnvVars)
	if v, ok := m["CONF_REFRESH_TOPICS_INTERVAL_SECONDS"]; !ok || v != "30" {
		t.Errorf("expected CONF_REFRESH_TOPICS_INTERVAL_SECONDS=30, got %v", m)
	}
	if v, ok := m["CONF_SYNC_GROUP_OFFSETS_ENABLED"]; !ok || v != "true" {
		t.Errorf("expected CONF_SYNC_GROUP_OFFSETS_ENABLED=true, got %v", m)
	}
}

func TestMirrorMakerConfig_EnvironmentVariablesWins(t *testing.T) {
	config := map[string]string{
		"refresh.topics.interval.seconds": "30",
	}
	environmentVariables := []string{"CONF_REFRESH_TOPICS_INTERVAL_SECONDS=60"}
	configEnvVars := configMapToEnvVars(config, "CONF_", environmentVariables, noopLogger)
	allEnvVars := buildEnvs(configEnvVars, environmentVariables, noopLogger)
	m := envVarMap(allEnvVars)
	if v, ok := m["CONF_REFRESH_TOPICS_INTERVAL_SECONDS"]; !ok || v != "60" {
		t.Errorf("environmentVariables should win; expected CONF_REFRESH_TOPICS_INTERVAL_SECONDS=60, got %v", m)
	}
}
