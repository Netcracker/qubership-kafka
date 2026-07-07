#!/usr/bin/python

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

import argparse
import ast
import json
import logging
import os
import re
import sys

from kafka.admin import KafkaAdminClient, ConfigResource, ConfigResourceType, ACLFilter, ACLPermissionType, \
    ACLOperation, ResourcePatternFilter, ResourceType, ACLResourcePatternType

REQUEST_HEADERS = {
    'Accept': 'application/json',
    'Content-type': 'application/json',
}
CA_CERT_PATH = '/tls/ca.crt'
TLS_CERT_PATH = '/tls/tls.crt'
TLS_KEY_PATH = '/tls/tls.key'
DEFAULT_SECRETS_DIR = '/etc/secrets/backup-daemon-pod-secrets'
SECRETS_DIR = os.getenv("BACKUP_DAEMON_SECRETS_DIR", DEFAULT_SECRETS_DIR)

loggingLevel = logging.DEBUG if os.getenv(
    'KAFKA_BACKUP_DAEMON_DEBUG') else logging.INFO
logging.basicConfig(level=loggingLevel,
                    format='[%(asctime)s,%(msecs)03d][%(levelname)s][category=Backup] %(message)s',
                    datefmt='%Y-%m-%dT%H:%M:%S')

def get_env_or_file(name: str):
    default_file_path = os.path.join(SECRETS_DIR, name)
    if os.path.isfile(default_file_path):
        with open(default_file_path, "r", encoding="utf-8") as secret_file:
            return secret_file.read().strip()
    return os.getenv(name)


class Backup:

    def __init__(self, folder, mode=None):
        kafka_bootstrap_servers = os.getenv("KAFKA_BOOTSTRAP_SERVERS")
        if not kafka_bootstrap_servers:
            logging.error("Kafka bootstrap servers are not specified.")
            sys.exit(1)
        kafka_enable_ssl = self.str2bool(os.getenv("KAFKA_ENABLE_SSL", "false"))
        kafka_sasl_mechanism = os.getenv('KAFKA_SASL_MECHANISM', 'SCRAM-SHA-512')
        kafka_username = get_env_or_file("KAFKA_USERNAME")
        kafka_password = get_env_or_file("KAFKA_PASSWORD")

        security_protocol = 'SSL' if kafka_enable_ssl else 'PLAINTEXT'
        ssl_cafile = CA_CERT_PATH if kafka_enable_ssl and os.path.exists(CA_CERT_PATH) else None
        ssl_certfile = TLS_CERT_PATH if kafka_enable_ssl and os.path.exists(TLS_CERT_PATH) else None
        ssl_keyfile = TLS_KEY_PATH if kafka_enable_ssl and os.path.exists(TLS_KEY_PATH) else None
        configs = {
            'bootstrap_servers': kafka_bootstrap_servers
        }
        if kafka_username and kafka_password:
            configs['security_protocol'] = f'SASL_{security_protocol}'
            configs['sasl_mechanism'] = kafka_sasl_mechanism
            configs['sasl_plain_username'] = kafka_username
            configs['sasl_plain_password'] = kafka_password
        else:
            configs['security_protocol'] = security_protocol
        configs['ssl_cafile'] = ssl_cafile
        configs['ssl_certfile'] = ssl_certfile
        configs['ssl_keyfile'] = ssl_keyfile
        self._client = KafkaAdminClient(**configs)
        self._folder = folder
        self._mode = mode

    @staticmethod
    def str2bool(v: str):
        return v.lower() in ("yes", "true", "t", "1")

    def backup(self, topics=None, topic_regex=None):

        logging.info(f'Start topics backup')

        folder = f'{self._folder}'
        if not os.path.exists(folder):
            os.makedirs(folder)
        snapshot = {'topics': [], 'acls': []}
        cli = self._client

        if topics is None:
            topics = cli.list_topics()
            if topic_regex:
                pattern = re.compile(topic_regex)
                topics = [s for s in topics if pattern.match(s)]

        for topic in topics:
            configs = cli.describe_configs(config_resources=[ConfigResource(ConfigResourceType.TOPIC, topic)])
            config_list = configs[0].resources[0][4]
            if len(config_list) == 0:
                logging.error(f"Specified Kafka topic '{topic}' not found.")
                sys.exit(1)
            conf = {}
            for c in config_list:
                conf[c[0]] = c[1]
            topic_config = cli.describe_topics([topic])[0]
            num = len(topic_config['partitions'])
            rf = len(topic_config['partitions'][0]['replicas'])
            config = {'name': topic, 'num_partitions': num, 'replication_factor': rf, 'topic_configs': conf}
            snapshot['topics'].append(config)

        if self._mode and self._mode == 'acl':
            logging.info('Start ACL backup.')
            pattern = ResourcePatternFilter(resource_type=ResourceType.ANY,
                                            resource_name=None,
                                            pattern_type=ACLResourcePatternType.ANY)
            acl_filter = ACLFilter(principal=None,
                                   host=None,
                                   operation=ACLOperation.ANY,
                                   permission_type=ACLPermissionType.ANY,
                                   resource_pattern=pattern)
            acls = cli.describe_acls(acl_filter)[0]
            for acl in acls:
                a = {'host': acl.host,
                     'operation': acl.operation,
                     'permission_type': acl.permission_type,
                     'principal': acl.principal,
                     'resource_pattern': {'resource_name': acl.resource_pattern.resource_name,
                                          'resource_type': acl.resource_pattern.resource_type}}
                snapshot['acls'].append(a)

        with open(folder + '/snapshot.json', 'w', encoding='utf-8') as f:
            json.dump(snapshot, f, ensure_ascii=False, indent=4)

        logging.info('Snapshot completed successfully.')


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('folder')
    parser.add_argument('-d', '--topics')
    parser.add_argument('-topic_regex')
    parser.add_argument('-mode')
    args = parser.parse_args()

    backup_instance = Backup(args.folder, args.mode)
    backup_instance.backup(ast.literal_eval(args.topics) if args.topics else None, args.topic_regex)
