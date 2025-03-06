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

package util

import (
	"fmt"
	"os"
	"strings"
)

const (
	SwitchoverAnnotationKey = "switchoverRetry"
	RetryFailedComment      = "retry failed"
	kmmSecretPattern        = "%s-mirror-maker-secret"
	kmmConfigMapPattern     = "%s-mirror-maker-configuration"
	nameSeparator           = "-"
)

func DefaultIfEmpty(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func Contains(value string, list []string) bool {
	for _, element := range list {
		if value == element {
			return true
		}
	}
	return false
}

func EqualSlicesNil(actualSlice []string, expectedSlice []string) bool {
	if expectedSlice == nil && actualSlice == nil {
		return true
	} else {
		return EqualSlices(actualSlice, expectedSlice)
	}
}
func EqualSlices(actualSlice []string, expectedSlice []string) bool {
	if actualSlice == nil {
		return false
	}
	if len(actualSlice) != len(expectedSlice) {
		return false
	}
	for _, element := range actualSlice {
		if !Contains(element, expectedSlice) {
			return false
		}
	}
	return true
}

func EqualPlainMaps(actualMap map[string]string, expectedMap map[string]string) bool {
	if len(actualMap) != len(expectedMap) {
		return false
	}
	for key := range actualMap {
		if _, ok := expectedMap[key]; !ok {
			return false
		}
		if actualMap[key] != expectedMap[key] {
			return false
		}
	}
	return true
}

func EqualIntMaps(actualMap map[string]int64, expectedMap map[string]int64) bool {
	if len(actualMap) != len(expectedMap) {
		return false
	}
	for key := range actualMap {
		if _, ok := expectedMap[key]; !ok {
			return false
		} else {
			if actualMap[key] != expectedMap[key] {
				return false
			}
		}
	}
	return true
}

func EqualComplexMaps(actualMap map[string][]string, expectedMap map[string][]string) bool {
	if len(actualMap) != len(expectedMap) {
		return false
	}
	for key := range actualMap {
		if _, ok := expectedMap[key]; !ok {
			return false
		} else {
			if !EqualSlices(actualMap[key], expectedMap[key]) {
				return false
			}
		}
	}
	return true
}

func GetKmmSecretName() string {
	operatorName := os.Getenv("OPERATOR_NAME")
	return fmt.Sprintf(kmmSecretPattern, operatorName[:len(operatorName)-17])
}

func GetKmmConfigMapName() string {
	operatorName := os.Getenv("OPERATOR_NAME")
	return fmt.Sprintf(kmmConfigMapPattern, operatorName[:len(operatorName)-17])
}

func GetProperties(configMapContent string) map[string]interface{} {
	properties := make(map[string]interface{})
	rows := strings.Split(configMapContent, "\n")
	for _, row := range rows {
		if row != "" && !strings.HasPrefix(row, "#") {
			pair := strings.Split(row, "=")
			value := getTrimmedPropertyValue(pair[1])
			properties[strings.Trim(pair[0], " ")] = value
		}
	}
	return properties
}

func getTrimmedPropertyValue(property string) interface{} {
	values := strings.Split(property, ",")
	if len(values) == 1 {
		return strings.Trim(property, " ")
	}
	var trimmedValues []string
	for _, value := range values {
		trimmedValues = append(trimmedValues, strings.Trim(value, " "))
	}
	return trimmedValues
}

func JoinMaps(sideMap map[string]string, mainMap map[string]string) map[string]string {
	resMap := make(map[string]string)
	for key, value := range sideMap {
		resMap[key] = value
	}
	for key, value := range mainMap {
		resMap[key] = value
	}
	return resMap
}

func AreMapsEqual(first map[string]string, second map[string]string) bool {
	if len(first) != len(second) {
		return false
	}
	for key := range first {
		if _, ok := second[key]; !ok {
			return false
		}
		if first[key] != second[key] {
			return false
		}
	}
	return true
}

func JoinNames(names ...string) string {
	var elems []string
	for _, name := range names {
		if name != "" {
			elems = append(elems, name)
		}
	}
	return strings.Join(elems, nameSeparator)
}
