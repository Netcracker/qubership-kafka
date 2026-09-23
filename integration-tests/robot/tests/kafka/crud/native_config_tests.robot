*** Variables ***
${KAFKA_NATIVE_CONFIG_KEY_1}        %{KAFKA_NATIVE_CONFIG_KEY_1=}
${KAFKA_NATIVE_CONFIG_VALUE_1}      %{KAFKA_NATIVE_CONFIG_VALUE_1=}
${KAFKA_NATIVE_CONFIG_KEY_2}        %{KAFKA_NATIVE_CONFIG_KEY_2=}
${KAFKA_NATIVE_CONFIG_VALUE_2}      %{KAFKA_NATIVE_CONFIG_VALUE_2=}
${KAFKA_NATIVE_CONFIG_CONFLICT_KEY}         %{KAFKA_NATIVE_CONFIG_CONFLICT_KEY=}
${KAFKA_NATIVE_CONFIG_CONFLICT_ENV_VALUE}   %{KAFKA_NATIVE_CONFIG_CONFLICT_ENV_VALUE=}
${SERVER_PROPERTIES_PATH}           /opt/kafka/config/server.properties
${OPERATION_RETRY_COUNT}            10
${OPERATION_RETRY_INTERVAL}         5s

*** Settings ***
Resource  ../../shared/keywords.robot

*** Keywords ***
Get Kafka Pod Name
    ${pods}=  Get Pod Names For Deployment Entity  ${KAFKA_HOST}-1  ${KAFKA_OS_PROJECT}
    ${pod}=   Get From List  ${pods}  0
    RETURN  ${pod}

Read Server Properties
    [Arguments]  ${pod}
    ${output}=  Execute Command In Pod  ${pod}  ${KAFKA_OS_PROJECT}
    ...  cat ${SERVER_PROPERTIES_PATH}  container=kafka
    RETURN  ${output}

Server Properties Contains Entry
    [Arguments]  ${pod}  ${key}  ${value}
    ${props}=  Read Server Properties  ${pod}
    Should Contain  ${props}  ${key}=${value}

*** Test Cases ***
Test Kafka Native Config Happy Path
    [Tags]  kafka_crud  kafka  native_config
    [Documentation]  Verifies that kafka.config entries appear in server.properties under their native names.
    ...  Requires environment variables KAFKA_NATIVE_CONFIG_KEY_1, KAFKA_NATIVE_CONFIG_VALUE_1,
    ...  KAFKA_NATIVE_CONFIG_KEY_2, KAFKA_NATIVE_CONFIG_VALUE_2 to be set at deploy time.
    Pass Execution If  '${KAFKA_NATIVE_CONFIG_KEY_1}' == ''
    ...  KAFKA_NATIVE_CONFIG_KEY_1 not set; skipping native config happy-path test
    ${pod}=  Get Kafka Pod Name
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Server Properties Contains Entry  ${pod}  ${KAFKA_NATIVE_CONFIG_KEY_1}  ${KAFKA_NATIVE_CONFIG_VALUE_1}
    Pass Execution If  '${KAFKA_NATIVE_CONFIG_KEY_2}' == ''
    ...  KAFKA_NATIVE_CONFIG_KEY_2 not set; second-entry check skipped
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Server Properties Contains Entry  ${pod}  ${KAFKA_NATIVE_CONFIG_KEY_2}  ${KAFKA_NATIVE_CONFIG_VALUE_2}

Test Kafka Native Config EnvironmentVariables Precedence
    [Tags]  kafka_crud  kafka  native_config
    [Documentation]  Verifies that when a property is set via both kafka.config and
    ...  kafka.environmentVariables, the environmentVariables value wins.
    ...  Requires KAFKA_NATIVE_CONFIG_CONFLICT_KEY and KAFKA_NATIVE_CONFIG_CONFLICT_ENV_VALUE
    ...  to be set at deploy time (the environmentVariables value for the conflicting key).
    Pass Execution If  '${KAFKA_NATIVE_CONFIG_CONFLICT_KEY}' == ''
    ...  KAFKA_NATIVE_CONFIG_CONFLICT_KEY not set; skipping environmentVariables precedence test
    ${pod}=  Get Kafka Pod Name
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Server Properties Contains Entry  ${pod}  ${KAFKA_NATIVE_CONFIG_CONFLICT_KEY}  ${KAFKA_NATIVE_CONFIG_CONFLICT_ENV_VALUE}
