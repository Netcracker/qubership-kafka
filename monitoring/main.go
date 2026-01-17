//Copyright 2024-2025 NetCracker Technology Corporation
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/scram"
)

var (
	broker         = getEnv("KAFKA_ADDRESSES", "")
	saslUsername   = getEnv("KAFKA_USER", "")
	saslPassword   = getEnv("KAFKA_PASSWORD", "")
	serviceName    = getEnv("KAFKA_SERVICE_NAME", "kafka")
	sslMechanism   = getEnv("KAFKA_SASL_MECHANISM", "SCRAM-SHA-512")
	isDebugEnabled = getBoolEnv("KAFKA_MONITORING_SCRIPT_DEBUG", "false")
	sslEnabled     = getBoolEnv("KAFKA_ENABLE_SSL", "false")

	caCertPath     = "/tls/ca.crt"
	tlsCertPath    = "/tls/tls.crt"
	tlsKeyPath     = "/tls/tls.key"
	debugLogger, _ = initLogger("/opt/kafka-monitoring/exec-scripts/additional-metrics.log")
)

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue string) bool {
	return strings.ToLower(getEnv(key, defaultValue)) == "true"
}

type Logger struct {
	logger *log.Logger
	debug  bool
}

func initLogger(logFile string) (*Logger, error) {
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}

	logger := log.New(file, "", 0)

	return &Logger{logger: logger, debug: isDebugEnabled}, nil
}

func (c *Logger) Debug(message string) {
	if c.debug {
		// formatting the log message same as in kafka_metric.py
		timestamp := time.Now().Format("2006-01-02T15:04:05,000")
		logMessage := fmt.Sprintf("[%s][DEBUG] %s", timestamp, message)
		c.logger.Println(logMessage)
	}
}

func main() {

	debugLogger.Debug("Start of additional metrics script execution")
	startTime := time.Now().UnixMilli()

	brokerAddress := strings.Split(broker, ",")
	username := saslUsername
	password := saslPassword

	var mechanism sasl.Mechanism
	if username != "" && password != "" {
		if sslMechanism == "SCRAM-SHA-512" {
			mechanism, _ = scram.Mechanism(scram.SHA512, username, password)
		} else {
			mechanism, _ = scram.Mechanism(scram.SHA256, username, password)
		}
	}

	transport := kafka.Transport{
		SASL: mechanism,
	}

	if sslEnabled {
		tlsConfig, err := createTLSConfig(caCertPath, tlsCertPath, tlsKeyPath)
		if err != nil {
			log.Fatalf("Failed to create TLS config: %s", err)
		}
		transport.TLS = tlsConfig
	}

	client := kafka.Client{
		Addr:      kafka.TCP(brokerAddress...),
		Transport: &transport,
	}

	metadata, err := client.Metadata(context.Background(), &kafka.MetadataRequest{})
	if err != nil {
		log.Fatalf("Failed to get metadata: %s", err)
	}

	var resources []kafka.DescribeConfigRequestResource
	for _, topic := range metadata.Topics {
		resources = append(resources, kafka.DescribeConfigRequestResource{
			ResourceType: kafka.ResourceTypeTopic,
			ResourceName: topic.Name,
		})
	}

	request := &kafka.DescribeConfigsRequest{
		Resources: resources,
	}

	configs := &kafka.DescribeConfigsResponse{}

	configs, err = client.DescribeConfigs(context.Background(), request)
	if err != nil {
		log.Fatalf("Failed to describe configs: %s", err)
	}

	fmt.Println(CollectMetrics(*configs, *metadata, client))

	debugLogger.Debug(fmt.Sprintf("Time of additional metrics script execution: %f", float64((time.Now().UnixMilli()-startTime))/float64(1000)))

}

func CollectMetrics(configs kafka.DescribeConfigsResponse, metadata kafka.MetadataResponse, client kafka.Client) string {
	uncleanTopics := strings.Join(calcUncleanElectionLeaderTopics(configs), "\n")
	topicsWithoutLeader := strings.Join(CalcTopicsWithoutLeader(metadata.Topics), "\n")
	brokerMetrics := CalcSkewNFormMetric(metadata.Topics, CalcCleanerNReplicaThreads(metadata, client))
	result := brokerMetrics + uncleanTopics + "\n" + topicsWithoutLeader
	return result
}

