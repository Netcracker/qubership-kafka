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

package kafkauser

import (
	"fmt"
	"github.com/IBM/sarama"
	"github.com/go-logr/logr"
	"github.com/sethvargo/go-password/password"
	corev1 "k8s.io/api/core/v1"
	"net/url"
	"strings"
)

const (
	connectionPropertiesKey = "connection-properties"
	connectionUriKey        = "connection-uri"
	kafkaClusterKey         = "kafka-cluster"
	usernameKey             = "username"
	passwordKey             = "password"
	adminRole               = "admin"
	namespaceAdminRole      = "namespace-admin"
	scramSha512             = "scram-sha-512"
)

type UserProvider struct {
	logger      logr.Logger
	kafkaClient sarama.ClusterAdmin
}

func NewUserProvider(kafkaClient sarama.ClusterAdmin, logger logr.Logger) *UserProvider {
	return &UserProvider{
		kafkaClient: kafkaClient,
		logger:      logger,
	}
}

func (up *UserProvider) extractConnectionProperties(secret *corev1.Secret, format string) (string, string, error) {
	if connectionPropertiesKey == format {
		return string(secret.Data[usernameKey]), string(secret.Data[passwordKey]), nil
	} else {
		connectionUri := string(secret.Data[connectionUriKey])
		uri, err := url.Parse(connectionUri)
		if err != nil {
			return "", "", err
		}
		username := uri.User.Username()
		password, _ := uri.User.Password()
		return username, password, nil
	}
}

func (up *UserProvider) upsertKafkaUser(username string, password string, authenticationType string) error {
	var mechanism sarama.ScramMechanismType
	switch authenticationType {
	case scramSha512:
		mechanism = sarama.SCRAM_MECHANISM_SHA_512
	default:
		return fmt.Errorf("unsupported SCRAM authntication type: %s", authenticationType)
	}
	_, err := up.kafkaClient.UpsertUserScramCredentials([]sarama.AlterUserScramCredentialsUpsert{
		{
			Name:       username,
			Mechanism:  mechanism,
			Iterations: 8192,
			Password:   []byte(password),
		},
	})
	return err
}

func (up *UserProvider) deleteKafkaUser(username string, authenticationType string) error {
	var mechanism sarama.ScramMechanismType
	switch strings.ToLower(authenticationType) {
	case scramSha512:
		mechanism = sarama.SCRAM_MECHANISM_SHA_512
	default:
		return fmt.Errorf("unsupported SCRAM authntication type: %s", authenticationType)
	}
	_, err := up.kafkaClient.DeleteUserScramCredentials([]sarama.AlterUserScramCredentialsDelete{
		{
			Name:      username,
			Mechanism: mechanism,
		},
	})
	return err
}

