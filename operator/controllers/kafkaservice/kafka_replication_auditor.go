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
	"fmt"
	"github.com/IBM/sarama"
	kafkaservice "github.com/Netcracker/qubership-kafka/operator/api/v7"
	"github.com/Netcracker/qubership-kafka/operator/controllers"
	"github.com/Netcracker/qubership-kafka/operator/controllers/handlers"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	delay = time.Second * 5
)

var repLogger = log.WithName("Replication auditor with timeout")

type KafkaReplicationAuditor struct {
	cr         *kafkaservice.KafkaService
	reconciler *KafkaServiceReconciler
}

func NewKafkaReplicationAuditor(cr *kafkaservice.KafkaService, r *KafkaServiceReconciler) *KafkaReplicationAuditor {
	return &KafkaReplicationAuditor{cr: cr, reconciler: r}
}

func (a *KafkaReplicationAuditor) CheckFullReplication(timeout time.Duration) (bool, error) {
	repLogger.Info("Kafka replication auditor started")
	ticker := time.NewTicker(timeout)
	replicationDone := make(chan error)
	stopAudit := make(chan error)
	defer func() {
		ticker.Stop()
		close(stopAudit)
	}()
	go func() {
		replicationDone <- a.checkFullReplication(stopAudit)
	}()
	select {
	case err := <-replicationDone:
		repLogger.Info(fmt.Sprintf("Replication audit has finished with error - [%v]", err))
		return true, err
	case <-ticker.C:
		repLogger.Info("Audit replication is not finished. Timeout is occurred.")
		return false, nil
	}
}

func (a *KafkaReplicationAuditor) checkFullReplication(stopAudit chan error) error {
	kmmConfigHandler, err := handlers.NewKmmConfigHandler(a.reconciler.Client)
	if err != nil {
		return err
	}
	if !kmmConfigHandler.IsReplicationEnabled() {
		repLogger.Info("WARNING! Replication between Kafka clusters is disabled. " +
			"Switch over replication check will be skipped")
		return nil
	}
	clientStandby, clientActive, err := a.getKafkaClients(kmmConfigHandler)
	if err != nil {
		return err
	}
	allowRegExps, blockRegExps := kmmConfigHandler.GetRegularExpressions()
	topicsActive, err := clientActive.Topics()
	if err != nil {
		repLogger.Error(err, "can not list Kafka topics from active side")
		return err
	}
	var replicatedTopics []string
	for _, topic := range topicsActive {
		if mustBeTopicReplicated(topic, allowRegExps, blockRegExps) {
			replicatedTopics = append(replicatedTopics, topic)
		}
	}
	repLogger.Info(fmt.Sprintf("check full replication for topics: %s", replicatedTopics))

	var wg sync.WaitGroup
	activeChan := make(chan map[string]int64)
	standbyChan := make(chan map[string]int64)
	go getMessagesCountWrapped(&wg, activeChan, clientActive, replicatedTopics)
	go getMessagesCountWrapped(&wg, standbyChan, clientStandby, replicatedTopics)
	wg.Wait()
	messagesActive := <-activeChan
	messagesStandby := <-standbyChan
	close(activeChan)
	close(standbyChan)
	notReplicatedPartitions := flushOut(messagesActive, messagesStandby)
	if len(notReplicatedPartitions) == 0 {
		return nil
	}
	repLogger.Info(fmt.Sprintf("[%+v] partitions have not been replicated yet", notReplicatedPartitions))
	return waitUntilActiveMessagesFlushOut(notReplicatedPartitions, clientStandby, stopAudit)
}

func isChannelOpen(channel chan error) bool {
	ok := true
	select {
	case _, ok = <-channel:
	default:
	}
	return ok
}

func (a *KafkaReplicationAuditor) getKafkaClients(handler *handlers.KmmConfigHandler) (sarama.Client, sarama.Client, error) {
	targetBrokers, sourceBrokers := handler.GetBrokers()
	repLogger.Info(fmt.Sprintf("%s-%s", "source brokers", sourceBrokers))
	repLogger.Info(fmt.Sprintf("%s-%s", "target brokers", targetBrokers))
	targetCluster, err := findCluster(handler.GetTargetCluster(), a.cr)
	if err != nil {
		return nil, nil, err
	}
	sourceCluster, err := findCluster(handler.GetSourceCluster(), a.cr)
	if err != nil {
		return nil, nil, err
	}
	targetCreds, sourceCreds := handler.GetKafkaUsernamePasswords()
	targetSaslSettings := &controllers.SaslSettings{
		Mechanism: targetCluster.SaslMechanism,
		Username:  targetCreds[0],
		Password:  targetCreds[1],
	}
	sourceSaslSettings := &controllers.SaslSettings{
		Mechanism: sourceCluster.SaslMechanism,
		Username:  sourceCreds[0],
		Password:  sourceCreds[1],
	}

	namespace := os.Getenv("OPERATOR_NAMESPACE")
	targetSslCertificates, err := a.getSslCertificates(targetCluster, namespace)
	if err != nil {
		return nil, nil, err
	}
	sourceSslCertificates, err := a.getSslCertificates(sourceCluster, namespace)
	if err != nil {
		return nil, nil, err
	}
	repLogger.Info(fmt.Sprintf("Define Kafka client config for target cluster with name: %s", targetCluster.Name))
	targetKafkaConfig, err := controllers.NewKafkaClientConfig(targetSaslSettings, targetCluster.EnableSsl, targetSslCertificates)
	if err != nil {
		return nil, nil, err
	}
	repLogger.Info(fmt.Sprintf("Define Kafka client config for source cluster with name: %s", sourceCluster.Name))
	sourceKafkaConfig, err := controllers.NewKafkaClientConfig(sourceSaslSettings, sourceCluster.EnableSsl, sourceSslCertificates)
	if err != nil {
		return nil, nil, err
	}
	targetClient, err := sarama.NewClient(targetBrokers, targetKafkaConfig)
	if err != nil {
		return nil, nil, err
	}
	sourceClient, err := sarama.NewClient(sourceBrokers, sourceKafkaConfig)
	if err != nil {
		return nil, nil, err
	}
	return targetClient, sourceClient, nil
}

