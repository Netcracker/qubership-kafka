*** Variables ***
${KAFKA_BACKUP_DAEMON_HOST}                 %{BACKUP_DAEMON_HOST}
${KAFKA_BACKUP_DAEMON_PORT}                 %{BACKUP_DAEMON_PORT}
${KAFKA_BACKUP_DAEMON_USER}                 %{BACKUP_DAEMON_USER}
${KAFKA_BACKUP_DAEMON_PASSWORD}             %{BACKUP_DAEMON_PASSWORD}
${KAFKA_BACKUP_DAEMON_PROTOCOL}             %{BACKUP_DAEMON_PROTOCOL}
${IDENTITY_PROVIDER_URL}                    %{IDENTITY_PROVIDER_URL}
${IDENTITY_PROVIDER_REGISTRATION_TOKEN}     %{IDENTITY_PROVIDER_REGISTRATION_TOKEN}
${IDENTITY_PROVIDER_USERNAME}               %{IDENTITY_PROVIDER_USERNAME}
${IDENTITY_PROVIDER_PASSWORD}               %{IDENTITY_PROVIDER_PASSWORD}
${BACKUP_RESTORE_TIMEOUT}                   20s
${KAFKA_BACKUP_TOPIC}                       kafka-backup-acl-topic
${OPERATION_RETRY_COUNT}                    30
${OPERATION_RETRY_INTERVAL}                 3s
${CLIENT_NAME}                              kafka-integration-tests-client
${ROLE}                                     kafka


*** Settings ***
Library  RequestsLibrary
Library  OAuthLibrary  url=${IDENTITY_PROVIDER_URL}
...                    registration_token=${IDENTITY_PROVIDER_REGISTRATION_TOKEN}
...                    username=${IDENTITY_PROVIDER_USERNAME}
...                    password=${IDENTITY_PROVIDER_PASSWORD}
...                    grant_type=client_credentials
Resource  ../../shared/keywords.robot
Suite Setup  Prepare

*** Keywords ***
Prepare
    ${auth}=  Create List  ${KAFKA_BACKUP_DAEMON_USER}  ${KAFKA_BACKUP_DAEMON_PASSWORD}
    ${verify}=  Set Variable If  '${KAFKA_BACKUP_DAEMON_PROTOCOL}' == 'https'  /backupTLS/ca.crt  ${True}
    Create Session  backup_daemon_session  ${KAFKA_BACKUP_DAEMON_PROTOCOL}://${KAFKA_BACKUP_DAEMON_HOST}:${KAFKA_BACKUP_DAEMON_PORT}  auth=${auth}  verify=${verify}
    ${postfix}=  Generate Random String  5
    Set Suite Variable  ${KAFKA_BACKUP_ACL_TOPIC}  ${KAFKA_BACKUP_TOPIC}-${postfix}
    ${admin}=  Create Admin Client
    Set Suite Variable  ${admin}
    ${kafka_brokers_count}=  Get Brokers Count  ${admin}
    Set Suite Variable  ${kafka_brokers_count}
    &{headers}=  Create Dictionary  Content-Type=application/json  Accept=application/json
    Set Suite Variable  ${headers}
    &{properties}=  Create Dictionary  delete.retention.ms=86400002  cleanup.policy=delete
    Set Suite Variable  ${properties}
    ${token_endpoint}=  Set Variable  ${IDENTITY_PROVIDER_URL}/token
    Set Suite Variable  ${token_endpoint}
    ${client_id}  ${client_secret}=  Register New Client
    Set Suite Variable  ${client_id}
    Set Suite Variable  ${client_secret}

Cleanup ACL And Topics
    Delete Acl  ${admin}  ${KAFKA_BACKUP_ACL_TOPIC}
    Delete Acl  ${admin}  ${KAFKA_BACKUP_ACL_TOPIC}-2
    Delete Topics

Register New Client
    ${client}=  Register Client  ${CLIENT_NAME}  ${ROLE}
    [Return]  ${client['client_id']}  ${client['client_secret']}

Check ACL
    [Arguments]  ${acl_list}  ${operation}  ${type}
    Should Contain  str(${acl_list})  name=${KAFKA_BACKUP_ACL_TOPIC}  msg=ACL doesn't contain correct topic name!
    Should Contain  str(${acl_list})  operation=${operation}  msg=ACL doesn't contain correct operation!
    Should Contain  str(${acl_list})  type=${type}  msg=ACL doesn't contain correct type!

