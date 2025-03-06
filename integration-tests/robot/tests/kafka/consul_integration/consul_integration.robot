*** Variables ***
${KAFKA_SERVICE_NAME}           %{KAFKA_HOST}
${KAFKA_OS_PROJECT}             %{KAFKA_OS_PROJECT}
${CONSUL_HOST}                  %{CONSUL_HOST}
${CONSUL_PORT}                  %{CONSUL_PORT}
${CONSUL_TOKEN}                 %{CONSUL_TOKEN}
${OPERATION_RETRY_COUNT}        300
${OPERATION_RETRY_INTERVAL}     5s


*** Settings ***
Library  RequestsLibrary
Resource  ../../shared/keywords.robot
Suite Setup  Prepare

*** Keywords ***
Prepare
    &{headers}=  Create Dictionary  Content-Type=application/json  Accept=application/json  X-Consul-Token=${CONSUL_TOKEN}
    Set Suite Variable  ${headers}
    Create Session  consul_session  http://${CONSUL_HOST}:${CONSUL_PORT}
    ${kafka_brokers}=  Get Active Deployment Entities Count For Service  ${KAFKA_OS_PROJECT}  ${KAFKA_SERVICE_NAME}
    Set Suite Variable  ${kafka_brokers}

Convert Json ${json} To Type
    ${json_dictionary}=  Evaluate  json.loads('''${json}''')  json
    [Return]  ${json_dictionary}

Check List Services In Consul
    ${response}=  Get Request  consul_session  /v1/catalog/service/${KAFKA_SERVICE_NAME}-${KAFKA_OS_PROJECT}
    ${content}=  Convert Json ${response.content} To Type
    ${length} =	Get Length  ${content}
    Should Be True  ${length} > ${kafka_brokers}-1

Check Health Status
    [Arguments]  ${kafka_broker}  ${status}
    ${response}=  Get Request  consul_session  /v1/health/checks/${KAFKA_SERVICE_NAME}-${KAFKA_OS_PROJECT}
    ${content}=  Convert Json ${response.content} To Type
    FOR  ${var}  IN  @{content}
    Run keyword if  "${var['ServiceID']}" == "${kafka_broker}"
    ...  Should Be Equal As Strings  ${var['Status']}  ${status}
    END

Check That Kafka Broker Is Up
    [Arguments]  ${service_name}
    ${replicas}=  Get Active Deployment Entities Count For Service  ${KAFKA_OS_PROJECT}  ${service_name}
    Run keyword if  "${replicas}" == "0"
    ...  Scale Up Deployment Entities By Service Name  ${service_name}  ${KAFKA_OS_PROJECT}  replicas=1
    Sleep  20s


*** Test Cases ***
Check Kafka Status In Consul
    [Tags]  kafka  kafka_status_in_consul
    Check List Services In Consul
    Scale Down Deployment Entities By Service Name  ${KAFKA_SERVICE_NAME}-1  ${KAFKA_OS_PROJECT}  replicas=0
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Health Status  ${KAFKA_SERVICE_NAME}-1  critical
    Scale Up Deployment Entities By Service Name  ${KAFKA_SERVICE_NAME}-1  ${KAFKA_OS_PROJECT}  replicas=1
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Run Keywords
    ...  Check List Services In Consul  AND
    ...  Check Health Status  ${KAFKA_SERVICE_NAME}-1  passing
    [Teardown]  Check That Kafka Broker Is Up  ${KAFKA_SERVICE_NAME}-1