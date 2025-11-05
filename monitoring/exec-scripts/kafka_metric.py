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

import logging
import os
import re
import time
import json
import urllib.request
import urllib.error
from logging.handlers import RotatingFileHandler
from multiprocessing.dummy import Pool as ThreadPool

import requests
from kafka.admin import KafkaAdminClient, ConfigResource, ConfigResourceType
from kafka.errors import NoBrokersAvailable

REQUEST_HEADERS = {
    'Accept': 'application/json',
    'Content-type': 'application/json'
}

logger = logging.getLogger(__name__)
KAFKA_USER = ""
KAFKA_PASSWORD = ""
KAFKA_TIMEOUT = 60
OS_PROJECT = ""
KAFKA_SERVICE_NAME = "kafka"
KAFKA_ADDRESSES = ""
KRAFT_ENABLED = False
KAFKA_SASL_MECHANISM = 'SCRAM-SHA-512'
KAFKA_ENABLE_SSL = False
KAFKA_TOTAL_BROKERS_COUNT = 0
RUNNING_AS_BROKER_STATE = 3
CONTROLLER_SHUTDOWN_STATE = 12
MAX_PROBLEM_TOPICS_STR_LEN = 350
ALLOWED_DIFFERENT_CONFIGS = ["listeners", "zookeeper.connect", "node.id"]
CA_CERT_PATH = '/tls/ca.crt'
TLS_CERT_PATH = '/tls/tls.crt'
TLS_KEY_PATH = '/tls/tls.key'
MIN_VERSION = "0.0.0"
MAX_VERSION = "x.x.x"


