*** Variables ***
${TOPIC_NAME}                               kafka-topic-acl-test
${CLIENT_NAME}                              kafka-integration-tests-client
${ROLE}                                     kafka
${OPERATION_RETRY_COUNT}                    30
${OPERATION_RETRY_INTERVAL}                 3s
${IDENTITY_PROVIDER_URL}                    %{IDENTITY_PROVIDER_URL}
${IDENTITY_PROVIDER_REGISTRATION_TOKEN}     %{IDENTITY_PROVIDER_REGISTRATION_TOKEN}
${IDENTITY_PROVIDER_USERNAME}               %{IDENTITY_PROVIDER_USERNAME}
${IDENTITY_PROVIDER_PASSWORD}               %{IDENTITY_PROVIDER_PASSWORD}

*** Settings ***
Library  OAuthLibrary  url=${IDENTITY_PROVIDER_URL}
...                    registration_token=${IDENTITY_PROVIDER_REGISTRATION_TOKEN}
...                    username=${IDENTITY_PROVIDER_USERNAME}
...                    password=${IDENTITY_PROVIDER_PASSWORD}
...                    grant_type=client_credentials
Resource  ../../shared/keywords.robot
Suite Setup  Setup

*** Keywords ***
Setup
    ${admin} =  Create Admin Client
    Set Suite Variable  ${admin}
    ${token_endpoint}=  Set Variable  ${IDENTITY_PROVIDER_URL}/token
    Set Suite Variable  ${token_endpoint}
    ${postfix}=  Generate Random String  5
    Set Suite Variable  ${TOPIC_NAME}  ${TOPIC_NAME}-${postfix}
    ${kafka_brokers_count}=  Get Brokers Count  ${admin}
    Set Suite Variable  ${kafka_brokers_count}
    ${client_id}  ${client_secret}=  Register New Client
    Set Suite Variable  ${client_id}
    Set Suite Variable  ${client_secret}

Cleanup
    Delete Acl  ${admin}  ${TOPIC_NAME}
    Delete Acl  ${admin}  ${TOPIC_NAME}  resource_type=group
    Delete Topic  ${admin}  ${TOPIC_NAME}
    Sleep  5s
    ${admin} =  Set Variable  ${None}
    ${oauth_producer} =  Set Variable  ${None}
    ${oauth_consumer} =  Set Variable  ${None}

Check Consumed Message
    [Arguments]  ${consumer}  ${topic_name}  ${message}
    ${receivedMessage} =  Consume Message  ${consumer}  ${topic_name}
    Should Contain  ${receivedMessage}  ${message}

Register New Client
    ${client}=  Register Client  ${CLIENT_NAME}  ${ROLE}
    [Return]  ${client['client_id']}  ${client['client_secret']}

Check ACL
    [Arguments]  ${acl_list}  ${operation}  ${type}
    Should Contain  str(${acl_list})  name=${TOPIC_NAME}  msg=ACL doesn't contain correct topic name!
    Should Contain  str(${acl_list})  operation=${operation}  msg=ACL doesn't contain correct operation!
    Should Contain  str(${acl_list})  type=${type}  msg=ACL doesn't contain correct type!

*** Test Cases ***
Test Producing And Consuming Data With Allow Type Of ACL
    [Tags]  kafka  acl  acl_all_allow
    [Teardown]  Cleanup
    Create Topic  ${admin}  ${TOPIC_NAME}  ${kafka_brokers_count}  ${kafka_brokers_count}
    Create Acl  ${admin}  ${TOPIC_NAME}
    Create Acl  ${admin}  ${TOPIC_NAME}  resource_type=group
    ${acl_list}=  Get Acls  ${admin}  ${TOPIC_NAME}
    Check ACL  ${acl_list}  ALL  ALLOW
    ${oauth_producer} =  Create Kafka Oauth Producer  ${token_endpoint}  ${client_id}  ${client_secret}
    ${message} =  Create Test Message
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Produce Message  ${oauth_producer}  ${TOPIC_NAME}  ${message}
    ${oauth_consumer} =  Create Kafka Oauth Consumer  ${TOPIC_NAME}  ${token_endpoint}  ${client_id}  ${client_secret}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Consumed Message  ${oauth_consumer}  ${TOPIC_NAME}  ${message}
    Close Kafka Consumer  ${oauth_consumer}

