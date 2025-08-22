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
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	akhqconfigv1 "github.com/Netcracker/qubership-kafka/operator/api/v1"
	"github.com/Netcracker/qubership-kafka/operator/util"
	"github.com/go-logr/logr"
	"github.com/go-yaml/yaml"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	tmpFileName                 = "/tmp/protobuf"
	tmpCompressedFileName       = "/tmp/compressed-proto.gz"
	akhqDeserializationCMPrefix = "akhqdcm-"
)

func (r *AkhqConfigReconciler) getConfigMap(configMapName, namespace string) (*v1.ConfigMap, error) {
	foundCM := &v1.ConfigMap{}
	if err := r.Client.Get(context.TODO(),
		types.NamespacedName{Name: configMapName, Namespace: namespace}, foundCM); err != nil {
		return nil, err
	}
	return foundCM, nil
}

func (r *AkhqConfigReconciler) getDeserializationConfig(deserializationCM *v1.ConfigMap) (*DeserializationConfig, error) {
	data := deserializationCM.Data["config"]
	dc := &DeserializationConfig{}
	if err := yaml.Unmarshal([]byte(data), dc); err != nil {
		return nil, err
	}
	return dc, nil
}

func (r *AkhqConfigReconciler) updateDeserializationConfigMap(deserializationCM *v1.ConfigMap, dc DeserializationConfig) error {
	binDc, err := yaml.Marshal(dc)
	if err != nil {
		return err
	}
	deserializationCM.Data["config"] = string(binDc)
	if err = r.Client.Update(context.TODO(), deserializationCM); err != nil {
		return err
	}
	return nil
}

func (r *AkhqConfigReconciler) applyBinaryConfigMap(namespace string, mapToApply map[string]string, labels map[string]string, logger logr.Logger) error {
	logger.Info(fmt.Sprintf("Applying %d AKHQ deserialization key binary config maps...", len(mapToApply)))

	for configName, rawConfig := range mapToApply {
		namespacedName := types.NamespacedName{Name: akhqDeserializationCMPrefix + configName, Namespace: namespace}
		configAlreadyPresented := true
		descKeysCM := &v1.ConfigMap{}

		// Try to load config map from the cluster
		if err := r.Client.Get(context.TODO(), namespacedName, descKeysCM); err != nil {
			if !k8sErrors.IsNotFound(err) {
				return err
			}
			logger.Info(fmt.Sprintf("Config map %s was not found in cluster", namespacedName.Name))
			configAlreadyPresented = false
		}

		// Compress raw config
		content, err := base64.StdEncoding.DecodeString(rawConfig)
		if err != nil {
			continue
		}
		compressedBytes, err := compressBytes(&content)
		if err != nil {
			return err
		}

		// Store with ".gz" suffix because the AKHQ decomposes it using `gzip` tool, it requires to have `.gz` suffix.
		logger.Info(fmt.Sprintf("Storing %s file (compressed)", configName))
		if descKeysCM.BinaryData == nil {
			descKeysCM.BinaryData = make(map[string][]byte, 1)
		}
		descKeysCM.BinaryData[configName+".gz"] = *compressedBytes
		descKeysCM.Labels = labels

		// Save the result
		if !configAlreadyPresented {
			logger.Info(fmt.Sprintf("Creating %s config map in %s namespace", namespacedName.Name, namespacedName.Namespace))
			descKeysCM.Name = namespacedName.Name
			descKeysCM.Namespace = namespace

			if err := r.Client.Create(context.TODO(), descKeysCM); err != nil {
				logger.Error(err, fmt.Sprintf("Error while %s config map creation", namespacedName.Name))
				return err
			}

			continue
		}

		logger.Info(fmt.Sprintf("Updating %s config map in %s namespace.", namespacedName.Name, namespacedName.Namespace))
		if err := r.Client.Update(context.TODO(), descKeysCM); err != nil {
			return err
		}

	}

	return nil
}

func (r *AkhqConfigReconciler) deleteBinaryConfigMap(namespace string, mapToDelete map[string]string, logger logr.Logger) error {
	logger.Info(fmt.Sprintf("Deleting %d AKHQ deserialization key binary config maps...", len(mapToDelete)))

	for configName := range mapToDelete {
		logger.Info(fmt.Sprintf("Checking Existence of [%s] config map", configName))

		foundConfigMap := &corev1.ConfigMap{}
		configMapName := akhqDeserializationCMPrefix + configName
		namespacedName := types.NamespacedName{Name: configMapName, Namespace: namespace}

		err := r.Client.Get(context.TODO(), namespacedName, foundConfigMap)
		if err != nil {
			if errors.IsNotFound(err) {
				logger.Info(fmt.Sprintf("Deserialization config map %s not found, so nothing to delete", namespacedName.Name))
				continue
			}
			return err
		}

		if err := r.Client.Delete(context.TODO(), foundConfigMap); err != nil {
			logger.Error(err, fmt.Sprintf("Error while %s config map deletion", namespacedName.Name))
			return err
		}
	}

	return nil
}

