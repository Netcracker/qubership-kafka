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
    Set Suite Variable  ${BROKERS_COUNT}  ${0}
    ${postfix}=  Generate Random String  5
    Set Suite Variable  ${TOPIC_NAME_PATTERN}  ${TOPIC_NAME}-.{5}
    Set Suite Variable  ${TOPIC_NAME}  ${TOPIC_NAME}-${postfix}

    # Current credentials come from SecretData (services-secret mount → KAFKA_USER / KAFKA_PASSWORD)
    Pass Execution If  '${KAFKA_USER}' == '${EMPTY}' or '${KAFKA_PASSWORD}' == '${EMPTY}'
    ...  Kafka credentials are empty, password change is not applicable
    Set Suite Variable  ${CLIENT_USERNAME}  ${KAFKA_USER}
    Set Suite Variable  ${ORIGINAL_CLIENT_PASSWORD}  ${KAFKA_PASSWORD}

    ${brokers_count}=  Get Active Deployment Entities Count For Service  %{KAFKA_OS_PROJECT}  %{KAFKA_HOST}
    Set Suite Variable  ${BROKERS_COUNT}  ${brokers_count}
    Wait For Kafka Brokers Ready

    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Create Suite Admin Client
    KafkaLibrary.Delete Topic By Pattern  ${admin}  ${TOPIC_NAME_PATTERN}

Create Suite Admin Client
    ${admin}=  KafkaLibrary.Create Admin Client
    Set Suite Variable  ${admin}

Close Suite Admin Client
    Return From Keyword If  '${admin}' == '${NONE}'
    Evaluate  $admin.close()
    Set Suite Variable  ${admin}  ${NONE}

Cleanup
    Run Keyword If  ${PASSWORD_RESTORED} == ${FALSE}  Restore Client Password
    Run Keyword If  '${admin}' == '${NONE}'  Create Suite Admin Client
    Run Keyword If  '${admin}' != '${NONE}'  KafkaLibrary.Delete Topic By Pattern  ${admin}  ${TOPIC_NAME_PATTERN}
    Close Suite Admin Client

Decode Secret Value
    [Arguments]  ${value}
    ${decoded}=  Evaluate  base64.b64decode($value).decode('utf-8')  modules=base64
    RETURN  ${decoded}

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
    Close Suite Admin Client
    ${uids_before}=  Get Kafka Pod Uids
    Patch Client Password  ${ORIGINAL_CLIENT_PASSWORD}
    Wait For Operator Password Rollout  ${ORIGINAL_CLIENT_PASSWORD}  ${uids_before}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Produce And Consume With Current Credentials
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  SecretData Password Should Be  ${ORIGINAL_CLIENT_PASSWORD}
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

Get Kafka Pod Uids
    ${pods}=  Get Pods By Service Name  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}
    ${uids}=  Create List
    FOR  ${pod}  IN  @{pods}
        Append To List  ${uids}  ${pod.metadata.uid}
    END
    ${uids}=  Evaluate  tuple(sorted($uids))
    RETURN  ${uids}

Wait For Operator Password Rollout
    [Arguments]  ${expected_password}  ${uids_before}
    # services-secret update means operator reconciled (SCRAM sync done)
    Wait For Services Secret Password  ${expected_password}
    # autoRestartOnSecretChange should recreate broker pods
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Kafka Pods Recreated  ${uids_before}
    Wait For Kafka Brokers Ready

Kafka Pods Recreated
    [Arguments]  ${uids_before}
    ${uids_after}=  Get Kafka Pod Uids
    Should Not Be Equal  ${uids_before}  ${uids_after}

Kafka Brokers Are Ready
    ${ready}=  Number Of Pods In Ready Status  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}
    Should Be True  ${BROKERS_COUNT} > 0
    Should Be Equal As Integers  ${ready}  ${BROKERS_COUNT}

Wait For Kafka Brokers Ready
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Kafka Brokers Are Ready

Check Consumed Message
    [Arguments]  ${consumer}  ${topic_name}  ${message}
    ${received_message}=  KafkaLibrary.Consume Message  ${consumer}  ${topic_name}
    Should Contain  ${received_message}  ${message}

Check Consumed Message With New Credentials
    [Arguments]  ${consumer}  ${topic_name}  ${message}
    ${received_message}=  KafkaNew.Consume Message  ${consumer}  ${topic_name}
    Should Contain  ${received_message}  ${message}

Produce And Consume With Current Credentials
    ${producer}=  KafkaLibrary.Create Kafka Producer
    ${message}=  KafkaLibrary.Create Test Message
    KafkaLibrary.Produce Message  ${producer}  ${TOPIC_NAME}  ${message}
    ${consumer}=  KafkaLibrary.Create Kafka Consumer  ${TOPIC_NAME}
    Wait Until Keyword Succeeds  ${CONSUME_MESSAGE_RETRY_COUNT}  ${CONSUME_MESSAGE_RETRY_INTERVAL}
    ...  Check Consumed Message  ${consumer}  ${TOPIC_NAME}  ${message}
    KafkaLibrary.Close Kafka Consumer  ${consumer}
    Evaluate  $producer.close()

SecretData Password Should Be
    [Arguments]  ${expected_password}
    Import Variables  %{ROBOT_HOME}/SecretData.py
    Should Be Equal  ${KAFKA_PASSWORD}  ${expected_password}

Produce And Consume With New Credentials
    ${producer}=  KafkaNew.Create Kafka Producer
    ${message}=  KafkaNew.Create Test Message
    KafkaNew.Produce Message  ${producer}  ${TOPIC_NAME}  ${message}
    ${consumer}=  KafkaNew.Create Kafka Consumer  ${TOPIC_NAME}
    Wait Until Keyword Succeeds  ${CONSUME_MESSAGE_RETRY_COUNT}  ${CONSUME_MESSAGE_RETRY_INTERVAL}
    ...  Check Consumed Message With New Credentials  ${consumer}  ${TOPIC_NAME}  ${message}
    KafkaNew.Close Kafka Consumer  ${consumer}
    Evaluate  $producer.close()

Produce With Current Credentials Should Fail
    ${producer}=  KafkaLibrary.Create Kafka Producer
    ${message}=  KafkaLibrary.Create Test Message
    Run Keyword And Expect Error  *
    ...  KafkaLibrary.Produce Message  ${producer}  ${TOPIC_NAME}  ${message}  retries=${1}  delay=${1}
    Evaluate  $producer.close()

*** Test Cases ***
Test Client Password Change
    [Tags]  kafka_password_change  kafka
    # Baseline with SecretData credentials (KAFKA_USER / KAFKA_PASSWORD from services-secret)
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  KafkaLibrary.Create Topic  ${admin}  ${TOPIC_NAME}  ${1}  ${1}
    Produce And Consume With Current Credentials

    Close Suite Admin Client
    ${uids_before}=  Get Kafka Pod Uids
    ${new_password}=  Generate Random String  16  [LETTERS][NUMBERS]
    Set Suite Variable  ${PASSWORD_RESTORED}  ${FALSE}
    Patch Client Password  ${new_password}
    Wait For Operator Password Rollout  ${new_password}  ${uids_before}
    Import Kafka Library With Credentials  ${CLIENT_USERNAME}  ${new_password}  KafkaNew

    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Produce And Consume With New Credentials

    # KafkaLibrary still has suite-start SecretData password (= original)
    Produce With Current Credentials Should Fail

    # Always put the original password back so the cluster is left as before the test
    Restore Client Password

    [Teardown]  Run Keyword If  ${PASSWORD_RESTORED} == ${FALSE}  Restore Client Password