Test Producing And Consuming Data With Deny Type Of ACL
    [Tags]  kafka  acl  acl_read_deny
    [Teardown]  Cleanup
    Create Topic  ${admin}  ${TOPIC_NAME}  ${kafka_brokers_count}  ${kafka_brokers_count}
    Create Acl  ${admin}  ${TOPIC_NAME}  CREATE  ALLOW
    Create Acl  ${admin}  ${TOPIC_NAME}  WRITE  ALLOW
    Create Acl  ${admin}  ${TOPIC_NAME}  READ  DENY
    Create Acl  ${admin}  ${TOPIC_NAME}  resource_type=group
    ${acl_list}=  Get Acls  ${admin}  ${TOPIC_NAME}
    Check ACL  ${acl_list}  READ  DENY
    ${oauth_producer} =  Create Kafka Oauth Producer  ${token_endpoint}  ${client_id}  ${client_secret}
    ${message} =  Create Test Message
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Produce Message  ${oauth_producer}  ${TOPIC_NAME}  ${message}
    ${oauth_consumer} =  Create Kafka Oauth Consumer  ${TOPIC_NAME}  ${token_endpoint}  ${client_id}  ${client_secret}
    Sleep  15s
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Consumed Message  ${oauth_consumer}  ${TOPIC_NAME}  ${EMPTY}
    Close Kafka Consumer  ${oauth_consumer}

Test Producing And Consuming Data With Deny Type Of ACL By IP
    [Tags]  kafka  acl  acl_by_ip
    [Teardown]  Cleanup
    ${pod_names}=  Get Pod Names For Deployment Entity  ${KAFKA_HOST}-integration-tests-runner  %{KAFKA_OS_PROJECT}
    ${pod}=  Get From List  ${pod_names}  0
    ${pod_ip}=  Look Up Pod Ip By Pod Name  ${pod}  %{KAFKA_OS_PROJECT}
    Create Topic  ${admin}  ${TOPIC_NAME}  ${kafka_brokers_count}  ${kafka_brokers_count}
    Create Acl  admin=${admin}  resource_name=${TOPIC_NAME}  operation=CREATE  permission=ALLOW  host=${pod_ip}
    Create Acl  admin=${admin}  resource_name=${TOPIC_NAME}  operation=WRITE  permission=ALLOW  host=${pod_ip}
    Create Acl  admin=${admin}  resource_name=${TOPIC_NAME}  operation=READ  permission=DENY  host=${pod_ip}
    Create Acl  ${admin}  ${TOPIC_NAME}  resource_type=group
    ${acl_list}=  Get Acls  ${admin}  ${TOPIC_NAME}  ${pod_ip}
    Check ACL  ${acl_list}  READ  DENY
    ${oauth_producer} =  Create Kafka Oauth Producer  ${token_endpoint}  ${client_id}  ${client_secret}
    ${message} =  Create Test Message
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Produce Message  ${oauth_producer}  ${TOPIC_NAME}  ${message}
    ${consumer}=  Create Kafka Consumer  ${TOPIC_NAME}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Consumed Message  ${consumer}  ${TOPIC_NAME}  ${message}
    Close Kafka Consumer  ${consumer}
    ${oauth_consumer} =  Create Kafka Oauth Consumer  ${TOPIC_NAME}  ${token_endpoint}  ${client_id}  ${client_secret}
    Sleep  20s
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Consumed Message  ${oauth_consumer}  ${TOPIC_NAME}  ${EMPTY}
    Close Kafka Consumer  ${oauth_consumer}