func (a *KafkaReplicationAuditor) getSslCertificates(cluster *kafkaservice.Cluster, namespace string) (*controllers.SslCertificates, error) {
	if cluster.EnableSsl && cluster.SslSecretName != "" {
		sslCertificates, err := a.reconciler.GetSslCertificates(cluster.SslSecretName, namespace, repLogger)
		if err != nil {
			return nil, err
		}
		return sslCertificates, nil
	}
	return &controllers.SslCertificates{}, nil
}

func findCluster(clusterName string, cr *kafkaservice.KafkaService) (*kafkaservice.Cluster, error) {
	for _, cluster := range cr.Spec.MirrorMaker.Clusters {
		if strings.EqualFold(clusterName, cluster.Name) {
			return &cluster, nil
		}
	}
	return nil, fmt.Errorf("cannot find cluster with name: %s", clusterName)
}

func mustBeTopicReplicated(topic string, allowTopicRegExps []string, blockTopicRegExps []string) bool {
	// service topics must not be replicated
	if strings.HasPrefix(topic, "__") || topic == "heartbeats" {
		return false
	}
	for _, blockPattern := range blockTopicRegExps {
		blockRes, err := matchTopic(blockPattern, topic)
		if err != nil {
			repLogger.Error(err, "Regular expression from block list is invalid")
		}
		if blockRes {
			return false
		}
	}
	if allowTopicRegExps == nil {
		return true
	}
	for _, allowPattern := range allowTopicRegExps {
		allowRes, err := matchTopic(allowPattern, topic)
		if err != nil {
			repLogger.Error(err, "Regular expression from allow list is invalid")
		}
		if allowRes {
			return true
		}
	}
	return false
}

func matchTopic(pattern string, topic string) (bool, error) {
	return regexp.MatchString("^"+pattern+"$", topic)
}

func waitUntilActiveMessagesFlushOut(notReplicatedActivePartitions map[string]int64, clientStandby sarama.Client,
	stopAudit chan error) error {
	for isChannelOpen(stopAudit) {
		topics := getTopicList(notReplicatedActivePartitions)
		repLogger.Info(fmt.Sprintf("not replicated topics are %s", topics))
		notReplicatedStandByPartitions, err := getMessagesCount(clientStandby, topics)
		if err != nil {
			return err
		}
		notReplicatedActivePartitions = flushOut(notReplicatedActivePartitions, notReplicatedStandByPartitions)
		if len(notReplicatedActivePartitions) == 0 {
			return nil
		}
		repLogger.Info(fmt.Sprintf("have not been replicated partitions - %+v", notReplicatedActivePartitions))
		time.Sleep(delay)
	}
	return nil
}

func getMessagesCount(client sarama.Client, topics []string) (map[string]int64, error) {
	messagesCount := make(map[string]int64)
	for _, topic := range topics {
		partitions, err := client.Partitions(topic)
		if err != nil {
			repLogger.Error(err, fmt.Sprintf("can not list partitions for the particular topic - %s", topic))
			return messagesCount, err
		}
		for _, partition := range partitions {
			latestOffset, err := client.GetOffset(topic, partition, sarama.OffsetNewest)
			if err != nil {
				repLogger.Error(err, fmt.Sprintf("can not get the particular latest offset for topic partition - %s, %d", topic, partition))
				return messagesCount, err
			}
			earliestOffset, err := client.GetOffset(topic, partition, sarama.OffsetOldest)
			if err != nil {
				repLogger.Error(err, fmt.Sprintf("can not get the particular earliest offset for topic partition - %s, %d", topic, partition))
				return messagesCount, err
			}
			messagesCount[fmt.Sprintf("%s-%d", topic, partition)] = calculateMessagesCount(latestOffset, earliestOffset)
		}
	}
	return messagesCount, nil
}

func getMessagesCountWrapped(wg *sync.WaitGroup, ch chan map[string]int64, client sarama.Client, topics []string) {
	defer wg.Done()
	wg.Add(1)
	messagesCount, _ := getMessagesCount(client, topics)
	ch <- messagesCount
}

func calculateMessagesCount(latestOffset int64, earliestOffset int64) int64 {
	if earliestOffset > latestOffset {
		return 0
	}
	if latestOffset < 0 {
		latestOffset = 0
	}
	if earliestOffset < 0 {
		earliestOffset = 0
	}
	return latestOffset - earliestOffset
}

func flushOut(activePartitions map[string]int64, standByPartitions map[string]int64) map[string]int64 {
	result := make(map[string]int64)
	for activePartition, valueActive := range activePartitions {
		if valueStandby, ok := standByPartitions[activePartition]; !ok {
			result[activePartition] = valueActive
		} else if valueActive > valueStandby {
			result[activePartition] = valueActive
		}
	}
	return result
}

func getTopicList(m map[string]int64) []string {
	var topics []string
	for key := range m {
		index := strings.LastIndex(key, "-")
		topics = append(topics, key[:index])
	}
	return topics
}
