# Copyright 2024-2025 NetCracker Technology Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import random
import string
import unittest
from unittest import mock

import kafka_metric
from kafka_metric import \
    _collect_metrics, \
    _check_config_consistency


def partition(partition_number: int, leader: int, replicas: list, isr: list) -> dict:
    return {'partition': partition_number, 'leader': leader, 'replicas': replicas, 'isr': isr}


def topic_metadata(name: str, partitions: list) -> dict:
    return {'topic': name, 'partitions': partitions}


def topic_config(name: str, value: str) -> dict:
    return {'config_names': name, 'config_value': value}


def topic_configs(name: str, unclean_leader_election: bool = False) -> dict:
    config_entries = [topic_config('min.insync.replicas', '2'),
                      topic_config('cleanup.policy', 'compact'),
                      topic_config('retention.ms', '604800000'),
                      topic_config('unclean.leader.election.enable', str(unclean_leader_election).lower())]
    return {'resource_name': name, 'config_entries': config_entries}


def prepare_topics():
    ok_topic_partitions = [partition(0, 2, [1, 2, 3], [1, 2, 3]),
                           partition(1, 1, [1, 2, 3], [1, 2, 3])]
    ok_topic = topic_metadata('ok_topic', ok_topic_partitions)

    topic_without_leader_partitions = [partition(0, 1, [1, 2, 3], [1, 2, 3]),
                                       partition(1, -1, [1, 2, 3], [1, 2, 3])]
    topic_without_leader = topic_metadata('topic_without_leader', topic_without_leader_partitions)

    topic_under_replicated_partitions = [partition(0, 1, [1, 2, 3], [1, 2, 3]),
                                         partition(2, 2, [1, 2, 3], [1, 2])]
    topic_under_replicated = topic_metadata('topic_under_replicated', topic_under_replicated_partitions)

    topic_without_leader_and_under_replicated_partitions = [partition(0, -1, [1, 2, 3], [1, 2])]
    topic_without_leader_and_under_replicated = topic_metadata('topic_without_leader_and_under_replicated',
                                                               topic_without_leader_and_under_replicated_partitions)

    topics = [ok_topic,
              topic_without_leader,
              topic_under_replicated,
              topic_without_leader_and_under_replicated]
    return topics


def prepare_topic_configs(topics_metadata: list) -> list:
    configs = []
    unclean_leader_election_enabled = True
    for topic in topics_metadata:
        configs.append(topic_configs(topic.get('topic'), unclean_leader_election_enabled))
        unclean_leader_election_enabled = not unclean_leader_election_enabled
    return configs


def generate_list_of_random_topic_names(topics_count) -> list:
    topics = list()
    for i in range(topics_count):
        topic_name_len = random.randint(10, 20)
        topic_name = ''.join(random.choices(string.ascii_uppercase + string.digits, k=topic_name_len))
        topics.append(topic_name)
    return topics


