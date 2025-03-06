*** Variables ***
${ZOOKEEPER_SHUTDOWN_TOPIC_NAME}       zookeeper-shutdown-test-topic
${PARTITION_LEADER_CRASH_TOPIC_NAME}   partition-leader-crash-test-topic
${ZOOKEEPER_AFTER_RESTART_TOPIC_NAME}  zookeeper-after-restart-test-topic
${KAFKA_DISK_FILLED_TOPIC_NAME}        kafka-disk-filled-topic
${LIBRARIES}                           /opt/robot/tests/shared/lib
${OPERATION_RETRY_COUNT}               17x
${OPERATION_RETRY_INTERVAL}            4s
${SLEEP_TIME}                          20s
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
    [Arguments]  ${consumer}  ${message}
    ${received_message} =  Consume Message  ${consumer}
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
    Sleep  10s  reason=Kafka pod can be unavailable
    ${producer} =  Set Variable  ${None}
    Run Keyword If Any Tests Failed
    ...  Scale Up Full Service  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}
    Delete Topics
    ${admin} =  Set Variable  ${None}

Get Filled Space
    [Arguments]  ${pod_name}
    ${disk_information}  ${errors}  Execute Command In Pod  ${pod_name}  %{KAFKA_OS_PROJECT}
    ...  du -m /var/opt/kafka/data -d 0
    ${filled_space_in_mb}  ${disk_folder} =  Split String  ${disk_information}  ${EMPTY}  1
    [Return]  ${filled_space_in_mb}

Get Disk Space
    [Documentation]  There are cases when disk space differs from PV size:
    ...  1) Available disk space is much less than PV size (for example, command 'df -h' shows that
    ...  3G is available, but PV size is 10G).
    ...  2) If classical hostPath is used, command 'df -h' shows free space on the root system, but
    ...  in reality available space depends on PV size (for example, command 'df -h' shows
    ...  that 80G is available, but PV size is 2G).
    [Arguments]  ${pod_name}
    ${full_information}  ${errors}  Execute Command In Pod  ${pod_name}  %{KAFKA_OS_PROJECT}
    ...  df -m /var/opt/kafka/data
    ${header}  ${disk_information}  Split String  ${full_information}  \n  1
    ${disk}  ${space_size_in_mb}  ${rest}  Split String  ${disk_information}  ${EMPTY}  2
    ${space_size_int} =  Convert To Integer  ${space_size_in_mb}
    ${pv_size_in_mb} =  Evaluate  %{KAFKA_VOLUME_SIZE} * 1024
    ${minimum} =  Evaluate  min(${space_size_int}, ${pv_size_in_mb})
    [Return]  ${minimum}

Check Disk Is Full
    [Arguments]  ${pod_name}  ${disk_space}
    ${filled_space} =  Get Filled Space  ${pod_name}
    ${disk_fullness} =  Evaluate  100 * ${filled_space} / ${disk_space}
    Should Be True  ${disk_fullness} > 90

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

Recovery After Disk Is Full
    [Arguments]  ${pod_name}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Clean Disk Space  ${pod_name}
    Scale Up Full Service  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}

Clean Disk Space
    [Arguments]  ${pod_name}
    ${result}  ${errors} =  Execute Command In Pod  ${pod_name}  %{KAFKA_OS_PROJECT}  rm /var/opt/kafka/data/busy_space
    Should Match Regexp  ${errors}  rm: can('|no)t remove '/var/opt/kafka/data/busy_space': No such file or directory

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
    ...  Check Consumed Message  ${consumer}  ${message}
    ${message_without_zookeeper} =  Create Test Message
    Produce Message  ${producer}  ${ZOOKEEPER_SHUTDOWN_TOPIC_NAME}  ${message_without_zookeeper}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Consumed Message  ${consumer}  ${message_without_zookeeper}
    Close Kafka Consumer  ${consumer}
    ${consumer} =  Set Variable  ${None}

    Scale Up Full Service  %{ZOOKEEPER_HOST}  %{ZOOKEEPER_OS_PROJECT}
    Sleep  ${SLEEP_TIME}  reason=Waiting for Kafka to connect to ZooKeeper after restart

    Check Topic Management

    [Teardown]  Run Keywords  Cleanup  AND  Scale Up Full Service  %{ZOOKEEPER_HOST}  %{ZOOKEEPER_OS_PROJECT}

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
    ...  Check Consumed Message  ${consumer}  ${message}
    Close Kafka Consumer  ${consumer}
    ${consumer} =  Set Variable  ${None}

    [Teardown]  Scale Up Full Service  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}

Test Disk Is Filled On One Node
    [Tags]  kafka_ha  kafka_ha_disk_is_filled
    ${pod_names}=  Get Pod Names By Service Name  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}
    ${pod_name}=  Get From List  ${pod_names}  0
    ${admin} =  Create Admin Client
    ${postfix} =  Generate Random String  5
    ${replication_factor} =  Get Length  ${pod_names}
    Create Topic  ${admin}  ${KAFKA_DISK_FILLED_TOPIC_NAME}-${postfix}  ${replication_factor}  ${1}

    ${filled_space_in_mb} =  Get Filled Space  ${pod_name}
    ${disk_space_in_mb} =  Get Disk Space  ${pod_name}
    ${20_gigabytes} =  Evaluate  20 * 1024
    Run Keyword If  ${disk_space_in_mb} > ${20_gigabytes}
    ...  Pass Execution  Current test can't be executed due to too large size (${disk_space_in_mb}Mb) of the Kafka storage
    ${free_space_in_mb} =  Evaluate  ${disk_space_in_mb} - ${filled_space_in_mb}
    ${evaluate_amount_of_blocks} =  Evaluate  ${free_space_in_mb} / 50
    ${blocks_count} =  Convert To Integer  ${evaluate_amount_of_blocks}
    Log  Node space is filling  DEBUG
    Run Keyword And Ignore Error  Execute Command In Pod  ${pod_name}  %{KAFKA_OS_PROJECT}
    ...  dd if=/dev/zero of=/var/opt/kafka/data/busy_space bs=50M count=${blocks_count}

    Wait Until Keyword Succeeds  ${DISK_FILLED_RETRY_COUNT}  ${DISK_FILLED_RETRY_INTERVAL}
    ...  Check Disk Is Full  ${pod_name}  ${disk_space_in_mb}
    Log  Node space is filled with more than 90%  DEBUG

    Set Test Variable  ${index}  -1
    Sleep  20s  reason=Kafka pod can be unavailable
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Create Kafka Topic With Exception  ${admin}  ${replication_factor}

    [Teardown]  Recovery After Disk Is Full  ${pod_name}