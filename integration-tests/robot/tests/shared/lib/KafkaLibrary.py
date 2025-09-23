# Copyright 2024-2025 NetCracker Technology Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import re
import time

import requests
from kafka import KafkaConsumer, KafkaProducer
from kafka.admin import (NewTopic, NewPartitions, KafkaAdminClient, ACL, ACLFilter, ResourceType,
                         ACLOperation, ACLPermissionType, ResourcePattern, ConfigResource, ConfigResourceType)
from kafka.sasl.oauth import AbstractTokenProvider
from robot.api import logger
from robot.libraries.BuiltIn import BuiltIn

CA_CERT_PATH = '/tls/ca.crt'
TLS_CERT_PATH = '/tls/tls.crt'
TLS_KEY_PATH = '/tls/tls.key'


def _str2bool(v: str) -> bool:
    return v.lower() in ("yes", "true", "t", "1")


class KafkaLibrary(object):
    """
    Kafka testing library for Robot Framework.
    """

    def __init__(self, bootstrap_servers, namespace, host, port, username, password, enable_ssl):
        if not bootstrap_servers:
            if namespace:
                bootstrap_servers = f'{host}.{namespace}:{port}'
            else:
                bootstrap_servers = f'{host}:{port}'
        self._common_configs = {
            'bootstrap_servers': bootstrap_servers.split(','),
            'api_version': (2, 7, 1)
        }
        kafka_enable_ssl = _str2bool(enable_ssl)
        security_protocol = 'SSL' if kafka_enable_ssl else 'PLAINTEXT'
        if username and password:
            self._common_configs['security_protocol'] = f'SASL_{security_protocol}'
        else:
            self._common_configs['security_protocol'] = security_protocol
        self._kafka_username = username
        self._kafka_password = password
        self._common_configs['ssl_cafile'] = \
            CA_CERT_PATH if kafka_enable_ssl and os.path.exists(CA_CERT_PATH) else None
        self._common_configs['ssl_certfile'] = \
            TLS_CERT_PATH if kafka_enable_ssl and os.path.exists(TLS_CERT_PATH) else None
        self._common_configs['ssl_keyfile'] = \
            TLS_KEY_PATH if kafka_enable_ssl and os.path.exists(TLS_KEY_PATH) else None
        self.builtin = BuiltIn()

    def create_kafka_producer(self):
        """
        Creates Kafka producer. If security is enabled, received credentials are used.
        *Args:*\n
        *Returns:*\n
            Producer - Kafka producer
        *Example:*\n
            | Create Kafka Producer |
        """
        producer = None
        try:
            configs = dict(self._common_configs.items())
            configs['retries'] = 10000000
            configs['max_in_flight_requests_per_connection'] = 1
            configs['request_timeout_ms'] = 90000
            configs['connections_max_idle_ms'] = 1000000
            if self._kafka_username and self._kafka_password:
                configs['sasl_mechanism'] = 'SCRAM-SHA-512'
                configs['sasl_plain_username'] = self._kafka_username
                configs['sasl_plain_password'] = self._kafka_password
            producer = KafkaProducer(**configs)
            logger.debug("Kafka producer is created.")
        except Exception as e:
            self.builtin.fail(f'Exception while connecting to Kafka: {e}')
        return producer

    def create_kafka_consumer(self, topic_name):
        """
        Creates Kafka consumer. If security is enabled, received credentials are used.
        *Args:*\n
            _topic_name_ (str) - name of topic;\n
        *Returns:*\n
            Consumer - Kafka consumer
        *Example:*\n
            | Create Kafka Consumer | consumer-producer-tests-topic |
        """
        consumer = None
        try:
            configs = dict(self._common_configs.items())
            configs['auto_offset_reset'] = 'earliest'
            configs['enable_auto_commit'] = False
            configs['group_id'] = f'{topic_name}-group-{time.time()}'
            configs['request_timeout_ms'] = 90000
            configs['connections_max_idle_ms'] = 1000000
            if self._kafka_username and self._kafka_password:
                configs['sasl_mechanism'] = 'SCRAM-SHA-512'
                configs['sasl_plain_username'] = self._kafka_username
                configs['sasl_plain_password'] = self._kafka_password
            consumer = KafkaConsumer(topic_name, **configs)
            logger.debug("Kafka consumer is created.")
        except Exception as e:
            self.builtin.fail(f'Exception while connecting to Kafka: {e}')
        return consumer

    def create_admin_client(self):
        """
        Creates Kafka admin client. If security is enabled, received credentials are used.
        *Args:*\n
        *Returns:*\n
            AdminClient - Kafka admin client
        *Example:*\n
            | Create Admin Client |
        """
        admin = None
        try:
            configs = dict(self._common_configs.items())
            configs['metadata_max_age_ms'] = 1000
            if self._kafka_username and self._kafka_password:
                configs['sasl_mechanism'] = 'SCRAM-SHA-512'
                configs['sasl_plain_username'] = self._kafka_username
                configs['sasl_plain_password'] = self._kafka_password
                configs['request_timeout_ms'] = 90000
            admin = KafkaAdminClient(**configs)
            logger.debug("Kafka admin client is created.")
        except Exception as e:
            self.builtin.fail(f'Exception while connecting to Kafka: {e}')
        return admin

    def close_kafka_consumer(self, consumer):
        """
        Closes necessary Kafka consumer.
        *Args:*\n
             _consumer_ (Consumer) - Kafka consumer to close;\n
        *Example:*\n
            | Close Kafka Consumer | consumer |
        """
        if consumer is not None:
            consumer.unsubscribe()
            consumer.close()

    def create_test_message(self):
        """
        Creates a message to send to Kafka. It contains current timestamp in seconds.
        *Args:*\n
        *Returns:*\n
            str - created message
        *Example:*\n
            | Create Test Message |
        """
        message = f'Kafka msg: {time.time()}'
        return message

    def produce_message(self, producer: KafkaProducer, topic_name, message):
        """
        Sends the message to Kafka using Kafka producer.
        *Args:*\n
             _producer_ (Producer) - Kafka producer;\n
             _topic_name_ (str) - name of topic;\n
             _message_ (str) - message to send;\n
        *Example:*\n
            | Produce Message | producer | consumer-producer-tests-topic | 1541506923 |
        """
        try:
            # producer.send(topic_name, message.encode('utf-8'))
            # producer.flush(timeout=10)
            future = producer.send(topic_name, message.encode("utf-8"))
            record_metadata = future.get(timeout=10)
            producer.flush(timeout=10)
            self.builtin.log(
                f"Produced to {record_metadata.topic} partition {record_metadata.partition} offset {record_metadata.offset}",
                "INFO"
            )
        except Exception as e:
            self.builtin.log(traceback.format_exc(), "ERROR")
            self.builtin.fail(f'Failed to produce message: "{message}" to '
                              f'topic: {topic_name} {e}')

    def consume_message(self, consumer, topic_name: str) -> str:
        """
        Receives a message from Kafka using Kafka consumer.
        *Args:*\n
             _consumer_ (Consumer) - Kafka consumer;\n
        *Returns:*\n
            bytes - received message
        *Example:*\n
            | Consume Message | consumer |
        """
        consumer.subscribe([topic_name])
        message = consumer.poll(10.0)
        if message:
            return str(message)
        else:
            logger.debug(f'Received message is "{message}".')
            return ""

    def get_brokers_count(self, admin: KafkaAdminClient) -> int:
        """
        Receives the count of Kafka brokers using Kafka admin client.
        *Args:*\n
             _admin_ (AdminClient) - Kafka admin client;\n
        *Returns:*\n
            int - Kafka brokers count
        *Example:*\n
            | Get Brokers Count | admin |
        """
        return len(admin.describe_cluster()['brokers'])

    def get_topics_list(self, admin, topic_name_pattern=None):
        """
        Receives the list of topics from Kafka using Kafka admin client.
        *Args:*\n
             _admin_ (AdminClient) - Kafka admin client;\n
             _topic_name_pattern_ (str) - topic name pattern;\n
        *Returns:*\n
            list - list of topics
        *Example:*\n
            | Get Topics List | admin | kafka-topic-* |
        """
        topics = admin.list_topics()
        result = []
        for topic_name in topics:
            if topic_name_pattern is not None:
                if re.fullmatch(topic_name_pattern, topic_name):
                    result.append(topic_name)
            else:
                result.append(topic_name)
        return result

    def get_topic_info(self, admin, topic_name):
        """
        Receives the information about the topic.
        *Args:*\n
             _admin_ (AdminClient) - Kafka admin client;\n
             _topic_name_ (str) - name of topic;\n
        *Returns:*\n
            TopicMetadata - information about specific topic
        *Example:*\n
            | Get Topic Info | admin | kafka-topic-tests |
        """
        topic = admin.describe_topics(topics=[topic_name])
        return topic

    def create_topic(self, admin, topic_name, replication_factor, partitions, configs=None):
        """
        Creates topic with specific name.
        *Args:*\n
             _admin_ (AdminClient) - Kafka admin client;\n
             _topic_name_ (str) - name of topic;\n
             _replication_factor_ (int) - replication factor;\n
             _partitions_ (int) - number of partitions;\n
        *Example:*\n
            | Create Topic | admin | kafka-topic-tests | ${1} | ${1} |
        """
        topics = self.get_topics_list(admin)
        if topic_name in topics:
            logger.debug(f'Topic "{topic_name}" has already been created.')
            return
        new_topic = NewTopic(topic_name,
                             num_partitions=partitions,
                             replication_factor=replication_factor,
                             topic_configs=configs)
        try:
            admin.create_topics([new_topic])
            logger.debug(f'Topic "{topic_name}" is created.')
        except Exception as e:
            self.builtin.fail(f'Failed to create topic "{topic_name}": {e}')

    def create_topic_with_expected_exception(self, admin, topic_name, replication_factor, partitions):
        """
        Tries to create topic with specific name and returns an exception message (empty string by default).
        *Args:*\n
             _admin_ (AdminClient) - Kafka admin client;\n
             _topic_name_ (str) - name of topic;\n
             _replication_factor_ (int) - replication factor;\n
             _partitions_ (int) - number of partitions;\n
        *Example:*\n
            | Create Topic With Expected Exception | admin | kafka-topic-tests | ${1} | ${1} |
        """
        error_message = ''
        topics = self.get_topics_list(admin)
        if topic_name in topics:
            logger.debug(f'Topic "{topic_name}" has already been created.')
            return error_message
        topic_configs = {'min.insync.replicas': 1} if int(replication_factor) == 1 else {}
        new_topic = NewTopic(topic_name,
                             num_partitions=partitions,
                             replication_factor=replication_factor,
                             topic_configs=topic_configs)
        try:
            admin.create_topics([new_topic])
            logger.debug(f'Topic "{topic_name}" is created.')
        except Exception as e:
            error_message = str(e)
        return error_message

    def create_partitions(self, admin, topic_name, count):
        """
        Creates additional partitions for specific topic.
        *Args:*\n
             _admin_ (AdminClient) - Kafka admin client;\n
             _topic_name_ (str) - name of topic;\n
             _count_ (int) - number of partitions;\n
        *Example:*\n
            | Create Partitions | admin | kafka-topic-tests | ${3} |
        """
        try:
            topic_partitions = {topic_name: NewPartitions(total_count=count)}
            admin.create_partitions(topic_partitions)
            logger.debug(f'Additional partitions for topic "{topic_name}" are created.')
        except Exception as e:
            self.builtin.fail(f'Failed to add partitions to topic "{topic_name}": {e}')

    def get_partitions_count(self, admin, topic_name):
        """
        Get partitions count for specified topic.
        *Example:*\n
                | Get Partitions Count | admin | kafka-topic-tests
        """
        info = self.get_topic_info(admin, topic_name)
        partitions = info[0]['partitions']
        return len(partitions)

    def get_replication_factor(self, admin, topic_name):
        """
        Get replication factor for specified topic.
        *Example:*\n
                | Get Replication Factor | admin | kafka-topic-tests
        """
        info = self.get_topic_info(admin, topic_name)
        replicas = info[0]['partitions'][0]['replicas']
        return len(replicas)

    def delete_topic(self, admin, topic_name):
        """
        Deletes topic with specific name.
        *Args:*\n
             _admin_ (AdminClient) - Kafka admin client;\n
             _topic_name_ (str) - name of topic;\n
        *Example:*\n
            | Delete Topic | admin | kafka-topic-tests |
        """
        topics = self.get_topics_list(admin)
        if topic_name not in topics:
            logger.debug(f'Topic "{topic_name}" has already been deleted.')
            return
        self.__delete_topics(admin, [topic_name])

    def delete_topic_by_pattern(self, admin, topic_name_pattern):
        """
        Deletes topics by pattern.
        *Args:*\n
             _admin_ (AdminClient) - Kafka admin client;\n
             _topic_name_pattern_ (str) - topic name pattern;\n
        *Example:*\n
            | Delete Topic By Pattern | admin | kafka-topic-* |
        """
        topics = self.get_topics_list(admin, topic_name_pattern)
        if not topics:
            logger.debug('Topics have already been deleted.')
            return
        self.__delete_topics(admin, topics)

    def __delete_topics(self, admin, topics):
        try:
            admin.delete_topics(topics)
            logger.debug(f'Topic "{topics}" is deleted.')
        except Exception as e:
            self.builtin.fail(f'Failed to delete topic "{topics}": {e}')

    def find_out_leader_broker(self, admin, brokers, topic_name):
        """
        Finds out name of leader broker by id pointed out in topic partitions setting.
        *Args:*\n
             _admin_ (AdminClient) - Kafka admin client;\n
             _brokers_ (dict) - dictionary of available broker name and its environment variables;\n
             _topic_name_ (str) - name of topic;\n
        *Example:*\n
            | Find Out Leader Broker | {'kafka-1': {'BROKER_ID': '1'}, 'kafka-2': {'BROKER_ID': '2'}, 'kafka-3': {'BROKER_ID': '3'}} | kafka-topic-tests |
        """
        topic_info = self.get_topic_info(admin, topic_name)
        leader_broker_id = topic_info[0]['partitions'][0]['leader']
        if leader_broker_id:
            for broker in brokers:
                broker_id = int(brokers.get(broker)['BROKER_ID'])
                if broker_id == leader_broker_id:
                    logger.debug(f'Resolved "{broker}" as leader.')
                    return broker
        return ''

    class TokenProvider(AbstractTokenProvider):

        def __init__(self, token_endpoint, client_id, client_secret):
            super().__init__()
            self._token_endpoint = token_endpoint
            self._client_id = client_id
            self._client_secret = client_secret

        def token(self):
            return self.retrieve_token(self._token_endpoint, self._client_id, self._client_secret)

        def retrieve_token(self, token_endpoint, client_id, client_secret):
            data = {'grant_type': 'client_credentials'}
            resp = requests.post(token_endpoint, auth=(client_id, client_secret), data=data)
            return resp.json()["access_token"]

    def create_kafka_oauth_producer(self, token_endpoint, client_id, client_secret):
        oauth_producer = None
        try:
            configs = dict(self._common_configs.items())
            configs['sasl_mechanism'] = 'OAUTHBEARER'
            configs['sasl_oauth_token_provider'] = self.TokenProvider(token_endpoint, client_id, client_secret)
            configs['retries'] = 10000000
            configs['max_in_flight_requests_per_connection'] = 1
            configs['request_timeout_ms'] = 900000
            configs['connections_max_idle_ms'] = 1000000
            oauth_producer = KafkaProducer(**configs)
            logger.debug("Kafka Oauth producer is created.")
        except Exception as e:
            self.builtin.fail('Exception while connecting Kafka: {}'.format(e))
        return oauth_producer

    def create_kafka_oauth_consumer(self, topic_name, token_endpoint, client_id, client_secret, group_id=None):
        oauth_consumer = None
        try:
            configs = dict(self._common_configs.items())
            configs['sasl_mechanism'] = 'OAUTHBEARER'
            configs['sasl_oauth_token_provider'] = self.TokenProvider(token_endpoint, client_id, client_secret)
            configs['auto_offset_reset'] = 'earliest'
            configs['enable_auto_commit'] = False
            configs['request_timeout_ms'] = 900000
            configs['connections_max_idle_ms'] = 1000000
            if group_id is not None:
                configs['group_id'] = group_id
            oauth_consumer = KafkaConsumer(topic_name, **configs)
            logger.debug("Kafka Oauth consumer is created.")
        except Exception as e:
            self.builtin.fail('Exception while connecting Kafka: {}'.format(e))
        return oauth_consumer

    def __resolve_resource_type(self, resource_type):
        if resource_type == 'topic':
            return ResourceType.TOPIC
        elif resource_type == 'group':
            return ResourceType.GROUP
        else:
            return ResourceType.UNKNOWN

    def create_acl(self, admin, resource_name, operation='ALL', permission='ALLOW', principal='Role:kafka', host='*',
                   resource_type='topic'):
        operation = ACLOperation[operation]
        permission = ACLPermissionType[permission]
        resource_type = self.__resolve_resource_type(resource_type)
        acl = ACL(
            principal=principal,
            host=host,
            operation=operation,
            permission_type=permission,
            resource_pattern=ResourcePattern(resource_type, resource_name)
        )
        try:
            admin.create_acls([acl])
            logger.debug(f'ACL {[acl]} was successfully created.')
        except Exception as e:
            self.builtin.fail('Failed to create ACL')

    def get_acls(self, admin, topic_name, host='*', resource_type='topic'):
        resource_type = self.__resolve_resource_type(resource_type)
        acl_filter = ACLFilter(
            principal=None,
            host=host,
            operation=ACLOperation.ANY,
            permission_type=ACLPermissionType.ANY,
            resource_pattern=ResourcePattern(resource_type, topic_name)
        )
        existing_acls = admin.describe_acls(acl_filter)
        return existing_acls

    def delete_acl(self, admin, topic_name, operation='ANY', permission='ANY', principal='Role:kafka',
                   resource_type='topic'):
        operation = ACLOperation[operation]
        permission = ACLPermissionType[permission]
        resource_type = self.__resolve_resource_type(resource_type)
        acl_filter = ACLFilter(
            principal=principal,
            host=None,
            operation=operation,
            permission_type=permission,
            resource_pattern=ResourcePattern(resource_type, topic_name))
        try:
            admin.delete_acls([acl_filter])
            logger.debug('ACL was deleted')
        except Exception as e:
            self.builtin.fail('Failed to delete ACL')

    def get_topic_configs(self, admin, topic):
        return admin.describe_configs(config_resources=[ConfigResource(ConfigResourceType.TOPIC, topic)])

    def get_value_of_topic_config(self, admin, topic, config_name=None):
        topic_configs = self.get_topic_configs(admin, topic)
        config_list = topic_configs[0].resources[0][4]
        for conf in config_list:
            if config_name in conf:
                return conf[1]
