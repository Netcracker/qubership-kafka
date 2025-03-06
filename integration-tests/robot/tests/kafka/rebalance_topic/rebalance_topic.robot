*** Variables ***
${KAFKA_SERVICE_NAME}         %{KAFKA_HOST}
${KAFKA_REBALANCE_TOPIC}      kafka-rebalance-topic
${SLEEP}                      5s

*** Settings ***
Library  RequestsLibrary
Resource  ../../shared/keywords.robot
Suite Setup  Prepare

*** Keywords ***
Prepare
    ${postfix}=  Generate Random String  5
    Set Suite Variable  ${KAFKA_REBALANCE_TOPIC}  ${KAFKA_REBALANCE_TOPIC}-${postfix}
    ${admin}=  Create Admin Client
    Set Suite Variable  ${admin}

Cleanup
    Delete Topic  ${admin}  ${KAFKA_REBALANCE_TOPIC}

Check Topic Exists
    [Arguments]  ${topic_name}
    ${list_topics}=  Get Topics List  ${admin}
    Should Contain  ${list_topics}  ${topic_name}

Check Replication Factor For Topic
    [Arguments]  ${admin}  ${topic_name}  ${expected_rf}
    ${rf}=  Get Replication Factor  ${admin}  ${topic_name}
    Should Be Equal  ${expected_rf}  ${rf}

*** Test Cases ***
Test Rebalance Topic
    [Tags]  kafka  rebalance
    ${replicas}=  Get Active Deployment Entities Count For Service  %{KAFKA_OS_PROJECT}  ${KAFKA_SERVICE_NAME}
    Pass Execution If  ${replicas} < 3  Kafka cluster has less than 3 brokers
    Create Topic  ${admin}  ${KAFKA_REBALANCE_TOPIC}  ${3}  ${1}
    Sleep  ${SLEEP}
    Check Topic Exists  ${KAFKA_REBALANCE_TOPIC}
    ${kafka_pod}=  Get Pod Names For Deployment Entity  ${KAFKA_SERVICE_NAME}-1  %{KAFKA_OS_PROJECT}
    ${pod}=  Get From List  ${kafka_pod}  0
    Execute Command In Pod  ${pod}  %{KAFKA_OS_PROJECT}  ./bin/kafka-partitions.sh rebalance_topic ${KAFKA_REBALANCE_TOPIC}  container=kafka
    Sleep  ${SLEEP}
    Check Replication Factor For Topic  ${admin}  ${KAFKA_REBALANCE_TOPIC}  ${replicas}
    [Teardown]  Cleanup