func (up *UserProvider) createACLs(namespace string, role string, username string) ([]string, error) {
	principal := fmt.Sprintf("User:%s", username)
	var resourceNamePattern string
	var resourcePatternType sarama.AclResourcePatternType
	switch role {
	case adminRole:
		resourcePatternType = sarama.AclPatternLiteral
		resourceNamePattern = "*"
	case namespaceAdminRole:
		resourcePatternType = sarama.AclPatternPrefixed
		resourceNamePattern = namespace
	default:
		return nil, fmt.Errorf("unsupported KafkaUser role: %s", role)
	}

	rACLs := []*sarama.ResourceAcls{
		{
			Resource: sarama.Resource{ResourceType: sarama.AclResourceCluster, ResourceName: kafkaClusterKey, ResourcePatternType: sarama.AclPatternLiteral},
			Acls: []*sarama.Acl{
				{Host: "*", Operation: sarama.AclOperationCreate, PermissionType: sarama.AclPermissionAllow, Principal: principal},
				{Host: "*", Operation: sarama.AclOperationDescribeConfigs, PermissionType: sarama.AclPermissionAllow, Principal: principal},
				{Host: "*", Operation: sarama.AclOperationAlterConfigs, PermissionType: sarama.AclPermissionAllow, Principal: principal},
			},
		},
		{
			Resource: sarama.Resource{ResourceType: sarama.AclResourceTopic, ResourceName: resourceNamePattern, ResourcePatternType: resourcePatternType},
			Acls: []*sarama.Acl{
				{Host: "*", Operation: sarama.AclOperationCreate, PermissionType: sarama.AclPermissionAllow, Principal: principal},
				{Host: "*", Operation: sarama.AclOperationAlter, PermissionType: sarama.AclPermissionAllow, Principal: principal},
				{Host: "*", Operation: sarama.AclOperationDelete, PermissionType: sarama.AclPermissionAllow, Principal: principal},
				{Host: "*", Operation: sarama.AclOperationDescribe, PermissionType: sarama.AclPermissionAllow, Principal: principal},
				{Host: "*", Operation: sarama.AclOperationWrite, PermissionType: sarama.AclPermissionAllow, Principal: principal},
				{Host: "*", Operation: sarama.AclOperationRead, PermissionType: sarama.AclPermissionAllow, Principal: principal},
				{Host: "*", Operation: sarama.AclOperationDescribeConfigs, PermissionType: sarama.AclPermissionAllow, Principal: principal},
			},
		},
		{
			Resource: sarama.Resource{ResourceType: sarama.AclResourceGroup, ResourceName: resourceNamePattern, ResourcePatternType: resourcePatternType},
			Acls: []*sarama.Acl{
				{Host: "*", Operation: sarama.AclOperationRead, PermissionType: sarama.AclPermissionAllow, Principal: principal},
			},
		},
		{
			Resource: sarama.Resource{ResourceType: sarama.AclResourceTransactionalID, ResourceName: resourceNamePattern, ResourcePatternType: resourcePatternType},
			Acls: []*sarama.Acl{
				{Host: "*", Operation: sarama.AclOperationWrite, PermissionType: sarama.AclPermissionAllow, Principal: principal},
			},
		},
	}
	err := up.kafkaClient.CreateACLs(rACLs)
	if err != nil {
		return nil, err
	}

	aclFilter := sarama.AclFilter{
		Principal:                 &principal,
		ResourceType:              sarama.AclResourceAny,
		ResourcePatternTypeFilter: sarama.AclPatternAny,
		PermissionType:            sarama.AclPermissionAllow,
		Operation:                 sarama.AclOperationAny,
	}
	aclResources, err := up.kafkaClient.ListAcls(aclFilter)
	if err != nil {
		return nil, err
	}

	for _, aclResource := range aclResources {
		for _, rACL := range rACLs {
			if aclResource.ResourceType == rACL.ResourceType {
				for _, createdACL := range aclResource.Acls {
					aclFound := false
					for _, expectedACL := range rACL.Acls {
						if *createdACL == *expectedACL {
							aclFound = true
							break
						}
					}
					if !aclFound {
						aclFilter := sarama.AclFilter{
							Principal:                 &principal,
							ResourceType:              aclResource.ResourceType,
							ResourcePatternTypeFilter: aclResource.ResourcePatternType,
							PermissionType:            createdACL.PermissionType,
							Operation:                 createdACL.Operation,
						}
						_, err = up.kafkaClient.DeleteACL(aclFilter, false)
						if err != nil {
							return nil, err
						}
					}
				}
				break
			}
		}
	}

	aclResources, err = up.kafkaClient.ListAcls(aclFilter)
	if err != nil {
		return nil, err
	}
	var createdAcls []string
	for _, aclResource := range aclResources {
		for _, acl := range aclResource.Acls {
			var namespacePattern string
			if aclResource.ResourceType == sarama.AclResourceCluster {
				namespacePattern = ""
			} else {
				if role == adminRole {
					namespacePattern = " *"
				} else {
					namespacePattern = fmt.Sprintf(" %s*", namespace)
				}

			}
			createdAcls = append(createdAcls, fmt.Sprintf("--operation %s --%s%s", acl.Operation.String(), aclResource.ResourceType.String(), namespacePattern))
		}
	}
	return createdAcls, nil
}

func (up *UserProvider) deleteACLs(username string) error {
	principal := fmt.Sprintf("User:%s", username)
	aclFilter := sarama.AclFilter{
		Principal:                 &principal,
		ResourceType:              sarama.AclResourceAny,
		ResourcePatternTypeFilter: sarama.AclPatternAny,
		PermissionType:            sarama.AclPermissionAllow,
		Operation:                 sarama.AclOperationAny,
	}
	_, err := up.kafkaClient.DeleteACL(aclFilter, false)
	return err
}

func (up *UserProvider) generatePassword() (string, error) {
	generator, err := password.NewGenerator(&password.GeneratorInput{Symbols: "_&!@#"})
	if err != nil {
		return "", err
	}
	return generator.Generate(12, 1, 1, false, false)
}
