*** Variables ***
${KAFKA_SERVICE_NAME}               %{KAFKA_HOST}
${KAFKA_OS_PROJECT}                 %{KAFKA_OS_PROJECT}
${KAFKA_IS_DEGRADED_ALERT_NAME}     KafkaIsDegradedAlert
${KAFKA_IS_DOWN_ALERT_NAME}         KafkaIsDownAlert
${ALERT_RETRY_TIME}                 5min
${ALERT_RETRY_INTERVAL}             10s

*** Settings ***
Library  MonitoringLibrary  host=%{PROMETHEUS_URL}
...                         username=%{PROMETHEUS_USER}
...                         password=%{PROMETHEUS_PASSWORD}
Resource  ../../shared/keywords.robot

*** Keywords ***
Check That Prometheus Alert Is Active
    [Arguments]  ${alert_name}
    ${status}=  Get Alert Status  ${alert_name}  ${KAFKA_OS_PROJECT}
    Should Be Equal As Strings  ${status}  pending

Check That Prometheus Alert Is Inactive
    [Arguments]  ${alert_name}
    ${status}=  Get Alert Status  ${alert_name}  ${KAFKA_OS_PROJECT}
    Should Be Equal As Strings  ${status}  inactive

Check That Kafka Broker Is Up
    [Arguments]  ${service_name}
    ${replicas}=  Get Active Deployment Entities Count For Service  ${KAFKA_OS_PROJECT}  ${service_name}
    Run keyword if  "${replicas}" == "0"
    ...  Scale Up Deployment Entities By Service Name  ${service_name}  ${KAFKA_OS_PROJECT}  replicas=1  with_check=True
    Sleep  30s


*** Test Cases ***
Kafka Is Degraded Alert
    [Tags]  kafka  prometheus  kafka_prometheus_alert  kafka_is_degraded_alert
    Check That Prometheus Alert Is Inactive  ${KAFKA_IS_DEGRADED_ALERT_NAME}
    ${replicas}=  Get Active Deployment Entities Count For Service  ${KAFKA_OS_PROJECT}  ${KAFKA_SERVICE_NAME}
    Pass Execution If  ${replicas} < 3  Kafka cluster has less than 3 brokers
    Scale Down Deployment Entities By Service Name  ${KAFKA_SERVICE_NAME}-1  ${KAFKA_OS_PROJECT}
    Wait Until Keyword Succeeds  ${ALERT_RETRY_TIME}  ${ALERT_RETRY_INTERVAL}
    ...  Check That Prometheus Alert Is Active  ${KAFKA_IS_DEGRADED_ALERT_NAME}
    Scale Up Deployment Entities By Service Name  ${KAFKA_SERVICE_NAME}-1  ${KAFKA_OS_PROJECT}  replicas=1
    Wait Until Keyword Succeeds  ${ALERT_RETRY_TIME}  ${ALERT_RETRY_INTERVAL}
    ...  Check That Prometheus Alert Is Inactive  ${KAFKA_IS_DEGRADED_ALERT_NAME}
    [Teardown]  Check That Kafka Broker Is Up  ${KAFKA_SERVICE_NAME}-1

Kafka Is Down Alert
    [Tags]  kafka  prometheus  kafka_prometheus_alert  kafka_is_down_alert
    Check That Prometheus Alert Is Inactive  ${KAFKA_IS_DOWN_ALERT_NAME}
    ${replicas}=  Get Active Deployment Entities Count For Service  ${KAFKA_OS_PROJECT}  ${KAFKA_SERVICE_NAME}
    Pass Execution If  ${replicas} < 3  Kafka cluster has less than 3 brokers
    Scale Down Deployment Entities By Service Name  ${KAFKA_SERVICE_NAME}  ${KAFKA_OS_PROJECT}
    Wait Until Keyword Succeeds  ${ALERT_RETRY_TIME}  ${ALERT_RETRY_INTERVAL}
    ...  Check That Prometheus Alert Is Active  ${KAFKA_IS_DOWN_ALERT_NAME}
    Scale Up Deployment Entities By Service Name  ${KAFKA_SERVICE_NAME}  ${KAFKA_OS_PROJECT}  replicas=1
    Wait Until Keyword Succeeds  ${ALERT_RETRY_TIME}  ${ALERT_RETRY_INTERVAL}
    ...  Check That Prometheus Alert Is Inactive  ${KAFKA_IS_DOWN_ALERT_NAME}
    [Teardown]  Check That Kafka Broker Is Up  ${KAFKA_SERVICE_NAME}