def __configure_logging(log):
    log.setLevel(logging.DEBUG)
    formatter = logging.Formatter(fmt='[%(asctime)s,%(msecs)03d][%(levelname)s] %(message)s',
                                  datefmt='%Y-%m-%dT%H:%M:%S')

    log_handler = RotatingFileHandler(filename='/opt/kafka-monitoring/exec-scripts/kafka_metric.log',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    log_handler.setFormatter(formatter)
    log_handler.setLevel(logging.DEBUG if os.getenv('KAFKA_MONITORING_SCRIPT_DEBUG') else logging.INFO)
    log.addHandler(log_handler)
    err_handler = RotatingFileHandler(filename='/opt/kafka-monitoring/exec-scripts/kafka_metric.err',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    err_handler.setFormatter(formatter)
    err_handler.setLevel(logging.ERROR)
    log.addHandler(err_handler)


def _determine_cluster_status_simple(active_brokers):
    # 0 - UP, 6 - DEGRADED, 10 - DOWN
    if not active_brokers:
        return 10
    elif len(active_brokers) < KAFKA_TOTAL_BROKERS_COUNT:
        return 6
    else:
        return 0


def _create_admin_client() -> KafkaAdminClient:
    security_protocol = 'SSL' if KAFKA_ENABLE_SSL else 'PLAINTEXT'
    ssl_cafile = CA_CERT_PATH if KAFKA_ENABLE_SSL and os.path.exists(CA_CERT_PATH) else None
    ssl_certfile = TLS_CERT_PATH if KAFKA_ENABLE_SSL and os.path.exists(TLS_CERT_PATH) else None
    ssl_keyfile = TLS_KEY_PATH if KAFKA_ENABLE_SSL and os.path.exists(TLS_KEY_PATH) else None
    configs = {
        'bootstrap_servers': KAFKA_ADDRESSES,
        'api_version_auto_timeout_ms': 6000
    }
    if KAFKA_USER and KAFKA_PASSWORD:
        configs['security_protocol'] = f'SASL_{security_protocol}'
        configs['sasl_mechanism'] = KAFKA_SASL_MECHANISM
        configs['sasl_plain_username'] = KAFKA_USER
        configs['sasl_plain_password'] = KAFKA_PASSWORD
    else:
        configs['security_protocol'] = security_protocol
    configs['ssl_cafile'] = ssl_cafile
    configs['ssl_certfile'] = ssl_certfile
    configs['ssl_keyfile'] = ssl_keyfile
    try:
        return KafkaAdminClient(**configs)
    except NoBrokersAvailable as e:
        logger.exception(f"Failed to create Kafka Admin Client: {e}")
        return None


# Remove configs which are unique for each broker
# and return broker configs which should be the same across all brokers
def _get_broker_configs(admin_client: KafkaAdminClient, broker_id) -> dict:
    obj = admin_client.describe_configs([ConfigResource(ConfigResourceType.BROKER, broker_id)])[0].to_object()
    config_list = [c for c in obj['resources'][0]['config_entries']]
    try:
        config = {}
        for c in config_list:
            config[c['config_names']] = c['config_value']
        del config["broker.id"]
        del config["log.dirs"]
        del config["advertised.listeners"]
        del config["broker.rack"]
        return config
    except Exception:
        logger.exception(f'Failed to describe broker {broker_id}')
        raise


# Return 'Yes' if configs of all brokers are the same, else return
# 'No: [{name of any discrepant config}: {value on one broker} VS {different value on another broker}]'
def _check_config_consistency(broker_configs_list: list) -> str:
    if not broker_configs_list:
        return "Empty configs"
    one_broker_configs = broker_configs_list.pop(0)
    for another_broker_configs in broker_configs_list:
        if one_broker_configs != another_broker_configs:
            for conf in one_broker_configs:
                if conf not in ALLOWED_DIFFERENT_CONFIGS and one_broker_configs[conf] != another_broker_configs[conf]:
                    return f'No: [{conf}]: {one_broker_configs[conf]} VS {another_broker_configs[conf]}'
    return "Yes"


# # Return Kafka version in [major version].x format (e.x. 2.7.x)
# def _get_kafka_version(broker_configs: dict, admin_client: KafkaAdminClient | None = None) -> str:
#     if admin_client is not None:
#         try:
#             ver = admin_client._client.check_version(timeout=KAFKA_TIMEOUT)
#             try:
#                 with open("/opt/kafka-monitoring/exec-scripts/kafka_version_debug.txt", "a", encoding="utf-8") as f:
#                     f.write(f"Detected Kafka version via ApiVersions: {ver}\n")
#             except Exception as log_err:
#                 logger.warning(f"Failed to write debug version file: {log_err}")
#             parts = (ver or "").split(".")
#             if len(parts) >= 2:
#                 return f"{parts[0]}.{parts[1]}.x"
#         except Exception as e:
#             logger.warning(f"ApiVersions detection failed: {e}")

    # ZK and old releases
    # return broker_configs["inter.broker.protocol.version"].split('-')[0] + '.x'


# Collects metrics for each broker in a separate thread
def _get_broker_metrics_simple(broker_id, admin_client):
    if not admin_client:
        return -1
    special_broker_metrics = {'broker_state': str}

    logger.info(f'Broker id={broker_id}.')
    try:
        broker_configs = _get_broker_configs(admin_client, str(broker_id))
        kafka_version = _get_kafka_version(broker_configs, admin_client)

    except Exception:
        logger.exception('Exception occurred working with broker id: %s', broker_id)
        return -1
    return broker_id, broker_configs, special_broker_metrics, kafka_version


# Concatenates metrics received for each broker for further processing
def _concatenate_all_brokers_metrics_simple(brokers_metrics):
    all_brokers_metrics = {'active_brokers': list(),
                           'kafka_version': ""}
    broker_configs_list = list()
    for broker_id, broker_configs, special_broker_metrics, kafka_version in brokers_metrics:
        all_brokers_metrics['active_brokers'].append(broker_id)
        broker_configs_list.append(broker_configs)
        all_brokers_metrics['kafka_version'] = kafka_version
    return all_brokers_metrics, None, broker_configs_list, None


def _parse_version(v):
    v = v.replace("x", "99")
    return tuple(map(int, v.split('.')))


def _is_version_compatible(version, min_version, max_version):
    version_parts = _parse_version(version)
    min_version_parts = _parse_version(min_version)
    max_version_parts = _parse_version(max_version)

    return min_version_parts <= version_parts <= max_version_parts


def _collect_compatibility_metric(admin_client: KafkaAdminClient):
    broker_ids = [broker['node_id'] for broker
                  in admin_client.describe_cluster()['brokers']]
    min_broker_version = None
    for broker_id in broker_ids:
        broker_configs = _get_broker_configs(admin_client, str(broker_id))
        kafka_version = _get_kafka_version(broker_configs, admin_client)
        if min_broker_version is None or _parse_version(kafka_version) < _parse_version(min_broker_version):
            min_broker_version = kafka_version

    version_compatible = 1 if _is_version_compatible(min_broker_version, MIN_VERSION, MAX_VERSION) else 0

    message = f'supplementary_services,' \
              f'application=kafka,' \
              f'namespace={OS_PROJECT},' \
              f'application_version={min_broker_version},' \
              f'min_version={MIN_VERSION},' \
              f'max_version={MAX_VERSION}' \
              f' version_compatible={version_compatible}i'
    return message


def _is_kraft(admin_client, broker_id):
    config_resource = ConfigResource(ConfigResourceType.BROKER, str(broker_id))
    configs = admin_client.describe_configs([config_resource])
    broker_configs = configs[0].resources[0][4]
    kraft_mode = False
    for config_entry in broker_configs:
        if config_entry[0] == "controller.quorum.voters" and config_entry[1] != "":
            kraft_mode = True
            break

    if kraft_mode:
        return True
    else:
        return False


def _collect_metrics():
    admin_client = _create_admin_client()
    is_kraft_enabled = KRAFT_ENABLED
    if not admin_client:
        broker_ids = []
    else:
        broker_ids = [broker['node_id'] for broker
                      in admin_client.describe_cluster()['brokers']]
        is_kraft_enabled = _is_kraft(admin_client, broker_ids[0])
        if KRAFT_ENABLED != is_kraft_enabled:
            logger.warning(f'KRAFT_ENABLED environment variable value is %s, but value returned from broker is %s',
                           KRAFT_ENABLED, is_kraft_enabled)
    args = [(idx, admin_client) for idx in broker_ids]
    metrics_func = _get_broker_metrics_simple
    collect_func = _concatenate_all_brokers_metrics_simple
    broker_count = len(broker_ids)

    if broker_count > 0:
        pool = ThreadPool(broker_count)
        metrics_per_broker = pool.starmap(metrics_func, args)
        pool.close()
        pool.join()
        metrics_per_broker = [x for x in metrics_per_broker if x != -1]
    else:
        metrics_per_broker = []

    all_brokers_metrics, controller_metrics, broker_configs_list, output_message = collect_func(
        metrics_per_broker)
    active_brokers = all_brokers_metrics["active_brokers"]
    logger.info('Active brokers: %s', active_brokers)
    same_configs = _check_config_consistency(broker_configs_list)
    kafka_version = all_brokers_metrics['kafka_version']
    cluster_status = _determine_cluster_status_simple(active_brokers)
    quorum_mode = 101 if is_kraft_enabled else 100
    logger.info('Cluster status: %s', cluster_status)
    message = f'kafka_cluster ' \
              f'size={len(active_brokers)}i,' \
              f'status={cluster_status},' \
              f'quorum_mode={quorum_mode}i,' \
              f'same_configs=\"{same_configs}\",' \
              f'kafka_version=\"{kafka_version}\"'
    if admin_client is not None:
        compatibility_message = _collect_compatibility_metric(admin_client)
        message = f'{message}\n{compatibility_message}'
    return message


def _str2bool(v: str):
    return v.lower() in ("yes", "true", "t", "1")


def _normalize_version_family(v: str) -> str:
    parts = v.split(".")
    if len(parts) >= 2:
        return f"{parts[0]}.{parts[1]}.x"
    if len(parts) == 1:
        return f"{parts[0]}.0.x"
    return "unknown"


def _get_kafka_version(broker_configs: dict,
                       admin_client: KafkaAdminClient | None = None) -> str:
    if broker_configs.get("inter.broker.protocol.version"):
        return broker_configs["inter.broker.protocol.version"].split('-')[0] + '.x'

    if admin_client is not None:
        try:
            ver = admin_client._client.check_version(timeout=KAFKA_TIMEOUT)
            if isinstance(ver, tuple):
                major = ver[0]
                minor = ver[1] if len(ver) > 1 else 0
                return f"{major}.{minor}.x"
            if isinstance(ver, str) and ver:
                return _normalize_version_family(ver)
        except Exception as e:
            logger.warning(f"ApiVersions detection failed: {e}")
    return "4.x"


def run():
    try:
        logger.info('Start script execution...')
        global KAFKA_SERVICE_NAME, KAFKA_TOTAL_BROKERS_COUNT, OS_PROJECT, KAFKA_USER, KAFKA_PASSWORD, KAFKA_TIMEOUT, KAFKA_ADDRESSES, KRAFT_ENABLED, KAFKA_SASL_MECHANISM, KAFKA_ENABLE_SSL, MIN_VERSION, MAX_VERSION
        KAFKA_SERVICE_NAME = os.getenv('KAFKA_SERVICE_NAME')
        KAFKA_ADDRESSES = os.getenv('KAFKA_ADDRESSES')
        KRAFT_ENABLED = _str2bool(os.getenv("KRAFT_ENABLED", "false"))
        KAFKA_SASL_MECHANISM = os.getenv('KAFKA_SASL_MECHANISM', 'SCRAM-SHA-512')
        KAFKA_ENABLE_SSL = _str2bool(os.getenv("KAFKA_ENABLE_SSL", "false"))
        KAFKA_TOTAL_BROKERS_COUNT = int(os.getenv('KAFKA_TOTAL_BROKERS_COUNT'))
        OS_PROJECT = os.getenv('OS_PROJECT')
        KAFKA_USER = os.getenv('KAFKA_USER')
        KAFKA_PASSWORD = os.getenv('KAFKA_PASSWORD')
        MIN_VERSION = os.getenv('MIN_VERSION', MIN_VERSION)
        MAX_VERSION = os.getenv('MAX_VERSION', MAX_VERSION)
        timeout = os.getenv('KAFKA_EXEC_PLUGIN_TIMEOUT', "60s")
        KAFKA_TIMEOUT = int(re.compile(r"(\d+)").match(timeout).group(1))
        message = _collect_metrics()
        logger.debug('Message to send:\n%s', message)
        logger.info('End script execution!\n')
        print(message)
    except Exception:
        logger.exception('Exception occurred during script execution:')
        raise


if __name__ == "__main__":
    __configure_logging(logger)
    start = time.time()
    run()
    logger.info(f'Time of execution is {time.time() - start}\n')
