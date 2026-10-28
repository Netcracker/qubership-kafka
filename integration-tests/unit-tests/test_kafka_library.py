# Copyright 2024-2026 NetCracker Technology Corporation
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
import sys
import unittest
from unittest import mock

from kafka.protocol.admin import DescribeConfigsResponse_v2

sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'robot', 'tests', 'shared', 'lib'))

import KafkaLibrary  # noqa: E402


def config_entry(name, value):
    return name, value, False, 4, False, []


def describe_response(broker_id, entries, error_code=0, error_message=None):
    return DescribeConfigsResponse_v2(
        throttle_time_ms=0,
        resources=[(error_code, error_message, 4, str(broker_id), entries)])


class KafkaLibraryTest(unittest.TestCase):

    def setUp(self):
        self.library = KafkaLibrary.KafkaLibrary('kafka:9092', 'ns', 'kafka', '9092', '', '', 'false')
        self.library.builtin = mock.Mock()
        self.library.builtin.fail.side_effect = AssertionError

    def mock_admin(self, responses_by_broker):
        admin = mock.Mock()
        admin.describe_cluster.return_value = {
            'brokers': [{'node_id': broker_id} for broker_id in responses_by_broker]}
        admin.describe_configs.side_effect = lambda config_resources: [
            responses_by_broker[int(resource.name)] for resource in config_resources]
        patcher = mock.patch.object(self.library, '_KafkaLibrary__build_admin_client', return_value=admin)
        patcher.start()
        self.addCleanup(patcher.stop)
        return admin


class NormalizeEnvironmentVariablesTest(KafkaLibraryTest):

    def test_empty_variables(self):
        self.assertEqual({}, self.library.normalize_environment_variables(None))
        self.assertEqual({}, self.library.normalize_environment_variables([]))

    def test_only_conf_kafka_variables_are_converted(self):
        variables = [
            'CONF_KAFKA_AUTO_CREATE_TOPICS_ENABLE=false',
            ' CONF_KAFKA_SASL_JAAS_CONFIG = a=b ',
            'HEAP_OPTS=-Xmx1g',
            'CONF_KAFKA_INVALID',
        ]
        self.assertEqual({'auto.create.topics.enable': 'false', 'sasl.jaas.config': 'a=b'},
                         self.library.normalize_environment_variables(variables))

    def test_last_variable_wins(self):
        variables = ['CONF_KAFKA_NUM_IO_THREADS=8', 'CONF_KAFKA_NUM_IO_THREADS=9']
        self.assertEqual({'num.io.threads': '9'}, self.library.normalize_environment_variables(variables))


class GetBrokerPropertiesTest(KafkaLibraryTest):

    def test_properties_of_every_broker_are_returned(self):
        self.mock_admin({
            1: describe_response(1, [config_entry('num.io.threads', '8'), config_entry('custom.secret', None)]),
            2: describe_response(2, [config_entry('num.io.threads', '16')]),
        })
        self.assertEqual({
            '1': {'num.io.threads': '8', 'custom.secret': None},
            '2': {'num.io.threads': '16'},
        }, self.library.get_broker_properties())

    def test_describe_error_fails(self):
        self.mock_admin({1: describe_response(1, [], error_code=29, error_message='denied')})
        with self.assertRaises(AssertionError):
            self.library.get_broker_properties()
        self.library.builtin.fail.assert_called_once_with('Failed to describe configs of broker 1: denied')


if __name__ == '__main__':
    unittest.main()
