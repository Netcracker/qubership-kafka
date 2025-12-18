*** Variables ***
${KAFKA_SERVICE_NAME}                  %{KAFKA_HOST}
${KAFKA_OS_PROJECT}                    %{KAFKA_OS_PROJECT}
${ZOOKEEPER_SHUTDOWN_TOPIC_NAME}       zookeeper-shutdown-test-topic
${PARTITION_LEADER_CRASH_TOPIC_NAME}   partition-leader-crash-test-topic
${ZOOKEEPER_AFTER_RESTART_TOPIC_NAME}  zookeeper-after-restart-test-topic
${KAFKA_DISK_FILLED_TOPIC_NAME}        kafka-disk-filled-topic
${LIBRARIES}                           /opt/robot/tests/shared/lib
${OPERATION_RETRY_COUNT}               17x
${OPERATION_RETRY_INTERVAL}            4s
${SLEEP_TIME}                          60s
${DISK_FILLED_RETRY_COUNT}             30x
${DISK_FILLED_RETRY_INTERVAL}          5s

*** Settings ***
Library  OperatingSystem
Resource  ../../shared/keywords.robot
Suite Setup  Setup
Suite Teardown  Cleanup

*** Keywords ***
Setup
    ${producer} =  Create Kafka Producer
    Set Suite Variable  ${producer}
    ${admin} =  Create Admin Client
    Set Suite Variable  ${admin}
    ${postfix} =  Generate Random String  5
    Set Suite Variable  ${ZOOKEEPER_SHUTDOWN_TOPIC_NAME_PATTERN}  ${ZOOKEEPER_SHUTDOWN_TOPIC_NAME}-.{5}
    Set Suite Variable  ${ZOOKEEPER_SHUTDOWN_TOPIC_NAME}  ${ZOOKEEPER_SHUTDOWN_TOPIC_NAME}-${postfix}
    Set Suite Variable  ${PARTITION_LEADER_CRASH_TOPIC_NAME_PATTERN}  ${PARTITION_LEADER_CRASH_TOPIC_NAME}-.{5}
    Set Suite Variable  ${PARTITION_LEADER_CRASH_TOPIC_NAME}  ${PARTITION_LEADER_CRASH_TOPIC_NAME}-${postfix}
    Set Suite Variable  ${ZOOKEEPER_AFTER_RESTART_TOPIC_NAME_PATTERN}  ${ZOOKEEPER_AFTER_RESTART_TOPIC_NAME}-.{5}
    Set Suite Variable  ${ZOOKEEPER_AFTER_RESTART_TOPIC_NAME}  ${ZOOKEEPER_AFTER_RESTART_TOPIC_NAME}-${postfix}
    Set Suite Variable  ${KAFKA_DISK_FILLED_TOPIC_NAME_PATTERN}  ${KAFKA_DISK_FILLED_TOPIC_NAME}-.{5}
    Delete Topics

Check Consumed Message
    [Arguments]  ${consumer}  ${topic_name}  ${message}
    ${received_message} =  Consume Message  ${consumer}  ${topic_name}
    Should Contain  ${received_message}  ${message}

Check Topic Management
    ${pod_names}=  Get Pod Names By Service Name  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}
    ${replication_factor}=  Get Length  ${pod_names}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Create Topic  ${admin}  ${ZOOKEEPER_AFTER_RESTART_TOPIC_NAME}  ${replication_factor}  ${1}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Delete Topic By Pattern  ${admin}  ${ZOOKEEPER_AFTER_RESTART_TOPIC_NAME_PATTERN}

Find Out Leader Among Brokers
    [Arguments]  ${broker_envs}  ${topic_name}
    ${leader}=  Find Out Leader Broker  ${admin}  ${broker_envs}  ${topic_name}
    Should Not Be Empty  ${leader}
    [Return]  ${leader}

Delete Topics
    Delete Topic By Pattern  ${admin}  ${ZOOKEEPER_SHUTDOWN_TOPIC_NAME_PATTERN}
    Delete Topic By Pattern  ${admin}  ${PARTITION_LEADER_CRASH_TOPIC_NAME_PATTERN}
    Delete Topic By Pattern  ${admin}  ${ZOOKEEPER_AFTER_RESTART_TOPIC_NAME_PATTERN}
    Delete Topic By Pattern  ${admin}  ${KAFKA_DISK_FILLED_TOPIC_NAME_PATTERN}

Cleanup
    Sleep  60s  reason=Kafka pod can be unavailable
    ${producer} =  Set Variable  ${None}
    Run Keyword If Any Tests Failed
    ...  Scale Up Full Service  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}
    Delete Topics
    ${admin} =  Set Variable  ${None}

Scale Up Full Service
    [Arguments]  ${service}  ${project}
    Scale Up Deployment Entities By Service Name  ${service}  ${project}  with_check=True  replicas=1

