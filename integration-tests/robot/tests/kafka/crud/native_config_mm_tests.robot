*** Variables ***
${MM_NATIVE_CONFIG_KEY_1}       %{MM_NATIVE_CONFIG_KEY_1=}
${MM_NATIVE_CONFIG_VALUE_1}     %{MM_NATIVE_CONFIG_VALUE_1=}
${MM_NATIVE_CONFIG_KEY_2}       %{MM_NATIVE_CONFIG_KEY_2=}
${MM_NATIVE_CONFIG_VALUE_2}     %{MM_NATIVE_CONFIG_VALUE_2=}
${MM_SERVICE_NAME}              %{KAFKA_HOST}-mirror-maker
${MM2_PROPERTIES_PATH}          /opt/kafka/config/mm2.properties
${OPERATION_RETRY_COUNT}        10
${OPERATION_RETRY_INTERVAL}     5s

*** Settings ***
Resource  ../../shared/keywords.robot

*** Keywords ***
Get MirrorMaker Pod Name
    ${pods}=  Get Pod Names For Deployment Entity  ${MM_SERVICE_NAME}  ${KAFKA_OS_PROJECT}
    ${pod}=   Get From List  ${pods}  0
    RETURN  ${pod}

Read MM2 Properties
    [Arguments]  ${pod}
    ${output}=  Execute Command In Pod  ${pod}  ${KAFKA_OS_PROJECT}
    ...  cat ${MM2_PROPERTIES_PATH}  container=kafka-mirror-maker
    RETURN  ${output}

MM2 Properties Contains Entry
    [Arguments]  ${pod}  ${key}  ${value}
    ${props}=  Read MM2 Properties  ${pod}
    Should Contain  ${props}  ${key} = ${value}

*** Test Cases ***
Test MirrorMaker Native Config Happy Path
    [Tags]  kafka_crud  kafka  native_config_mm
    [Documentation]  Verifies that mirrorMaker.config entries appear in mm2.properties under their native names.
    ...  Requires MM_NATIVE_CONFIG_KEY_1, MM_NATIVE_CONFIG_VALUE_1, MM_NATIVE_CONFIG_KEY_2,
    ...  MM_NATIVE_CONFIG_VALUE_2 to be set and MirrorMaker to be installed at deploy time.
    Pass Execution If  '${MM_NATIVE_CONFIG_KEY_1}' == ''
    ...  MM_NATIVE_CONFIG_KEY_1 not set; skipping MirrorMaker native config test
    ${pod}=  Get MirrorMaker Pod Name
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  MM2 Properties Contains Entry  ${pod}  ${MM_NATIVE_CONFIG_KEY_1}  ${MM_NATIVE_CONFIG_VALUE_1}
    Pass Execution If  '${MM_NATIVE_CONFIG_KEY_2}' == ''
    ...  MM_NATIVE_CONFIG_KEY_2 not set; second-entry check skipped
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  MM2 Properties Contains Entry  ${pod}  ${MM_NATIVE_CONFIG_KEY_2}  ${MM_NATIVE_CONFIG_VALUE_2}