func calcUncleanElectionLeaderTopics(configs kafka.DescribeConfigsResponse) []string {
	pattern := "kafka_cluster,topic=%s unclean_election_topics=1"
	metricsArr := make([]string, 0)
	for _, res := range configs.Resources {
		uncleanElectionEnabled := false
		for _, entry := range res.ConfigEntries {
			if entry.ConfigName == "unclean.leader.election.enable" && entry.ConfigValue == "true" {
				uncleanElectionEnabled = true
				break
			}
		}
		if uncleanElectionEnabled {
			metricsArr = append(metricsArr, fmt.Sprintf(pattern, res.ResourceName))
		}
	}
	// kafka_cluster unclean_election_topics=1,topic="test0"
	return metricsArr
}

func CalcTopicsWithoutLeader(topics []kafka.Topic) []string {
	pattern := "kafka_cluster,topic=%s topics_without_leader=1"
	topicsWithoutLeader := make([]string, 0)
	for _, topic := range topics {

		for _, partition := range topic.Partitions {
			if partition.Leader.ID == 0 {
				topicsWithoutLeader = append(topicsWithoutLeader, fmt.Sprintf(pattern, topic.Name))
				break
			}
		}
	}
	return topicsWithoutLeader
}

func CalcSkewNFormMetric(topics []kafka.Topic, result map[string]string) string {
	brokerPartitions := make(map[kafka.Broker]int)
	brokerLeaders := make(map[kafka.Broker]int)
	for _, topic := range topics {
		for _, partition := range topic.Partitions {
			brokerLeaders[partition.Leader]++
			for _, replica := range partition.Replicas {
				brokerPartitions[replica]++
			}
		}
	}

	var sumPartitions, sumLeaders float64
	for _, count := range brokerPartitions {
		sumPartitions += float64(count)
	}
	for _, count := range brokerLeaders {
		sumLeaders += float64(count)
	}

	numBrokers := float64(len(brokerPartitions))
	meanPartitions := sumPartitions / numBrokers
	meanLeaders := sumLeaders / numBrokers

	for broker, count := range brokerPartitions {
		deviation := ((float64(count) - meanPartitions) / meanPartitions) * 100
		brokerSignature := fmt.Sprintf("%s-%d", serviceName, broker.ID)
		result[brokerSignature] += fmt.Sprintf(",broker_skew=%di", int(deviation))
	}

	for broker, count := range brokerLeaders {
		deviation := ((float64(count) - meanLeaders) / meanLeaders) * 100
		brokerSignature := fmt.Sprintf("%s-%d", serviceName, broker.ID)
		result[brokerSignature] += fmt.Sprintf(",broker_leader_skew=%di", int(deviation))
	}

	metrics := new(bytes.Buffer)
	for _, metricPerBroker := range result {
		if strings.Contains(metricPerBroker, "kafka,broker=") {
			_, err := fmt.Fprintf(metrics, "%s\n", metricPerBroker)
			if err != nil {
				return ""
			}
		}
	}
	return metrics.String()
}

func CalcCleanerNReplicaThreads(metadata kafka.MetadataResponse, client kafka.Client) map[string]string {
	result := make(map[string]string)
	var resources []kafka.DescribeConfigRequestResource
	for _, broker := range metadata.Brokers {
		brokerSignature := fmt.Sprintf("%s-%d", serviceName, broker.ID)
		result[brokerSignature] = fmt.Sprintf("kafka,broker=%s ", brokerSignature)
		resources = append(resources, kafka.DescribeConfigRequestResource{
			ResourceType: kafka.ResourceTypeBroker,
			ResourceName: fmt.Sprintf("%d", broker.ID),
		})
	}
	request := &kafka.DescribeConfigsRequest{
		Resources: resources,
	}
	var cleanerVal string
	var fetchersVal string
	response, err := client.DescribeConfigs(context.Background(), request)
	if err != nil {
		log.Fatalf("Failed to describe configs: %s", err)
	}
	for _, resource := range response.Resources {
		brokerSignature := fmt.Sprintf("%s-%s", serviceName, resource.ResourceName)
		for _, config := range resource.ConfigEntries {
			if config.ConfigName == "log.cleaner.threads" {
				cleanerVal = config.ConfigValue
			}
			if config.ConfigName == "num.replica.fetchers" {
				fetchersVal = config.ConfigValue
			}
		}
		result[brokerSignature] += fmt.Sprintf("log_cleaner_threads_count=%si,replica_fetcher_threads_count=%si", cleanerVal, fetchersVal)
	}
	return result
}

func createTLSConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
	var cert tls.Certificate
	var err error
	if fileExists(certFile) && fileExists(keyFile) {
		cert, err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("could not load client key pair: %w", err)
		}
	}

	var caCertPool x509.CertPool
	if fileExists(caFile) {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("could not read ca certificate: %w", err)
		}

		caCertPool = *x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return nil, fmt.Errorf("failed to append ca cert to pool")
		}
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            &caCertPool,
		InsecureSkipVerify: false,
	}

	return tlsConfig, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