class TestKafkaMetric(unittest.TestCase):

    def test_get_kafka_hosts(self):
        kafka_metric.KAFKA_SERVICE_NAME = "kafka"
        kafka_metric.KAFKA_TOTAL_BROKERS_COUNT = 3
        kafka_metric.OS_PROJECT = "kafka-cluster"
        result = list()
        for broker_id in range(kafka_metric.KAFKA_TOTAL_BROKERS_COUNT):
            kafka_host = f'{kafka_metric.KAFKA_SERVICE_NAME}-{broker_id + 1}.{kafka_metric.KAFKA_SERVICE_NAME}-broker.{kafka_metric.OS_PROJECT}'
            result.append(kafka_host)
        self.assertEqual(['kafka-1.kafka-broker.kafka-cluster', 'kafka-2.kafka-broker.kafka-cluster',
                          'kafka-3.kafka-broker.kafka-cluster'], result)


    def test_check_config_consistency_when_configs_the_same(self):
        broker_configs = {'log.cleaner.min.compaction.lag.ms': '0', 'offsets.topic.num.partitions': '50',
                          'log.flush.interval.messages': '9223372036854775807', 'controller.socket.timeout.ms': '30000',
                          'log.flush.interval.ms': None, 'principal.builder.class': None, 'min.insync.replicas': '1',
                          'num.recovery.threads.per.data.dir': '1', 'ssl.keystore.type': 'JKS',
                          'sasl.mechanism.inter.broker.protocol': 'SCRAM-SHA-512',
                          'fetch.purgatory.purge.interval.requests': '1000',
                          'ssl.endpoint.identification.algorithm': 'https', 'replica.socket.timeout.ms': '30000',
                          'message.max.bytes': '1000012', 'connections.max.reauth.ms': '0',
                          'log.flush.offset.checkpoint.interval.ms': '60000', 'quota.window.num': '11',
                          'zookeeper.connect': 'zookeeper:2181', 'authorizer.class.name': '',
                          'password.encoder.secret': None,
                          'listener.name.external.oauthbearer.sasl.server.callback.handler.class': None,
                          'num.replica.fetchers': '1', 'alter.log.dirs.replication.quota.window.size.seconds': '1',
                          'log.roll.jitter.hours': '0', 'password.encoder.old.secret': None,
                          'log.cleaner.delete.retention.ms': '86400000', 'queued.max.requests': '500',
                          'log.cleaner.threads': '1', 'sasl.kerberos.service.name': None,
                          'socket.request.max.bytes': '104857600', 'log.message.timestamp.type': 'CreateTime',
                          'zookeeper.set.acl': 'false', 'connections.max.idle.ms': '600000',
                          'max.connections': '2147483647', 'delegation.token.expiry.time.ms': '86400000',
                          'transaction.state.log.num.partitions': '50',
                          'listener.security.protocol.map': 'INTERNAL:SASL_PLAINTEXT,INTER_BROKER:SASL_PLAINTEXT,EXTERNAL:SASL_PLAINTEXT',
                          'log.retention.hours': '168', 'client.quota.callback.class': None, 'ssl.provider': None,
                          'delete.records.purgatory.purge.interval.requests': '1', 'log.roll.ms': None,
                          'ssl.cipher.suites': '', 'security.inter.broker.protocol': 'PLAINTEXT',
                          'replica.high.watermark.checkpoint.interval.ms': '5000',
                          'replication.quota.window.size.seconds': '1',
                          'sasl.kerberos.ticket.renew.window.factor': '0.8', 'zookeeper.connection.timeout.ms': '18000',
                          'metrics.recording.level': 'INFO',
                          'password.encoder.cipher.algorithm': 'AES/CBC/PKCS5Padding',
                          'listener.name.external.oauthbearer.connections.max.reauth.ms': None,
                          'listener.name.internal.oauthbearer.sasl.login.callback.handler.class': None,
                          'ssl.principal.mapping.rules': 'DEFAULT', 'replica.selector.class': None,
                          'max.connections.per.ip': '2147483647', 'background.threads': '10',
                          'quota.consumer.default': '9223372036854775807', 'request.timeout.ms': '30000',
                          'log.message.format.version': '2.7-IV1', 'sasl.login.class': None,
                          'log.dir': '/tmp/kafka-logs', 'log.segment.bytes': '1073741824',
                          'listener.name.external.oauthbearer.sasl.jaas.config': None,
                          'replica.fetch.response.max.bytes': '10485760', 'group.max.session.timeout.ms': '1800000',
                          'port': '9092', 'log.retention.minutes': None, 'log.segment.delete.delay.ms': '60000',
                          'controlled.shutdown.enable': 'true',
                          'log.message.timestamp.difference.max.ms': '9223372036854775807',
                          'sasl.login.refresh.min.period.seconds': '60', 'password.encoder.key.length': '128',
                          'transaction.abort.timed.out.transaction.cleanup.interval.ms': '60000',
                          'sasl.kerberos.kinit.cmd': '/usr/bin/kinit',
                          'log.cleaner.io.max.bytes.per.second': '1.7976931348623157E308',
                          'auto.leader.rebalance.enable': 'true', 'leader.imbalance.check.interval.seconds': '300',
                          'log.cleaner.min.cleanable.ratio': '0.5', 'replica.lag.time.max.ms': '30000',
                          'num.network.threads': '3', 'sasl.client.callback.handler.class': None,
                          'metrics.num.samples': '2', 'socket.send.buffer.bytes': '102400',
                          'password.encoder.keyfactory.algorithm': None, 'socket.receive.buffer.bytes': '102400',
                          'replica.fetch.min.bytes': '1', 'broker.rack': None,
                          'unclean.leader.election.enable': 'false', 'offsets.retention.check.interval.ms': '600000',
                          'producer.purgatory.purge.interval.requests': '1000', 'metrics.sample.window.ms': '30000',
                          'log.retention.check.interval.ms': '300000', 'leader.imbalance.per.broker.percentage': '10',
                          'sasl.login.refresh.window.jitter': '0.05', 'advertised.host.name': None,
                          'metric.reporters': '', 'quota.producer.default': '9223372036854775807',
                          'auto.create.topics.enable': 'true', 'replica.socket.receive.buffer.bytes': '65536',
                          'replica.fetch.wait.max.ms': '500', 'password.encoder.iterations': '4096',
                          'listener.name.external.scram-sha-512.sasl.jaas.config': None,
                          'default.replication.factor': '1', 'ssl.truststore.password': None,
                          'log.preallocate': 'false', 'sasl.kerberos.principal.to.local.rules': 'DEFAULT',
                          'transactional.id.expiration.ms': '604800000',
                          'transaction.state.log.replication.factor': '1', 'control.plane.listener.name': None,
                          'num.io.threads': '8', 'sasl.login.refresh.buffer.seconds': '300',
                          'connection.failed.authentication.delay.ms': '100', 'offsets.commit.required.acks': '-1',
                          'delete.topic.enable': 'true', 'quota.window.size.seconds': '1', 'ssl.truststore.type': 'JKS',
                          'offsets.commit.timeout.ms': '5000',
                          'log.cleaner.max.compaction.lag.ms': '9223372036854775807', 'log.retention.ms': None,
                          'alter.log.dirs.replication.quota.window.num': '11', 'log.cleaner.enable': 'true',
                          'offsets.load.buffer.size': '5242880', 'ssl.client.auth': 'none',
                          'controlled.shutdown.max.retries': '3', 'offsets.topic.replication.factor': '1',
                          'transaction.state.log.min.isr': '1', 'ssl.secure.random.implementation': None,
                          'sasl.kerberos.ticket.renew.jitter': '0.05', 'ssl.trustmanager.algorithm': 'PKIX',
                          'zookeeper.session.timeout.ms': '18000',
                          'listener.name.internal.oauthbearer.sasl.jaas.config': None, 'log.retention.bytes': '-1',
                          'sasl.jaas.config': None, 'sasl.kerberos.min.time.before.relogin': '60000',
                          'offsets.retention.minutes': '10080', 'replica.fetch.backoff.ms': '1000',
                          'inter.broker.protocol.version': '2.7-IV1', 'kafka.metrics.reporters': '',
                          'num.partitions': '1',
                          'listeners': 'INTERNAL://0.0.0.0:9092,INTER_BROKER://0.0.0.0:9093,EXTERNAL://0.0.0.0:9094',
                          'broker.id.generation.enable': 'true', 'ssl.enabled.protocols': 'TLSv1.2,TLSv1.1,TLSv1',
                          'inter.broker.listener.name': 'INTER_BROKER', 'alter.config.policy.class.name': None,
                          'delegation.token.expiry.check.interval.ms': '3600000',
                          'zookeeper.max.in.flight.requests': '10',
                          'log.flush.scheduler.interval.ms': '9223372036854775807',
                          'log.index.size.max.bytes': '10485760',
                          'listener.name.internal.scram-sha-512.sasl.jaas.config': None,
                          'ssl.keymanager.algorithm': 'SunX509', 'sasl.login.callback.handler.class': None,
                          'replica.fetch.max.bytes': '1048576', 'sasl.server.callback.handler.class': None,
                          'listener.name.external.oauthbearer.sasl.login.callback.handler.class': None,
                          'advertised.port': None, 'log.cleaner.dedupe.buffer.size': '134217728',
                          'log.cleaner.io.buffer.size': '524288', 'create.topic.policy.class.name': None,
                          'controlled.shutdown.retry.backoff.ms': '5000', 'security.providers': None,
                          'log.roll.hours': '168', 'log.cleanup.policy': 'delete',
                          'log.flush.start.offset.checkpoint.interval.ms': '60000', 'host.name': '',
                          'log.roll.jitter.ms': None, 'transaction.state.log.segment.bytes': '104857600',
                          'offsets.topic.segment.bytes': '104857600', 'group.initial.rebalance.delay.ms': '3000',
                          'log.index.interval.bytes': '4096', 'log.cleaner.backoff.ms': '15000',
                          'offset.metadata.max.bytes': '4096', 'ssl.truststore.location': None,
                          'ssl.keystore.password': None, 'zookeeper.sync.time.ms': '2000',
                          'compression.type': 'producer', 'max.connections.per.ip.overrides': '',
                          'sasl.login.refresh.window.factor': '0.8', 'kafka.metrics.polling.interval.secs': '10',
                          'listener.name.internal.oauthbearer.sasl.server.callback.handler.class': None,
                          'max.incremental.fetch.session.cache.slots': '1000', 'delegation.token.master.key': None,
                          'ssl.key.password': None, 'reserved.broker.max.id': '1000',
                          'listener.name.internal.oauthbearer.connections.max.reauth.ms': None,
                          'transaction.remove.expired.transaction.cleanup.interval.ms': '3600000',
                          'log.message.downconversion.enable': 'true', 'ssl.protocol': 'TLS',
                          'transaction.state.log.load.buffer.size': '5242880', 'ssl.keystore.location': None,
                          'sasl.enabled.mechanisms': 'SCRAM-SHA-512,OAUTHBEARER',
                          'num.replica.alter.log.dirs.threads': None, 'group.min.session.timeout.ms': '6000',
                          'log.cleaner.io.buffer.load.factor': '0.9', 'transaction.max.timeout.ms': '900000',
                          'group.max.size': '2147483647', 'offsets.topic.compression.codec': '0',
                          'delegation.token.max.lifetime.ms': '604800000', 'replication.quota.window.num': '11',
                          'queued.max.request.bytes': '-1'}
        brokers_configs_list = [broker_configs, dict(broker_configs), dict(broker_configs)]
        result = _check_config_consistency(brokers_configs_list)
        self.assertEqual('Yes', result, 'Configs should be the same')

    def test_check_config_consistency_when_configs_differ(self):
        broker_configs = {'log.cleaner.min.compaction.lag.ms': '0', 'offsets.topic.num.partitions': '50',
                          'log.flush.interval.messages': '9223372036854775807', 'controller.socket.timeout.ms': '30000',
                          'log.flush.interval.ms': None, 'principal.builder.class': None, 'min.insync.replicas': '1',
                          'num.recovery.threads.per.data.dir': '1', 'ssl.keystore.type': 'JKS',
                          'sasl.mechanism.inter.broker.protocol': 'SCRAM-SHA-512',
                          'fetch.purgatory.purge.interval.requests': '1000',
                          'ssl.endpoint.identification.algorithm': 'https', 'replica.socket.timeout.ms': '30000',
                          'message.max.bytes': '1000012', 'connections.max.reauth.ms': '0',
                          'log.flush.offset.checkpoint.interval.ms': '60000', 'quota.window.num': '11',
                          'zookeeper.connect': 'zookeeper:2181', 'authorizer.class.name': '',
                          'password.encoder.secret': None,
                          'listener.name.external.oauthbearer.sasl.server.callback.handler.class': None,
                          'num.replica.fetchers': '1', 'alter.log.dirs.replication.quota.window.size.seconds': '1',
                          'log.roll.jitter.hours': '0', 'password.encoder.old.secret': None,
                          'log.cleaner.delete.retention.ms': '86400000', 'queued.max.requests': '500',
                          'log.cleaner.threads': '1', 'sasl.kerberos.service.name': None,
                          'socket.request.max.bytes': '104857600', 'log.message.timestamp.type': 'CreateTime',
                          'zookeeper.set.acl': 'false', 'connections.max.idle.ms': '600000',
                          'max.connections': '2147483647', 'delegation.token.expiry.time.ms': '86400000',
                          'transaction.state.log.num.partitions': '50',
                          'listener.security.protocol.map': 'INTERNAL:SASL_PLAINTEXT,INTER_BROKER:SASL_PLAINTEXT,EXTERNAL:SASL_PLAINTEXT',
                          'log.retention.hours': '168', 'client.quota.callback.class': None, 'ssl.provider': None,
                          'delete.records.purgatory.purge.interval.requests': '1', 'log.roll.ms': None,
                          'ssl.cipher.suites': '', 'security.inter.broker.protocol': 'PLAINTEXT',
                          'replica.high.watermark.checkpoint.interval.ms': '5000',
                          'replication.quota.window.size.seconds': '1',
                          'sasl.kerberos.ticket.renew.window.factor': '0.8', 'zookeeper.connection.timeout.ms': '18000',
                          'metrics.recording.level': 'INFO',
                          'password.encoder.cipher.algorithm': 'AES/CBC/PKCS5Padding',
                          'listener.name.external.oauthbearer.connections.max.reauth.ms': None,
                          'listener.name.internal.oauthbearer.sasl.login.callback.handler.class': None,
                          'ssl.principal.mapping.rules': 'DEFAULT', 'replica.selector.class': None,
                          'max.connections.per.ip': '2147483647', 'background.threads': '10',
                          'quota.consumer.default': '9223372036854775807', 'request.timeout.ms': '30000',
                          'log.message.format.version': '2.7-IV1', 'sasl.login.class': None,
                          'log.dir': '/tmp/kafka-logs', 'log.segment.bytes': '1073741824',
                          'listener.name.external.oauthbearer.sasl.jaas.config': None,
                          'replica.fetch.response.max.bytes': '10485760', 'group.max.session.timeout.ms': '1800000',
                          'port': '9092', 'log.retention.minutes': None, 'log.segment.delete.delay.ms': '60000',
                          'controlled.shutdown.enable': 'true',
                          'log.message.timestamp.difference.max.ms': '9223372036854775807',
                          'sasl.login.refresh.min.period.seconds': '60', 'password.encoder.key.length': '128',
                          'transaction.abort.timed.out.transaction.cleanup.interval.ms': '60000',
                          'sasl.kerberos.kinit.cmd': '/usr/bin/kinit',
                          'log.cleaner.io.max.bytes.per.second': '1.7976931348623157E308',
                          'auto.leader.rebalance.enable': 'true', 'leader.imbalance.check.interval.seconds': '300',
                          'log.cleaner.min.cleanable.ratio': '0.5', 'replica.lag.time.max.ms': '30000',
                          'num.network.threads': '3', 'sasl.client.callback.handler.class': None,
                          'metrics.num.samples': '2', 'socket.send.buffer.bytes': '102400',
                          'password.encoder.keyfactory.algorithm': None, 'socket.receive.buffer.bytes': '102400',
                          'replica.fetch.min.bytes': '1', 'broker.rack': None,
                          'unclean.leader.election.enable': 'false', 'offsets.retention.check.interval.ms': '600000',
                          'producer.purgatory.purge.interval.requests': '1000', 'metrics.sample.window.ms': '30000',
                          'log.retention.check.interval.ms': '300000', 'leader.imbalance.per.broker.percentage': '10',
                          'sasl.login.refresh.window.jitter': '0.05', 'advertised.host.name': None,
                          'metric.reporters': '', 'quota.producer.default': '9223372036854775807',
                          'auto.create.topics.enable': 'true', 'replica.socket.receive.buffer.bytes': '65536',
                          'replica.fetch.wait.max.ms': '500', 'password.encoder.iterations': '4096',
                          'listener.name.external.scram-sha-512.sasl.jaas.config': None,
                          'default.replication.factor': '1', 'ssl.truststore.password': None,
                          'log.preallocate': 'false', 'sasl.kerberos.principal.to.local.rules': 'DEFAULT',
                          'transactional.id.expiration.ms': '604800000',
                          'transaction.state.log.replication.factor': '1', 'control.plane.listener.name': None,
                          'num.io.threads': '8', 'sasl.login.refresh.buffer.seconds': '300',
                          'connection.failed.authentication.delay.ms': '100', 'offsets.commit.required.acks': '-1',
                          'delete.topic.enable': 'true', 'quota.window.size.seconds': '1', 'ssl.truststore.type': 'JKS',
                          'offsets.commit.timeout.ms': '5000',
                          'log.cleaner.max.compaction.lag.ms': '9223372036854775807', 'log.retention.ms': None,
                          'alter.log.dirs.replication.quota.window.num': '11', 'log.cleaner.enable': 'true',
                          'offsets.load.buffer.size': '5242880', 'ssl.client.auth': 'none',
                          'controlled.shutdown.max.retries': '3', 'offsets.topic.replication.factor': '1',
                          'transaction.state.log.min.isr': '1', 'ssl.secure.random.implementation': None,
                          'sasl.kerberos.ticket.renew.jitter': '0.05', 'ssl.trustmanager.algorithm': 'PKIX',
                          'zookeeper.session.timeout.ms': '18000',
                          'listener.name.internal.oauthbearer.sasl.jaas.config': None, 'log.retention.bytes': '-1',
                          'sasl.jaas.config': None, 'sasl.kerberos.min.time.before.relogin': '60000',
                          'offsets.retention.minutes': '10080', 'replica.fetch.backoff.ms': '1000',
                          'inter.broker.protocol.version': '2.7-IV1', 'kafka.metrics.reporters': '',
                          'num.partitions': '1',
                          'listeners': 'INTERNAL://0.0.0.0:9092,INTER_BROKER://0.0.0.0:9093,EXTERNAL://0.0.0.0:9094',
                          'broker.id.generation.enable': 'true', 'ssl.enabled.protocols': 'TLSv1.2,TLSv1.1,TLSv1',
                          'inter.broker.listener.name': 'INTER_BROKER', 'alter.config.policy.class.name': None,
                          'delegation.token.expiry.check.interval.ms': '3600000',
                          'zookeeper.max.in.flight.requests': '10',
                          'log.flush.scheduler.interval.ms': '9223372036854775807',
                          'log.index.size.max.bytes': '10485760',
                          'listener.name.internal.scram-sha-512.sasl.jaas.config': None,
                          'ssl.keymanager.algorithm': 'SunX509', 'sasl.login.callback.handler.class': None,
                          'replica.fetch.max.bytes': '1048576', 'sasl.server.callback.handler.class': None,
                          'listener.name.external.oauthbearer.sasl.login.callback.handler.class': None,
                          'advertised.port': None, 'log.cleaner.dedupe.buffer.size': '134217728',
                          'log.cleaner.io.buffer.size': '524288', 'create.topic.policy.class.name': None,
                          'controlled.shutdown.retry.backoff.ms': '5000', 'security.providers': None,
                          'log.roll.hours': '168', 'log.cleanup.policy': 'delete',
                          'log.flush.start.offset.checkpoint.interval.ms': '60000', 'host.name': '',
                          'log.roll.jitter.ms': None, 'transaction.state.log.segment.bytes': '104857600',
                          'offsets.topic.segment.bytes': '104857600', 'group.initial.rebalance.delay.ms': '3000',
                          'log.index.interval.bytes': '4096', 'log.cleaner.backoff.ms': '15000',
                          'offset.metadata.max.bytes': '4096', 'ssl.truststore.location': None,
                          'ssl.keystore.password': None, 'zookeeper.sync.time.ms': '2000',
                          'compression.type': 'producer', 'max.connections.per.ip.overrides': '',
                          'sasl.login.refresh.window.factor': '0.8', 'kafka.metrics.polling.interval.secs': '10',
                          'listener.name.internal.oauthbearer.sasl.server.callback.handler.class': None,
                          'max.incremental.fetch.session.cache.slots': '1000', 'delegation.token.master.key': None,
                          'ssl.key.password': None, 'reserved.broker.max.id': '1000',
                          'listener.name.internal.oauthbearer.connections.max.reauth.ms': None,
                          'transaction.remove.expired.transaction.cleanup.interval.ms': '3600000',
                          'log.message.downconversion.enable': 'true', 'ssl.protocol': 'TLS',
                          'transaction.state.log.load.buffer.size': '5242880', 'ssl.keystore.location': None,
                          'sasl.enabled.mechanisms': 'SCRAM-SHA-512,OAUTHBEARER',
                          'num.replica.alter.log.dirs.threads': None, 'group.min.session.timeout.ms': '6000',
                          'log.cleaner.io.buffer.load.factor': '0.9', 'transaction.max.timeout.ms': '900000',
                          'group.max.size': '2147483647', 'offsets.topic.compression.codec': '0',
                          'delegation.token.max.lifetime.ms': '604800000', 'replication.quota.window.num': '11',
                          'queued.max.request.bytes': '-1'}
        different_configs = dict(broker_configs)
        different_configs['auto.create.topics.enable'] = 'false'
        different_configs['listeners'] = 'INTERNAL://0.0.0.0:9092,INTER_BROKER://0.0.0.0:9093'
        brokers_configs_list = [different_configs, broker_configs, different_configs]
        result = _check_config_consistency(brokers_configs_list)
        self.assertEqual('No: [auto.create.topics.enable]: false VS true', result,
                         'First inconsistent config should be shown')


if __name__ == '__main__':
    unittest.main()
