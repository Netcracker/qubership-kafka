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

from kafka import KafkaAdminClient
from kafka.admin import NewTopic, ACL, ResourceType, ACLOperation, ACLPermissionType, ResourcePattern

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
                    format='[%(asctime)s,%(msecs)03d][%(levelname)s][category=Restore] %(message)s',
                    datefmt='%Y-%m-%dT%H:%M:%S')

def get_env_or_file(name: str):
    default_file_path = os.path.join(SECRETS_DIR, name)
    if os.path.isfile(default_file_path):
        with open(default_file_path, "r", encoding="utf-8") as secret_file:
            return secret_file.read().strip()
    return os.getenv(name)


class Restore:

    def __init__(self, folder, mode):
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

    def restore(self, topics=None, topic_regex=None):

        logging.info('Perform restore of snapshot')

        with open(self._folder + '/snapshot.json') as f:
            config = json.load(f)

        cli = self._client
        topics_restore = []

        if not topics:
            topics = []
            existing_topics = cli.list_topics()
            for topic in config['topics']:
                if topic['name'] in existing_topics:
                    logging.debug("Topic '%s' already exists in Kafka, skip it.", topic['name'])
                else:
                    topics.append(topic['name'])
            if topic_regex:
                pattern = re.compile(topic_regex)
                topics = [s for s in topics if pattern.match(s)]

        for topic in config['topics']:
            if topic['name'] in topics:
                topics_restore.append(NewTopic(topic['name'],
                                               topic['num_partitions'],
                                               topic['replication_factor'],
                                               topic_configs=topic['topic_configs']))

        cli.create_topics(topics_restore, validate_only=True)
        cli.create_topics(topics_restore)

        if self._mode and self._mode == 'acl':
            logging.info(f'Perform restore of ACL')
            acl_list = []
            for acl in config['acls']:
                resource_pattern = ResourcePattern(ResourceType(acl['resource_pattern']['resource_type']),
                                                   acl['resource_pattern']['resource_name'])
                acl_list.append(ACL(acl['principal'],
                                    acl['host'],
                                    ACLOperation(acl['operation']),
                                    ACLPermissionType(acl['permission_type']),
                                    resource_pattern))
            cli.create_acls(acl_list)

        logging.info(f'Snapshot restored successfully.')


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('folder')
    parser.add_argument('-d', '--topics')
    parser.add_argument('-topic_regex')
    parser.add_argument('-mode')
    args = parser.parse_args()

    restore_instance = Restore(args.folder, args.mode)
    restore_instance.restore(ast.literal_eval(args.topics) if args.topics else None, args.topic_regex)
