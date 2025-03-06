*** Variables ***
${TOPIC_NAME}                kafka-topic-tests
${OPERATION_RETRY_COUNT}     10
${OPERATION_RETRY_INTERVAL}  2s

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

Check Amount Of Partitions
    [Arguments]  ${expected_partitions_count}
    ${partitions_count} =  Get Partitions Count  ${admin}  ${TOPIC_NAME}
    Should Be Equal As Integers  ${partitions_count}  ${expected_partitions_count}

Check Existence of Topic
    ${topics} =  Get Topics List  ${admin}
    Should Contain  ${topics}  ${TOPIC_NAME}

Check Absence of Topic
    ${topics} =  Get Topics List  ${admin}
    Should Not Contain  ${topics}  ${TOPIC_NAME}

Cleanup
    Delete Topic By Pattern  ${admin}  ${TOPIC_NAME_PATTERN}
    ${admin} =  Set Variable  ${None}

*** Test Cases ***
Test Topic Creation
    [Tags]  kafka_crud  kafka
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Absence of Topic
    Create Topic  ${admin}  ${TOPIC_NAME}  ${1}  ${1}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Existence of Topic

Test Topic Change
    [Tags]  kafka_crud  kafka
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Amount Of Partitions  ${1}
    Create Partitions  ${admin}  ${TOPIC_NAME}  ${3}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Amount Of Partitions  ${3}

Test Topic Deletion
    [Tags]  kafka_crud  kafka
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Existence of Topic
    Delete Topic  ${admin}  ${TOPIC_NAME}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Absence of Topic