Create Topic With Custom Configuration
    [Arguments]  ${topic_name}  ${replication_factor}=${kafka_brokers_count}  ${partitions}=${kafka_brokers_count}  ${configs}=${properties}
    Create Topic  ${admin}  ${topic_name}  ${replication_factor}  ${partitions}  ${configs}
    Sleep  5s
    Check Topic Exists  ${topic_name}

Check Topic Exists
    [Arguments]  ${topic_name}
    ${list_topics}=  Get Topics List  ${admin}
    Should Contain  ${list_topics}  ${topic_name}

Check Topic Configuration
    [Arguments]  ${topic_name}  ${partions_count}
    ${count}=  Get Partitions Count  ${admin}  ${topic_name}
    Should Be Equal  ${partions_count}  ${count}
    ${value}=  Get Value Of Topic Config  ${admin}  ${topic_name}  cleanup.policy
    Should Be Equal As Strings  ${value}  delete
    ${value}=  Get Value Of Topic Config  ${admin}  ${topic_name}  delete.retention.ms
    Should Be Equal  ${value}  86400002

Cleanup
    Cleanup ACL And Topics
    ${admin} =  Set Variable  ${None}

Delete Topics
    Delete Topic  ${admin}  ${KAFKA_BACKUP_ACL_TOPIC}
    Delete Topic  ${admin}  ${KAFKA_BACKUP_ACL_TOPIC}-2
    Sleep  5s

Backup ACL
    ${data}=  Set Variable  {"mode":"acl"}
    ${response}=  Post Request  backup_daemon_session  /backup  data=${data}  headers=${headers}
    Check Backup Status  ${response}
    [Return]  ${response.content}

Check Backup Status
    [Arguments]  ${backup_response}
    Should Be Equal As Strings  ${backup_response.status_code}  200
    Sleep  ${BACKUP_RESTORE_TIMEOUT}
    ${response}=  Get Request  backup_daemon_session  /listbackups/${backup_response.content}
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Strings  ${content['failed']}  False

Granular Restore With ACL
    [Arguments]  ${backup_id}
    ${restore_data}=  Set Variable  {"vault":"${backup_id}", "mode":"acl", "dbs":["${KAFKA_BACKUP_ACL_TOPIC}","${KAFKA_BACKUP_ACL_TOPIC}-2"]}
    ${response}=  Post Request  backup_daemon_session  /restore  data=${restore_data}  headers=${headers}
    Check Restore Status  ${response}

Check Restore Status
    [Arguments]  ${restore_response}
    Should Be Equal As Strings  ${restore_response.status_code}  200
    Sleep  ${BACKUP_RESTORE_TIMEOUT}
    ${response}=  Get Request  backup_daemon_session  /jobstatus/${restore_response.content}
    Should Contain  str(${response.content})  Successful

Convert Json ${json} To Type
    ${json_dictionary}=  Evaluate  json.loads('''${json}''')  json
    [Return]  ${json_dictionary}

Check All ACL For Topics
    ${acl_list}=  Get Acls  ${admin}  ${KAFKA_BACKUP_ACL_TOPIC}
    Check ACL  ${acl_list}  CREATE  ALLOW
    Check ACL  ${acl_list}  WRITE  ALLOW
    Check ACL  ${acl_list}  READ  DENY
    ${acl_list}=  Get Acls  ${admin}  ${KAFKA_BACKUP_ACL_TOPIC}-2
    Check ACL  ${acl_list}  WRITE  DENY

*** Test Cases ***
Full Backup And Restore ACL
    [Tags]  kafka  acl_backup
    Create Topic With Custom Configuration  ${KAFKA_BACKUP_ACL_TOPIC}
    Create Topic With Custom Configuration  ${KAFKA_BACKUP_ACL_TOPIC}-2
    Create Acl  ${admin}  ${KAFKA_BACKUP_ACL_TOPIC}  CREATE  ALLOW
    Create Acl  ${admin}  ${KAFKA_BACKUP_ACL_TOPIC}  WRITE  ALLOW
    Create Acl  ${admin}  ${KAFKA_BACKUP_ACL_TOPIC}  READ  DENY
    Create Acl  ${admin}  ${KAFKA_BACKUP_ACL_TOPIC}-2  WRITE  DENY
    Check All ACL For Topics
    ${backup_id}=  Backup ACL
    Cleanup ACL And Topics
    Granular Restore With ACL  ${backup_id}
    Sleep  10s
    Check Topic Exists  ${KAFKA_BACKUP_ACL_TOPIC}
    Check Topic Exists  ${KAFKA_BACKUP_ACL_TOPIC}-2
    Check Topic Configuration  ${KAFKA_BACKUP_ACL_TOPIC}-2  ${kafka_brokers_count}
    Check All ACL For Topics
    [Teardown]  Cleanup
