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
import time

from PlatformLibrary import PlatformLibrary

environ = os.environ
managed_by_operator = environ.get("KAFKA_IS_MANAGED_BY_OPERATOR")
external = environ.get("EXTERNAL_KAFKA") is not None
namespace = environ.get("KAFKA_OS_PROJECT")
kafka = environ.get("KAFKA_HOST")
backup_daemon = environ.get("BACKUP_DAEMON_HOST")
timeout = 600
# Kafka can be updated by the operator node-by-node (rolling upgrade). During such an
# upgrade all deployments may momentarily report "ready" in the gap between two pods
# being rolled, so a single ready observation is not enough to assume the cluster is
# stable. Require several consecutive ready observations spaced apart in time before
# proceeding, so an in-progress rolling upgrade is reliably detected.
stability_checks = int(environ.get("READINESS_STABILITY_CHECKS", 3))
stability_retry_delay = int(environ.get("READINESS_STABILITY_RETRY_DELAY", 20))


def deployments_ready(k8s_lib) -> bool:
    deployments = k8s_lib.get_deployment_entities_count_for_service(namespace, kafka)
    ready_deployments = k8s_lib.get_active_deployment_entities_count_for_service(namespace, kafka)
    if backup_daemon:
        print(f'Adding Kafka Backup Daemon to check')
        deployments += k8s_lib.get_deployment_entities_count_for_service(namespace, backup_daemon, 'component')
        ready_deployments += k8s_lib.get_active_deployment_entities_count_for_service(namespace, backup_daemon, 'component')
    print(f'[Check status] deployments: {deployments}, ready deployments: {ready_deployments}')
    return deployments == ready_deployments and deployments != 0


if __name__ == '__main__':
    time.sleep(20)
    if external:
        if not backup_daemon:
            print(f'Kafka is external, there is no way to check its state')
            time.sleep(30)
            exit(0)
        else:
            print(f'Kafka is external, but Backup Daemon persist, checking its state')
    print("Checking Kafka deployments are ready")
    try:
        k8s_lib = PlatformLibrary(managed_by_operator)
    except Exception as e:
        print(e)
        exit(1)
    consecutive_ready = 0
    timeout_start = time.time()
    while time.time() < timeout_start + timeout:
        try:
            ready = deployments_ready(k8s_lib)
        except Exception as e:
            print(e)
            consecutive_ready = 0
            time.sleep(10)
            continue
        if ready:
            consecutive_ready += 1
            print(f'Kafka deployments are ready ({consecutive_ready}/{stability_checks} consecutive checks)')
            if consecutive_ready >= stability_checks:
                print("Kafka deployments are ready and stable")
                time.sleep(30)
                exit(0)
            time.sleep(stability_retry_delay)
        else:
            if consecutive_ready:
                print('Kafka deployments became not ready, resetting stability counter '
                      '(rolling upgrade may be in progress)')
            consecutive_ready = 0
            time.sleep(10)
    print(f'Kafka deployments are not ready and stable at least {timeout} seconds')
    exit(1)
