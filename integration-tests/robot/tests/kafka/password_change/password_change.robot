*** Variables ***
${TOPIC_NAME}                       client-password-change-topic
${KAFKA_SECRET_NAME}                %{KAFKA_HOST}-secret
${OPERATION_RETRY_COUNT}            60x
${OPERATION_RETRY_INTERVAL}         10s
${CONSUME_MESSAGE_RETRY_COUNT}      30
${CONSUME_MESSAGE_RETRY_INTERVAL}   2s

*** Settings ***
Library  Collections
Resource  ../../shared/keywords.robot
Suite Setup  Setup
Suite Teardown  Cleanup

*** Keywords ***
Setup
    ${admin}=  Create Admin Client
    Set Suite Variable  ${admin}
    ${postfix}=  Generate Random String  5
    Set Suite Variable  ${TOPIC_NAME_PATTERN}  ${TOPIC_NAME}-.{5}
    Set Suite Variable  ${TOPIC_NAME}  ${TOPIC_NAME}-${postfix}
    Set Suite Variable  ${ORIGINAL_CLIENT_PASSWORD}  ${EMPTY}
    Set Suite Variable  ${PASSWORD_RESTORED}  ${TRUE}
    ${brokers_count}=  Get Active Deployment Entities Count For Service  %{KAFKA_OS_PROJECT}  %{KAFKA_HOST}
    Set Suite Variable  ${BROKERS_COUNT}  ${brokers_count}
    Delete Topic By Pattern  ${admin}  ${TOPIC_NAME_PATTERN}

Cleanup
    Run Keyword If  ${PASSWORD_RESTORED} == ${FALSE}  Restore Client Password
    Delete Topic By Pattern  ${admin}  ${TOPIC_NAME_PATTERN}
    ${admin}=  Set Variable  ${None}

Decode Secret Value
    [Arguments]  ${value}
    ${decoded}=  Evaluate  base64.b64decode($value).decode('utf-8')  modules=base64
    RETURN  ${decoded}

Get Client Credentials From Kafka Secret
    ${secret}=  Get Secret  ${KAFKA_SECRET_NAME}  %{KAFKA_OS_PROJECT}
    ${client_username}=  Decode Secret Value  ${secret.data['client-username']}
    ${client_password}=  Decode Secret Value  ${secret.data['client-password']}
    RETURN  ${client_username}  ${client_password}  ${secret}

Patch Client Password
    [Arguments]  ${password}
    ${string_data}=  Create Dictionary  client-password=${password}
    ${body}=  Create Dictionary  stringData=${string_data}
    Patch Secret  ${KAFKA_SECRET_NAME}  %{KAFKA_OS_PROJECT}  ${body}

Import Kafka Library With Credentials
    [Arguments]  ${username}  ${password}
    Import Library  ../../shared/lib/KafkaLibrary.py
    ...  bootstrap_servers=${KAFKA_BOOTSTRAP_SERVERS}
    ...  namespace=%{KAFKA_OS_PROJECT}
    ...  host=${KAFKA_HOST}
    ...  port=%{KAFKA_PORT}
    ...  username=${username}
    ...  password=${password}
    ...  enable_ssl=%{KAFKA_ENABLE_SSL}
    ...  WITH NAME  AuthKafka

Restore Client Password
    Patch Client Password  ${ORIGINAL_CLIENT_PASSWORD}
    Wait For Kafka Brokers Ready
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Produce And Consume With Credentials  ${CLIENT_USERNAME}  ${ORIGINAL_CLIENT_PASSWORD}
    Set Suite Variable  ${PASSWORD_RESTORED}  ${TRUE}

Kafka Brokers Are Ready
    ${ready}=  Number Of Pods In Ready Status  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}
    Should Be Equal As Integers  ${ready}  ${BROKERS_COUNT}

Wait For Kafka Brokers Ready
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Kafka Brokers Are Ready

Check Consumed Message
    [Arguments]  ${consumer}  ${topic_name}  ${message}
    ${received_message}=  Consume Message  ${consumer}  ${topic_name}
    Should Contain  ${received_message}  ${message}

Produce And Consume With Credentials
    [Arguments]  ${username}  ${password}
    Import Kafka Library With Credentials  ${username}  ${password}
    ${producer}=  AuthKafka.Create Kafka Producer
    ${message}=  Create Test Message
    Produce Message  ${producer}  ${TOPIC_NAME}  ${message}
    ${consumer}=  AuthKafka.Create Kafka Consumer  ${TOPIC_NAME}
    Wait Until Keyword Succeeds  ${CONSUME_MESSAGE_RETRY_COUNT}  ${CONSUME_MESSAGE_RETRY_INTERVAL}
    ...  Check Consumed Message  ${consumer}  ${TOPIC_NAME}  ${message}
    Close Kafka Consumer  ${consumer}
    ${producer}=  Set Variable  ${None}
    ${consumer}=  Set Variable  ${None}

Produce With Credentials Should Fail
    [Arguments]  ${username}  ${password}
    Import Kafka Library With Credentials  ${username}  ${password}
    ${producer}=  AuthKafka.Create Kafka Producer
    ${message}=  Create Test Message
    Run Keyword And Expect Error  *
    ...  Produce Message  ${producer}  ${TOPIC_NAME}  ${message}  retries=${1}  delay=${1}
    ${producer}=  Set Variable  ${None}

*** Test Cases ***
Test Client Password Change
    [Tags]  kafka_password_change  kafka
    ${client_username}  ${client_password}  ${secret}=  Get Client Credentials From Kafka Secret
    Pass Execution If  '${client_username}' == '${EMPTY}' or '${client_password}' == '${EMPTY}'
    ...  Kafka client credentials are empty, password change is not applicable
    Set Suite Variable  ${CLIENT_USERNAME}  ${client_username}
    Set Suite Variable  ${ORIGINAL_CLIENT_PASSWORD}  ${client_password}

    ${annotations}=  Set Variable  ${secret.metadata.annotations}
    ${auto_restart}=  Evaluate  ($annotations or {}).get('kafkaservice.netcracker.com/auto-restart', 'false')
    Pass Execution If  '${auto_restart}' != 'true'
    ...  autoRestartOnSecretChange is disabled, password change restart is not automatic

    Create Topic  ${admin}  ${TOPIC_NAME}  ${1}  ${1}
    Produce And Consume With Credentials  ${client_username}  ${client_password}

    ${new_password}=  Generate Random String  16  [LETTERS][NUMBERS]
    Set Suite Variable  ${PASSWORD_RESTORED}  ${FALSE}
    Patch Client Password  ${new_password}

    Wait For Kafka Brokers Ready
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Produce And Consume With Credentials  ${client_username}  ${new_password}

    Produce With Credentials Should Fail  ${client_username}  ${client_password}

    [Teardown]  Run Keyword If  ${PASSWORD_RESTORED} == ${FALSE}  Restore Client Password
