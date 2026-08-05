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
    # Defaults first so Suite Teardown is safe even if Setup fails later
    Set Suite Variable  ${PASSWORD_RESTORED}  ${TRUE}
    Set Suite Variable  ${admin}  ${NONE}
    Set Suite Variable  ${ORIGINAL_CLIENT_PASSWORD}  ${EMPTY}
    Set Suite Variable  ${CLIENT_USERNAME}  ${EMPTY}
    Set Suite Variable  ${BROKERS_COUNT}  ${0}
    ${postfix}=  Generate Random String  5
    Set Suite Variable  ${TOPIC_NAME_PATTERN}  ${TOPIC_NAME}-.{5}
    Set Suite Variable  ${TOPIC_NAME}  ${TOPIC_NAME}-${postfix}

    ${brokers_count}=  Get Active Deployment Entities Count For Service  %{KAFKA_OS_PROJECT}  %{KAFKA_HOST}
    Set Suite Variable  ${BROKERS_COUNT}  ${brokers_count}
    Wait For Kafka Brokers Ready

    ${client_username}  ${client_password}  ${secret}=  Get Client Credentials From Kafka Secret
    Pass Execution If  '${client_username}' == '${EMPTY}' or '${client_password}' == '${EMPTY}'
    ...  Kafka client credentials are empty, password change is not applicable
    Set Suite Variable  ${CLIENT_USERNAME}  ${client_username}
    Set Suite Variable  ${ORIGINAL_CLIENT_PASSWORD}  ${client_password}
    Set Suite Variable  ${KAFKA_SECRET}  ${secret}

    # KafkaOld keeps original credentials for the whole suite (re-import with same alias is ignored by RF)
    Import Kafka Library With Credentials  ${client_username}  ${client_password}  KafkaOld
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Create Suite Admin Client
    KafkaOld.Delete Topic By Pattern  ${admin}  ${TOPIC_NAME_PATTERN}

Create Suite Admin Client
    ${admin}=  KafkaOld.Create Admin Client
    Set Suite Variable  ${admin}

Cleanup
    Run Keyword If  ${PASSWORD_RESTORED} == ${FALSE}  Restore Client Password
    Run Keyword If  '${admin}' != '${NONE}'  KafkaOld.Delete Topic By Pattern  ${admin}  ${TOPIC_NAME_PATTERN}
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
    [Arguments]  ${username}  ${password}  ${alias}
    Import Library  ../../shared/lib/KafkaLibrary.py
    ...  bootstrap_servers=${KAFKA_BOOTSTRAP_SERVERS}
    ...  namespace=%{KAFKA_OS_PROJECT}
    ...  host=${KAFKA_HOST}
    ...  port=%{KAFKA_PORT}
    ...  username=${username}
    ...  password=${password}
    ...  enable_ssl=%{KAFKA_ENABLE_SSL}
    ...  WITH NAME  ${alias}

Restore Client Password
    Patch Client Password  ${ORIGINAL_CLIENT_PASSWORD}
    Wait For Services Secret Password  ${ORIGINAL_CLIENT_PASSWORD}
    Restart Kafka Brokers
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Produce And Consume With Old Credentials
    Set Suite Variable  ${PASSWORD_RESTORED}  ${TRUE}

Wait For Services Secret Password
    [Arguments]  ${expected_password}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Services Secret Password Should Be  ${expected_password}

Services Secret Password Should Be
    [Arguments]  ${expected_password}
    ${secret}=  Get Secret  %{KAFKA_HOST}-services-secret  %{KAFKA_OS_PROJECT}
    ${password}=  Decode Secret Value  ${secret.data['client-password']}
    Should Be Equal  ${password}  ${expected_password}

Restart Kafka Brokers
    Scale Down Deployment Entities By Service Name  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}  with_check=True
    Scale Up Deployment Entities By Service Name  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}  with_check=True  replicas=1
    Wait For Kafka Brokers Ready

Kafka Brokers Are Ready
    ${ready}=  Number Of Pods In Ready Status  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}
    Should Be True  ${BROKERS_COUNT} > 0
    Should Be Equal As Integers  ${ready}  ${BROKERS_COUNT}

Wait For Kafka Brokers Ready
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Kafka Brokers Are Ready

Check Consumed Message With New Credentials
    [Arguments]  ${consumer}  ${topic_name}  ${message}
    ${received_message}=  KafkaNew.Consume Message  ${consumer}  ${topic_name}
    Should Contain  ${received_message}  ${message}

Check Consumed Message With Old Credentials
    [Arguments]  ${consumer}  ${topic_name}  ${message}
    ${received_message}=  KafkaOld.Consume Message  ${consumer}  ${topic_name}
    Should Contain  ${received_message}  ${message}

Produce And Consume With Old Credentials
    ${producer}=  KafkaOld.Create Kafka Producer
    ${message}=  KafkaOld.Create Test Message
    KafkaOld.Produce Message  ${producer}  ${TOPIC_NAME}  ${message}
    ${consumer}=  KafkaOld.Create Kafka Consumer  ${TOPIC_NAME}
    Wait Until Keyword Succeeds  ${CONSUME_MESSAGE_RETRY_COUNT}  ${CONSUME_MESSAGE_RETRY_INTERVAL}
    ...  Check Consumed Message With Old Credentials  ${consumer}  ${TOPIC_NAME}  ${message}
    KafkaOld.Close Kafka Consumer  ${consumer}
    ${producer}=  Set Variable  ${None}
    ${consumer}=  Set Variable  ${None}

Produce And Consume With New Credentials
    ${producer}=  KafkaNew.Create Kafka Producer
    ${message}=  KafkaNew.Create Test Message
    KafkaNew.Produce Message  ${producer}  ${TOPIC_NAME}  ${message}
    ${consumer}=  KafkaNew.Create Kafka Consumer  ${TOPIC_NAME}
    Wait Until Keyword Succeeds  ${CONSUME_MESSAGE_RETRY_COUNT}  ${CONSUME_MESSAGE_RETRY_INTERVAL}
    ...  Check Consumed Message With New Credentials  ${consumer}  ${TOPIC_NAME}  ${message}
    KafkaNew.Close Kafka Consumer  ${consumer}
    ${producer}=  Set Variable  ${None}
    ${consumer}=  Set Variable  ${None}

Produce With Old Credentials Should Fail
    ${producer}=  KafkaOld.Create Kafka Producer
    ${message}=  KafkaOld.Create Test Message
    Run Keyword And Expect Error  *
    ...  KafkaOld.Produce Message  ${producer}  ${TOPIC_NAME}  ${message}  retries=${1}  delay=${1}
    ${producer}=  Set Variable  ${None}

*** Test Cases ***
Test Client Password Change
    [Tags]  kafka_password_change  kafka
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  KafkaOld.Create Topic  ${admin}  ${TOPIC_NAME}  ${1}  ${1}
    Produce And Consume With Old Credentials

    ${new_password}=  Generate Random String  16  [LETTERS][NUMBERS]
    Set Suite Variable  ${PASSWORD_RESTORED}  ${FALSE}
    Patch Client Password  ${new_password}
    # Wait until operator reconciles (SCRAM sync on KRaft + services-secret update), then restart brokers ourselves
    Wait For Services Secret Password  ${new_password}
    Restart Kafka Brokers
    Import Kafka Library With Credentials  ${CLIENT_USERNAME}  ${new_password}  KafkaNew

    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Produce And Consume With New Credentials

    Produce With Old Credentials Should Fail

    [Teardown]  Run Keyword If  ${PASSWORD_RESTORED} == ${FALSE}  Restore Client Password
