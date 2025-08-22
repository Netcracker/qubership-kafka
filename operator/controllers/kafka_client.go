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

package controllers

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/Netcracker/qubership-kafka/operator/util"
)

type KafkaClient struct {
	serviceName                     string
	clientUsername                  string
	clientPassword                  string
	newBrokersCount                 int32
	allBrokersStartTimeoutSeconds   int
	topicReassignmentTimeoutSeconds int
	adminClient                     sarama.ClusterAdmin
	controllerId                    int32
	racksEnabled                    bool
	brokerRacks                     map[int32]string
}

type SaslSettings struct {
	Mechanism string
	Username  string
	Password  string
}

type SslCertificates struct {
	CaCert  []byte
	TlsCert []byte
	TlsKey  []byte
}

type BrokerInfo struct {
	brokerId        int32
	partitionsCount int64
	rack            string
	skew            int32
}

type TopicInfo struct {
	topicName string
	configs   sarama.TopicDetail
}

const (
	MaxRetryAttempts = 5
	RetryDelay       = 60 * time.Second
)

func NewKafkaClient(
	serviceName string,
	clientUsername string,
	clientPassword string,
	sslEnabled bool,
	sslCertificates *SslCertificates,
	newBrokersCount int32,
	allBrokersStartTimeoutSeconds int,
	topicReassignmentTimeoutSeconds int) (*KafkaClient, error) {
	saslSettings := &SaslSettings{
		Mechanism: sarama.SASLTypeSCRAMSHA512,
		Username:  clientUsername,
		Password:  clientPassword,
	}
	adminClient, err := NewKafkaAdminClient(fmt.Sprintf("%s:9092", serviceName), saslSettings, sslEnabled, sslCertificates)
	if err != nil {
		return nil, err
	}
	return &KafkaClient{
		serviceName:                     serviceName,
		clientUsername:                  clientUsername,
		clientPassword:                  clientPassword,
		newBrokersCount:                 newBrokersCount,
		allBrokersStartTimeoutSeconds:   allBrokersStartTimeoutSeconds,
		topicReassignmentTimeoutSeconds: topicReassignmentTimeoutSeconds,
		adminClient:                     adminClient,
	}, nil
}

func NewKafkaAdminClient(
	bootstrapServers string, saslSettings *SaslSettings, sslEnabled bool, sslCertificates *SslCertificates) (sarama.ClusterAdmin, error) {
	address := strings.Split(bootstrapServers, ",")
	config, err := NewKafkaClientConfig(saslSettings, sslEnabled, sslCertificates)
	if err != nil {
		return nil, err
	}
	adminClient, err := sarama.NewClusterAdmin(address, config)
	if err != nil {
		return nil, err
	}
	return adminClient, nil
}

func NewKafkaClientConfig(saslSettings *SaslSettings, sslEnabled bool, sslCertificates *SslCertificates) (*sarama.Config, error) {
	config := sarama.NewConfig()
	if saslSettings.Username != "" && saslSettings.Password != "" {
		log.Info("Configuring SASL...")
		mechanism := util.DefaultIfEmpty(saslSettings.Mechanism, sarama.SASLTypeSCRAMSHA512)
		if mechanism != sarama.SASLTypePlaintext && mechanism != sarama.SASLTypeSCRAMSHA512 {
			return nil, fmt.Errorf("cannot use given SASL Mechanism: %s", mechanism)
		}
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLMechanism(mechanism)
		config.Net.SASL.User = saslSettings.Username
		config.Net.SASL.Password = saslSettings.Password
		if mechanism == sarama.SASLTypeSCRAMSHA512 {
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
			}
		}
		log.Info("SASL configuration is applied")
	}
	if sslEnabled {
		log.Info("Configuring TLS...")
		if len(sslCertificates.CaCert) != 0 {
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(sslCertificates.CaCert)
			tlsConfig := &tls.Config{
				RootCAs: caCertPool,
			}
			if len(sslCertificates.TlsCert) != 0 && len(sslCertificates.TlsKey) != 0 {
				log.Info("Configuring mTLS...")
				cert, err := tls.X509KeyPair(sslCertificates.TlsCert, sslCertificates.TlsKey)
				if err != nil {
					return nil, err
				}
				tlsConfig.Certificates = []tls.Certificate{cert}
				log.Info("mTLS configuration is applied")
			}
			config.Net.TLS.Config = tlsConfig
		}
		config.Net.TLS.Enable = true
		log.Info("TLS configuration is applied")
	}
	config.Version = sarama.V3_6_0_0
	return config, nil
}