Create Kafka Topic With Exception
    [Arguments]  ${admin}  ${replication_factor}
    ${postfix} =  Generate Random String  5
    ${exception_message}=  Create Topic With Expected Exception
    ...  ${admin}
    ...  ${KAFKA_DISK_FILLED_TOPIC_NAME}-${postfix}
    ...  ${replication_factor}
    ...  ${1}
    Should Contain  ${exception_message}
    ...  InvalidReplicationFactorError

*** Test Cases ***
Test Producing And Consuming Data Without Zookeeper
    [Tags]  kafka_ha  kafka_ha_without_zookeeper  kafka
    ${pod_names}=  Get Pod Names By Service Name  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}
    ${replication_factor}=  Get Length  ${pod_names}
    Create Topic  ${admin}  ${ZOOKEEPER_SHUTDOWN_TOPIC_NAME}  ${replication_factor}  ${1}

    ${message} =  Create Test Message
    Produce Message  ${producer}  ${ZOOKEEPER_SHUTDOWN_TOPIC_NAME}  ${message}

    Scale Down Deployment Entities By Service Name  %{ZOOKEEPER_HOST}  %{ZOOKEEPER_OS_PROJECT}  with_check=True

    ${consumer} =  Create Kafka Consumer  ${ZOOKEEPER_SHUTDOWN_TOPIC_NAME}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Consumed Message  ${consumer}  ${ZOOKEEPER_SHUTDOWN_TOPIC_NAME}  ${message}
    ${message_without_zookeeper} =  Create Test Message
    Produce Message  ${producer}  ${ZOOKEEPER_SHUTDOWN_TOPIC_NAME}  ${message_without_zookeeper}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Consumed Message  ${consumer}  ${ZOOKEEPER_SHUTDOWN_TOPIC_NAME}  ${message_without_zookeeper}
    Close Kafka Consumer  ${consumer}
    ${consumer} =  Set Variable  ${None}

    Scale Up Full Service  %{ZOOKEEPER_HOST}  %{ZOOKEEPER_OS_PROJECT}
    Sleep  ${SLEEP_TIME}  reason=Waiting for Kafka to connect to ZooKeeper after restart

    Check Topic Management

    [Teardown]  Scale Up Full Service  %{ZOOKEEPER_HOST}  %{ZOOKEEPER_OS_PROJECT}

Test Producing And Consuming Data Without Kafka Master
    [Tags]  kafka_ha  kafka_ha_without_kafka_master  kafka
    ${admin} =  Create Admin Client
    ${env_names}=  Create List  BROKER_ID
    ${broker_envs}=  Get Pod Container Environment Variables For Service
    ...  %{KAFKA_OS_PROJECT}  %{KAFKA_HOST}  kafka  ${env_names}
    ${replication_factor} =  Get Length  ${broker_envs}
    Create Topic  ${admin}  ${PARTITION_LEADER_CRASH_TOPIC_NAME}  ${replication_factor}  ${1}

    ${message} =  Create Test Message
    Produce Message  ${producer}  ${PARTITION_LEADER_CRASH_TOPIC_NAME}  ${message}

    ${leader} =  Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Find Out Leader Among Brokers  ${broker_envs}  ${PARTITION_LEADER_CRASH_TOPIC_NAME}

    ${admin} =  Set Variable  ${None}
    Scale Down Deployment Entities By Service Name  ${leader}  %{KAFKA_OS_PROJECT}  with_check=True
    Remove From Dictionary  ${broker_envs}  ${leader}
    Sleep  ${SLEEP_TIME}  reason=Waiting for Kafka to choose new leader

    ${admin} =  Create Admin Client
    ${new_leader} =  Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Find Out Leader Among Brokers  ${broker_envs}  ${PARTITION_LEADER_CRASH_TOPIC_NAME}
    ${consumer} =  Create Kafka Consumer  ${PARTITION_LEADER_CRASH_TOPIC_NAME}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Consumed Message  ${consumer}  ${PARTITION_LEADER_CRASH_TOPIC_NAME}  ${message}
    Close Kafka Consumer  ${consumer}
    ${consumer} =  Set Variable  ${None}

    [Teardown]  Scale Up Full Service  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}

Test Invalid Topic Creation
    [Tags]  kafka_ha  kafka_ha_disk_is_filled
    ${admin} =  Create Admin Client
    ${env_names} =  Create List  BROKER_ID
    ${broker_envs} =  Get Pod Container Environment Variables For Service
    ...  %{KAFKA_OS_PROJECT}  %{KAFKA_HOST}  kafka  ${env_names}
    ${replication_factor} =  Get Length  ${broker_envs}

    ${replicas}=  Get Active Deployment Entities Count For Service  ${KAFKA_OS_PROJECT}  ${KAFKA_SERVICE_NAME}
    Pass Execution If  ${replicas} < 3  Kafka cluster has less than 3 brokers
    Scale Down Deployment Entities By Service Name  ${KAFKA_SERVICE_NAME}-1  ${KAFKA_OS_PROJECT}
    Sleep  ${SLEEP_TIME}  reason=Waiting for Kafka cluster to notice broker is down

    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Create Kafka Topic With Exception  ${admin}  ${replication_factor}

    [Teardown]  Scale Up Full Service  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}