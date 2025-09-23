*** Variables ***
${TOPIC_NAME}                      consumer-producer-tests-topic
${CONSUME_MESSAGE_RETRY_COUNT}     30
${CONSUME_MESSAGE_RETRY_INTERVAL}  2s

*** Settings ***
Resource  ../../shared/keywords.robot
Suite Setup  Setup
Suite Teardown  Cleanup

*** Keywords ***
Setup
    ${admin} =  Create Admin Client
    Set Suite Variable  ${admin}
    ${postfix} =  Generate Random String  5
    Set Suite Variable  ${TOPIC_NAME_PATTERN}  ${TOPIC_NAME}-.{5}
    Set Suite Variable  ${TOPIC_NAME}  ${TOPIC_NAME}-${postfix}
    Delete Topic By Pattern  ${admin}  ${TOPIC_NAME_PATTERN}

Check Consumed Message
    [Arguments]  ${consumer}  ${topic_name}  ${message}
    ${receivedMessage} =  Consume Message  ${consumer}  ${topic_name}
    Should Contain  ${receivedMessage}  ${message}

Cleanup
    Delete Topic By Pattern  ${admin}  ${TOPIC_NAME_PATTERN}
    ${admin} =  Set Variable  ${None}

*** Test Cases ***
Test Producing And Consuming Data
    [Tags]  kafka_consumer_producer  kafka
    Create Topic  ${admin}  ${TOPIC_NAME}  ${1}  ${1}
    ${producer} =  Create Kafka Producer
    ${message} =  Create Test Message
    Produce Message  ${producer}  ${TOPIC_NAME}  ${message}
    ${consumer} =  Create Kafka Consumer  ${TOPIC_NAME}
    Wait Until Keyword Succeeds  ${CONSUME_MESSAGE_RETRY_COUNT}  ${CONSUME_MESSAGE_RETRY_INTERVAL}
    ...  Check Consumed Message  ${consumer}  ${TOPIC_NAME}  ${message}
    ${producer} =  Set Variable  ${None}
    Close Kafka Consumer  ${consumer}
    ${consumer} =  Set Variable  ${None}