func (kc *KafkaClient) ReassignPartitions() error {
	err := kc.WaitUntilAllBrokersAreUp()
	if err != nil {
		return err
	}

	topics, err := kc.adminClient.ListTopics()
	if err != nil {
		return err
	}

	topicsCount := len(topics)
	var counter = 1
	for topic, config := range topics {
		log.Info(fmt.Sprintf("%d of %d: Trying to reassign partitions for topic %s...", counter, topicsCount, topic))

		globalPartitionCount, brokersInfo, err := kc.calculatePartitionsCount(topics)
		if err != nil {
			log.Error(err, "Failed to calculate broker partitions")
		}

		topicWithConfig := TopicInfo{topicName: topic, configs: config}

		kc.calculateBrokersSkew(globalPartitionCount, brokersInfo)
		skewIsNormal := checkBrokersSkewIsNormal(brokersInfo)
		if skewIsNormal {
			log.Info("Partitions are evenly distributed between all brokers.")
			return nil
		}
		err = kc.ReassignPartitionsForTopic(topicWithConfig, brokersInfo, globalPartitionCount)
		if err != nil {
			log.Error(err, fmt.Sprintf("Cannot reassign partitions for topic [%s]", topic))
		}
		counter++
	}
	return nil
}

func (kc *KafkaClient) calculatePartitionsCount(topics map[string]sarama.TopicDetail) (int64, []*BrokerInfo, error) {
	brokersPartitions := make(map[int32]int32, kc.newBrokersCount)
	brokersInfo := make([]*BrokerInfo, kc.newBrokersCount)
	globalPartitionCount := 0
	topicsTitles := make([]string, 0, len(topics))
	for topic := range topics {
		topicsTitles = append(topicsTitles, topic)
	}
	var metadata []*sarama.TopicMetadata
	var err error
	var attempt int
	for {
		metadata, err = kc.adminClient.DescribeTopics(topicsTitles)
		if err == nil {
			break
		}
		attempt++
		if attempt > MaxRetryAttempts {
			log.Error(err, "Maximum attempts exceeded when trying to describe topics")
		}
		time.Sleep(RetryDelay)
	}
	for _, topic := range metadata {
		for _, partition := range topic.Partitions {
			globalPartitionCount++
			brokersPartitions[partition.Leader]++
		}
	}
	for brokerId := 1; brokerId <= int(kc.newBrokersCount); brokerId++ {
		brokersInfo[brokerId-1] = kc.newBrokerInfo(int32(brokerId), int64(brokersPartitions[int32(brokerId)]))
		log.Info(fmt.Sprintf("PartitionCount for broker %d is %v", brokerId, brokersInfo[brokerId-1]))
	}
	log.Info(fmt.Sprintf("GlobalPartitionCount = %d", globalPartitionCount))

	return int64(globalPartitionCount), brokersInfo, nil
}

func (kc *KafkaClient) WaitUntilAllBrokersAreUp() error {
	log.Info("Waiting for all brokers are up...")
	count := 0
	remainingTime := kc.allBrokersStartTimeoutSeconds
	for {
		brokers, controller, err := kc.GetActiveBrokers()
		if err != nil {
			log.Error(err, "cannot get active brokers")
			count = -1
		} else {
			count = len(brokers)
		}
		if count >= int(kc.newBrokersCount) {
			kc.controllerId = controller
			log.Info(fmt.Sprintf("Brokers count is %d, controller id is %d", count, controller))

			racksEnabled := false
			brokerRacks := make(map[int32]string)
			for _, broker := range brokers {
				brokerRacks[broker.ID()] = broker.Rack()
				if broker.Rack() != "" {
					racksEnabled = true
				}
			}
			kc.brokerRacks = brokerRacks
			kc.racksEnabled = racksEnabled
			return nil
		} else {
			time.Sleep(10 * time.Second)
			remainingTime = remainingTime - 10
			if remainingTime <= 0 {
				return fmt.Errorf("not all brokers are started, timeout is expired")
			}
		}
	}
}

func (kc *KafkaClient) GetActiveBrokers() ([]*sarama.Broker, int32, error) {
	return kc.adminClient.DescribeCluster()
}

