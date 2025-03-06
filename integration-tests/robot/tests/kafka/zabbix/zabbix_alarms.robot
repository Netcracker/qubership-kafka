*** Variables ***
${HOST_GROUP_NAME}                 Kafka group
${TEMPLATE_NAME}                   Kafka template
${TEMPLATE_VISIBLE_NAME}           Kafka visible name
${APPLICATION_NAME}                Kafka application
${HOST_NAME}                       Kafka host name
${EXPORT_FILE_PATH}                tests/kafka/zabbix/resources/template_export.xml
${KAFKA_INTERVAL}                  5m
${SNMP_COMMUNITY}                  public
${ALARM_RETRY_TIME}                5min
${ALARM_RETRY_INTERVAL}            10s

*** Settings ***
Library  ../../shared/lib/ZabbixLibrary.py  url=%{ZABBIX_URL}
...                                         user=%{ZABBIX_USER}
...                                         password=%{ZABBIX_PASSWORD}
Library  PlatformLibrary  managed_by_operator=%{KAFKA_IS_MANAGED_BY_OPERATOR}
Suite Setup  Run Keywords  Clean Up
...                        Set Prerequisites
Suite Teardown  Clean Up

*** Keywords ***
Set Prerequisites
    Load Template  ${HOST_GROUP_NAME}
    ...  ${TEMPLATE_NAME}
    ...  ${TEMPLATE_VISIBLE_NAME}
    ...  ${APPLICATION_NAME}
    ...  ${EXPORT_FILE_PATH}
    Create Host  ${HOST_NAME}  ${HOST_GROUP_NAME}  ${TEMPLATE_NAME}
    Add Macros To Host  ${HOST_NAME}
    ...  KAFKA_INTERVAL=${KAFKA_INTERVAL}
    ...  KAFKA_PROJECT_NAME=%{KAFKA_OS_PROJECT}
    ...  KAFKA_PV_NAMES=%{KAFKA_PV_NAMES}
    ...  CLOUD_DB=%{CLOUD_DB}
    ...  DR_SIDE=%{DR_SIDE}
    ...  SNMP_COMMUNITY=${SNMP_COMMUNITY}
    ${all_deployments}=  Get All Deployment Entity Names
    Set Suite Variable  ${all_deployments}

Clean Up
    Delete Host By Name  ${HOST_NAME}
    Delete Template By Name  ${TEMPLATE_NAME}
    Delete Host Group  ${HOST_GROUP_NAME}

Check That Zabbix Problem Exists
    [Arguments]  ${problem_name}
    ${boolean_result}=  Is Zabbix Problem Exist  ${HOST_NAME}  ${problem_name}
    Should Be True  ${boolean_result}

Check That Zabbix Does Not Exist
    [Arguments]  ${problem_name}
    ${boolean_result}=  Is Zabbix Problem Exist  ${HOST_NAME}  ${problem_name}
    Should Not be True  ${boolean_result}

Get All Deployment Entity Names
    ${all_deployments}=  Get Deployment Entity Names For Service  %{KAFKA_OS_PROJECT}  %{KAFKA_HOST}
    [Return]  ${all_deployments}

Scale Up Kafka If It Is Down
    ${is_up}=  Check Service Is Scaled  ${all_deployments}  %{KAFKA_OS_PROJECT}  direction=up  timeout=30
    Run Keyword If  ${is_up} == ${TRUE}  Pass Execution  Kafka service is full up

    ${inaсtive_deployments}=  Get Inactive Deployment Entities Names For Service  %{KAFKA_OS_PROJECT}  %{KAFKA_HOST}
    FOR  ${deployment}  IN  @{inaсtive_deployments}
      Set Replicas For Deployment Entity  ${deployment}  %{KAFKA_OS_PROJECT}  ${1}
    END

Skip Test If Kafka Deployment Entity Is Down
     ${is_down}=  Check Service Is Scaled  ${all_deployments}  %{KAFKA_OS_PROJECT}  direction=down  timeout=30
     Run Keyword If  ${is_down}  Pass Execution  This test is skipped because Kafka service is not full up

*** Test Cases ***
Kafka Is Degraded Alarm
    [Tags]  zabbix  kafka_zabbix  kafka  kafka_is_degraded_alarm
    Skip Test If Kafka Deployment Entity Is Down
    ${deployment}=  Get First Deployment Entity Name For Service  %{KAFKA_OS_PROJECT}  %{KAFKA_HOST}
    Scale Down Deployment Entity  ${deployment}  %{KAFKA_OS_PROJECT}
    Wait Until Keyword Succeeds  ${ALARM_RETRY_TIME}  ${ALARM_RETRY_INTERVAL}
    ...  Check That Zabbix Problem Exists  Kafka is Degraded on ${HOST_NAME}
    Set Replicas For Deployment Entity  ${deployment}  %{KAFKA_OS_PROJECT}  ${1}
    Wait Until Keyword Succeeds  ${ALARM_RETRY_TIME}  ${ALARM_RETRY_INTERVAL}
    ...  Check That Zabbix Does Not Exist  Kafka is Degraded on ${HOST_NAME}
    [Teardown]  Scale Up Kafka If It Is Down

Kafka Is Down Alarm
    [Tags]  zabbix  kafka_zabbix  kafka  kafka_is_down_alarm
    Skip Test If Kafka Deployment Entity Is Down
    Scale Down Deployment Entities By Service Name  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}  with_check=True
    Wait Until Keyword Succeeds  ${ALARM_RETRY_TIME}  ${ALARM_RETRY_INTERVAL}
    ...  Check That Zabbix Problem Exists  Kafka is Down on ${HOST_NAME}
    Scale Up Deployment Entities By Service Name  %{KAFKA_HOST}  %{KAFKA_OS_PROJECT}  with_check=True  replicas=1
    Wait Until Keyword Succeeds  ${ALARM_RETRY_TIME}  ${ALARM_RETRY_INTERVAL}
    ...  Check That Zabbix Does Not Exist  Kafka is Down on ${HOST_NAME}
    [Teardown]  Scale Up Kafka If It Is Down