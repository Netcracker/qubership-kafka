# Copyright 2024-2025 NetCracker Technology Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import logging
import os
import requests
from logging.handlers import RotatingFileHandler

logger = logging.getLogger(__name__)


def __configure_logging(log):
    log.setLevel(logging.DEBUG)
    formatter = logging.Formatter(fmt='[%(asctime)s,%(msecs)03d][%(levelname)s] %(message)s',
                                  datefmt='%Y-%m-%dT%H:%M:%S')
    log_handler = RotatingFileHandler(filename='/opt/kafka-kafka-mirror-maker-monitoring/exec-scripts/health_metric.log',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    log_handler.setFormatter(formatter)
    log_handler.setLevel(logging.DEBUG if os.getenv('KAFKA_MIRROR_MAKER_MONITORING_SCRIPT_DEBUG') else logging.INFO)
    log.addHandler(log_handler)
    err_handler = RotatingFileHandler(filename='/opt/kafka-kafka-mirror-maker-monitoring/exec-scripts/health_metric.err',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    err_handler.setFormatter(formatter)
    err_handler.setLevel(logging.ERROR)
    log.addHandler(err_handler)


def _get_number_of_alive_nodes(servers: list, timeout):
    """
    Calculates and returns number of alive nodes in KMM cluster.
    :param servers: list of server urls
    :return: number of alive nodes
    """
    nodes_number = 0
    for server in servers:
        try:
            response = requests.get(f'{server}/metrics/kafka.connect.mirror:type=MirrorSourceConnector', timeout=timeout)
            if response.status_code == 200 or response.status_code == 401:
                nodes_number += 1
        except:
            logger.exception(f'There are problems connecting to the server {server}')
    return nodes_number


def _get_status_code(alive_nodes: int, total_nodes: int):
    """
    Receives status of KMM nodes.
    :param alive_nodes: number of alive nodes
    :param total_nodes: total number of nodes
    :return: 0 - if all nodes are alive;
             1 - if not all nodes are alive;
             2 - if all nodes are failed.
    """
    if total_nodes == alive_nodes:
        return 0
    elif alive_nodes == 0:
        return 2
    else:
        return 1


def _get_list_of_servers(servers: str):
    return [element.replace("'", "") for element in servers[1:-1].split(",")]


def run():
    servers = os.getenv('PROMETHEUS_URLS')
    servers_list = _get_list_of_servers(servers)
    logger.debug(f'Servers are {servers_list}')
    health_timeout = os.getenv('KMM_HEALTH_TIMEOUT', 6)
    alive_nodes = _get_number_of_alive_nodes(servers_list, health_timeout)
    total_nodes = len(servers_list)
    status_code = _get_status_code(alive_nodes, total_nodes)
    failed_nodes = total_nodes - alive_nodes

    print(f'health status_code={status_code},alive_nodes={alive_nodes},total_nodes={total_nodes},failed_nodes={failed_nodes}')


if __name__ == "__main__":
    __configure_logging(logger)
    run()