func (kc *KafkaClient) ReassignPartitionsForTopic(topic TopicInfo, brokersInfo []*BrokerInfo, globalPartitionCount int64) error {
	newReplicaAssignment := CopyCurrentReplicaAssignment(topic)
	previousAssignment := -1
	for partition := 0; partition < int(topic.configs.NumPartitions); partition++ {
		kc.calculateBrokersSkew(globalPartitionCount, brokersInfo)

		sort.Slice(brokersInfo, func(i, j int) bool {
			return brokersInfo[i].skew > brokersInfo[j].skew
		})

		brokersWithMostPartitions, brokersWithLeastPartitions := kc.PrepareBrokersWithMostAndLeastPartitions(brokersInfo)

		brokerWithMostPartitionsToSwap, idx := CalcBrokerWithMostPartitionsToSwap(brokersWithMostPartitions, newReplicaAssignment, int32(partition))
		if brokerWithMostPartitionsToSwap == -1 {
			continue
		}
		brokerWithLeastPartitionsToSwap := kc.CalcBrokerWithLeastPartitionsToSwap(brokersWithLeastPartitions, newReplicaAssignment, int32(partition), int32(brokerWithMostPartitionsToSwap))

		if brokerWithLeastPartitionsToSwap == -1 {
			continue
		}

		if topic.configs.ReplicationFactor == 1 && previousAssignment == brokerWithLeastPartitionsToSwap {
			previousAssignment = -1
			continue
		} else {
			previousAssignment = brokerWithLeastPartitionsToSwap
		}

		newReplicaAssignment[partition][idx] = int32(brokerWithLeastPartitionsToSwap)
		UpdateBrokersInfoWithNewPartitionsDistribution(brokersInfo, brokerWithMostPartitionsToSwap, brokerWithLeastPartitionsToSwap)
	}

	log.Info(fmt.Sprintf("New assignment for topic %s is: %v", topic.topicName, newReplicaAssignment))
	err := kc.adminClient.AlterPartitionReassignments(topic.topicName, newReplicaAssignment)
	if err != nil {
		log.Error(err, fmt.Sprintf("Cannot reassign partitions for topic [%s]", topic.topicName))
	}

	kc.WaitUntilPartitionsReassignmentFinished(topic)
	return nil
}

func UpdateBrokersInfoWithNewPartitionsDistribution(brokersInfo []*BrokerInfo, brokerWithMostPartitionsToSwap int, brokerWithLeastPartitionsToSwap int) {
	for _, broker := range brokersInfo {
		if broker.brokerId == int32(brokerWithMostPartitionsToSwap) {
			broker.partitionsCount = broker.partitionsCount - 1
		} else if broker.brokerId == int32(brokerWithLeastPartitionsToSwap) {
			broker.partitionsCount = broker.partitionsCount + 1
		}
	}
}

func (kc *KafkaClient) WaitUntilPartitionsReassignmentFinished(topic TopicInfo) {
	partitions := make([]int32, topic.configs.NumPartitions)
	for i := 0; i < int(topic.configs.NumPartitions); i++ {
		partitions[i] = int32(i)
	}

	remaining := kc.topicReassignmentTimeoutSeconds
	for {
		reassignmentStatus, err := kc.adminClient.ListPartitionReassignments(topic.topicName, partitions)
		if err == nil && len(reassignmentStatus[topic.topicName]) == 0 {
			return
		}
		time.Sleep(2 * time.Second)
		remaining = remaining - 2
		if remaining <= 0 {
			return
		}
	}
}

func CopyCurrentReplicaAssignment(topic TopicInfo) [][]int32 {
	newReplicaAssignment := make([][]int32, topic.configs.NumPartitions)
	rf := int(topic.configs.ReplicationFactor)
	log.Info(fmt.Sprintf("Current assignment for topic %s is: %v", topic.topicName, topic.configs.ReplicaAssignment))
	for partition := 0; partition < int(topic.configs.NumPartitions); partition++ {
		newReplicaAssignment[partition] = make([]int32, rf)
		for replica := 0; replica < rf; replica++ {
			newReplicaAssignment[partition][replica] = topic.configs.ReplicaAssignment[int32(partition)][replica]
		}
	}
	return newReplicaAssignment
}

func (kc *KafkaClient) PrepareBrokersWithMostAndLeastPartitions(brokersInfo []*BrokerInfo) ([]*BrokerInfo, []*BrokerInfo) {
	threshold := int(kc.newBrokersCount)
	for i := 0; i < int(kc.newBrokersCount); i++ {
		if brokersInfo[i].skew < 0 {
			threshold = i
			break
		}
	}

	brokersWithMostPartitions := make([]*BrokerInfo, threshold)
	for i := 0; i < threshold; i++ {
		brokersWithMostPartitions[i] = brokersInfo[i]
	}
	brokersWithLeastPartitions := make([]*BrokerInfo, int(kc.newBrokersCount)-threshold)
	for i := threshold; i < int(kc.newBrokersCount); i++ {
		brokersWithLeastPartitions[i-threshold] = brokersInfo[i]
	}

	sort.Slice(brokersWithLeastPartitions, func(i, j int) bool {
		return brokersWithLeastPartitions[i].skew < brokersWithLeastPartitions[j].skew
	})
	return brokersWithMostPartitions, brokersWithLeastPartitions
}