func compressBytes(bytes *[]byte) (*[]byte, error) {

	if err := storeBytesToTmpFile(bytes); err != nil {
		return nil, err
	}

	if err := compressBytesFromTmpFile(); err != nil {
		return nil, err
	}
	defer os.Remove(tmpFileName)
	defer os.Remove(tmpCompressedFileName)

	fileBytes, err := os.ReadFile(tmpCompressedFileName)
	if err != nil {
		return nil, err
	}

	return &fileBytes, nil
}

func compressBytesFromTmpFile() error {
	tmpFile, err := os.Open(tmpFileName)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	compressedTmp, err := os.Create(tmpCompressedFileName)
	if err != nil {
		return err
	}
	defer compressedTmp.Close()

	writer := gzip.NewWriter(compressedTmp)
	defer writer.Close()

	if _, err = io.Copy(writer, tmpFile); err != nil {
		return err
	}

	return nil
}

func storeBytesToTmpFile(bytes *[]byte) error {
	tmpFile, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	if _, err = tmpFile.Write(*bytes); err != nil {
		return err
	}

	return nil
}

func (r *AkhqConfigReconciler) fullUpdateDeserializationConfig(existingConfigs []TopicMapping,
	crConfigsMap map[string]akhqconfigv1.Config, crFullName string) ([]TopicMapping, map[string]string) {
	mapToUpdate := make(map[string]string)
	insertPosition := len(existingConfigs)
	for i := 0; i < len(existingConfigs); i++ {
		existedConfig := existingConfigs[i]
		var shortFileName string
		splitNames := strings.Split(existedConfig.DescriptorFile, "/")
		if len(splitNames) == 2 {
			shortFileName = splitNames[1]
		} else {
			shortFileName = splitNames[0]
		}
		if existedConfig.Name == crFullName {
			if crConfig, ok := crConfigsMap[existedConfig.TopicRegex]; ok {
				existedConfig.KeyMessageType = crConfig.KeyType
				existedConfig.ValueMessageType = crConfig.MessageType
				mapToUpdate[shortFileName] = crConfig.DescriptorFileBase64
				existingConfigs[i] = existedConfig
				delete(crConfigsMap, crConfig.TopicRegex)
			} else {
				copy(existingConfigs[i:], existingConfigs[i+1:])
				existingConfigs[len(existingConfigs)-1] = TopicMapping{}
				existingConfigs = existingConfigs[:len(existingConfigs)-1]
				i--
				mapToUpdate[shortFileName] = ""
			}
			insertPosition = i + 1
		}
	}
	toInsert := make([]TopicMapping, 0, len(crConfigsMap))
	for _, value := range crConfigsMap {
		fileName := fmt.Sprintf("%s-%s", crFullName, util.StringHash(value.TopicRegex))
		topicMapping := TopicMapping{
			TopicRegex:       value.TopicRegex,
			Name:             crFullName,
			KeyMessageType:   value.KeyType,
			ValueMessageType: value.MessageType,
			DescriptorFile:   fmt.Sprintf("descs/%s", fileName),
		}
		toInsert = append(toInsert, topicMapping)
		mapToUpdate[fileName] = value.DescriptorFileBase64
	}
	existingConfigs = append(existingConfigs[:insertPosition], append(toInsert, existingConfigs[insertPosition:]...)...)
	return existingConfigs, mapToUpdate
}

func (r *AkhqConfigReconciler) deleteDeserializationConfig(existingConfigs []TopicMapping,
	crFullName string) ([]TopicMapping, map[string]string) {
	mapToDelete := make(map[string]string)
	for i := 0; i < len(existingConfigs); i++ {
		existedConfig := existingConfigs[i]
		if existedConfig.Name == crFullName {
			if existedConfig.DescriptorFile != "" {
				shortFileName := strings.Split(existedConfig.DescriptorFile, "/")[1]
				mapToDelete[shortFileName] = ""
			}
			copy(existingConfigs[i:len(existingConfigs)-1], existingConfigs[i+1:])
			existingConfigs[len(existingConfigs)-1] = TopicMapping{}
			existingConfigs = existingConfigs[:len(existingConfigs)-1]
			i--
		}
	}
	return existingConfigs, mapToDelete
}
