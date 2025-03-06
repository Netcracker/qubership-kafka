import os
import time

from PlatformLibrary import PlatformLibrary

environ = os.environ
managed_by_operator = environ.get("KAFKA_IS_MANAGED_BY_OPERATOR")
external = environ.get("EXTERNAL_KAFKA") is not None
namespace = environ.get("KAFKA_OS_PROJECT")
service = environ.get("KAFKA_HOST")
timeout = 300

if __name__ == '__main__':
    time.sleep(10)
    if external:
        print(f'Kafka is external, there is no way to check its state')
        time.sleep(30)
        exit(0)
    print("Checking Kafka deployments are ready")
    try:
        k8s_lib = PlatformLibrary(managed_by_operator)
    except Exception as e:
        print(e)
        exit(1)
    timeout_start = time.time()
    while time.time() < timeout_start + timeout:
        try:
            deployments = k8s_lib.get_deployment_entities_count_for_service(namespace, service)
            ready_deployments = k8s_lib.get_active_deployment_entities_count_for_service(namespace, service)
            print(f'[Check status] deployments: {deployments}, ready deployments: {ready_deployments}')
        except Exception as e:
            print(e)
            continue
        if deployments == ready_deployments and deployments != 0:
            print("Kafka deployments are ready")
            time.sleep(30)
            exit(0)
        time.sleep(10)
    print(f'Kafka deployments are not ready at least {timeout} seconds')
    exit(1)