func CalcBrokerWithMostPartitionsToSwap(brokersWithMostPartitions []*BrokerInfo, newReplicaAssignment [][]int32, currentPartition int32) (int, int) {
	brokerWithMostPartitionsToSwap := -1
	for _, broker := range brokersWithMostPartitions {
		for idx, replica := range newReplicaAssignment[currentPartition] {
			if broker.brokerId == replica {
				brokerWithMostPartitionsToSwap = int(replica)
				return brokerWithMostPartitionsToSwap, idx
			}
		}
	}
	return brokerWithMostPartitionsToSwap, -1
}

func (kc *KafkaClient) CalcBrokerWithLeastPartitionsToSwap(brokersWithLeastPartitions []*BrokerInfo, newReplicaAssignment [][]int32, currentPartition int32, brokerWithMostPartitionsToSwap int32) int {
	brokerWithLeastPartitionsToSwap := -1

	if kc.racksEnabled {
		currentRacks := kc.getRacksForBrokers(newReplicaAssignment[currentPartition])

		for _, broker := range brokersWithLeastPartitions {
			if !containsString(currentRacks, broker.rack) &&
				!containsInt32(newReplicaAssignment[currentPartition], broker.brokerId) {
				brokerWithLeastPartitionsToSwap = int(broker.brokerId)
				return brokerWithLeastPartitionsToSwap
			}
		}

		if brokerWithLeastPartitionsToSwap == -1 {
			for _, broker := range brokersWithLeastPartitions {
				if kc.brokerRacks[brokerWithMostPartitionsToSwap] == broker.rack &&
					!containsInt32(newReplicaAssignment[currentPartition], broker.brokerId) {
					brokerWithLeastPartitionsToSwap = int(broker.brokerId)
					return brokerWithLeastPartitionsToSwap
				}
			}
		}
	}

	if brokerWithLeastPartitionsToSwap == -1 {
		for _, broker := range brokersWithLeastPartitions {
			if !containsInt32(newReplicaAssignment[currentPartition], broker.brokerId) {
				brokerWithLeastPartitionsToSwap = int(broker.brokerId)
				return brokerWithLeastPartitionsToSwap
			}
		}
	}

	return brokerWithLeastPartitionsToSwap
}

func (kc *KafkaClient) getRacksForBrokers(brokers []int32) []string {
	racks := make([]string, len(brokers))
	for broker := 0; broker < len(brokers); broker++ {
		racks[broker] = kc.brokerRacks[int32(broker)]
	}
	return racks
}

func (kc *KafkaClient) newBrokerInfo(brokerId int32, partitionsCount int64) *BrokerInfo {
	return &BrokerInfo{
		brokerId:        brokerId,
		partitionsCount: partitionsCount,
		rack:            kc.brokerRacks[brokerId],
	}
}

func (kc *KafkaClient) calculateBrokersSkew(globalPartitionCount int64, brokersInfo []*BrokerInfo) {
	brokersSkew := make(map[int32]int32)
	partitionsOnBrokersSum := int64(0)

	for _, broker := range brokersInfo {
		partitionsOnBrokersSum = partitionsOnBrokersSum + broker.partitionsCount
	}

	averageNumOfPartitionsPerBroker := RoundFloatTo2Decimals(float64(partitionsOnBrokersSum) / float64(kc.newBrokersCount))
	maxPossibleSkew := Max(float64(globalPartitionCount)-averageNumOfPartitionsPerBroker, averageNumOfPartitionsPerBroker)

	for _, broker := range brokersInfo {
		brokerSkew := float64(broker.partitionsCount) - averageNumOfPartitionsPerBroker
		if brokerSkew == 0 || maxPossibleSkew == 0 {
			brokersSkew[broker.brokerId] = 0
		} else {
			brokersSkew[broker.brokerId] = int32(RoundFloatTo2Decimals(brokerSkew/maxPossibleSkew) * 100)
		}
	}
	for _, broker := range brokersInfo {
		broker.skew = brokersSkew[broker.brokerId]
	}
}

func checkBrokersSkewIsNormal(brokersInfo []*BrokerInfo) bool {
	for _, broker := range brokersInfo {
		if Abs(broker.skew) > 5 {
			return false
		}
	}
	return true
}

func Max(x, y float64) float64 {
	if x < y {
		return y
	}
	return x
}

func RoundFloatTo2Decimals(x float64) float64 {
	return math.Round(x*100) / 100
}

func Abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

func containsString(array []string, element string) bool {
	for _, a := range array {
		if a == element {
			return true
		}
	}
	return false
}

func containsInt32(array []int32, element int32) bool {
	for _, a := range array {
		if a == element {
			return true
		}
	}
	return false
}
