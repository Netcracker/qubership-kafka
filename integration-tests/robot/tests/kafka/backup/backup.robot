*** Variables ***
${KAFKA_BACKUP_DAEMON_HOST}         %{BACKUP_DAEMON_HOST}
${KAFKA_BACKUP_DAEMON_PORT}         %{BACKUP_DAEMON_PORT}
${KAFKA_BACKUP_DAEMON_PROTOCOL}     %{BACKUP_DAEMON_PROTOCOL}
${KAFKA_BACKUP_DAEMON_USER}         %{BACKUP_DAEMON_USER}
${KAFKA_BACKUP_DAEMON_PASSWORD}     %{BACKUP_DAEMON_PASSWORD}
${BACKUP_RESTORE_TIMEOUT}           30s
${KAFKA_BACKUP_TOPIC}               kafka-backup-topic
${OPERATION_RETRY_COUNT}            10x
${OPERATION_RETRY_INTERVAL}         3s
${SLEEP}                            5s

*** Settings ***
Library  RequestsLibrary
Resource  ../../shared/keywords.robot
Suite Setup  Prepare

*** Keywords ***
Prepare
    ${auth}=  Create List  ${KAFKA_BACKUP_DAEMON_USER}  ${KAFKA_BACKUP_DAEMON_PASSWORD}
    ${verify}=  Set Variable If  '${KAFKA_BACKUP_DAEMON_PROTOCOL}' == 'https'  /backupTLS/ca.crt  ${True}
    Create Session  backup_daemon_session  ${KAFKA_BACKUP_DAEMON_PROTOCOL}://${KAFKA_BACKUP_DAEMON_HOST}:${KAFKA_BACKUP_DAEMON_PORT}  auth=${auth}  verify=${verify}
    ${postfix}=  Generate Random String  5
    Set Suite Variable  ${KAFKA_BACKUP_TOPIC}  ${KAFKA_BACKUP_TOPIC}-${postfix}
    ${admin}=  Create Admin Client
    Set Suite Variable  ${admin}
    ${kafka_brokers_count}=  Get Brokers Count  ${admin}
    Set Suite Variable  ${kafka_brokers_count}
    &{headers}=  Create Dictionary  Content-Type=application/json  Accept=application/json
    Set Suite Variable  ${headers}
    &{properties}=  Create Dictionary  delete.retention.ms=86400001  cleanup.policy=delete
    Set Suite Variable  ${properties}

Delete Topics
    Delete Topic  ${admin}  ${KAFKA_BACKUP_TOPIC}
    Delete Topic  ${admin}  ${KAFKA_BACKUP_TOPIC}-1
    Delete Topic  ${admin}  ${KAFKA_BACKUP_TOPIC}-2
    Sleep  ${SLEEP}

Convert Json ${json} To Type
    ${json_dictionary}=  Evaluate  json.loads('''${json}''')  json
    [Return]  ${json_dictionary}

Full Backup
    ${response}=  Post Request  backup_daemon_session  /backup
    Check Backup Status  ${response}
    [Return]  ${response.content}

Granular Backup
    ${data}=  Set Variable  {"dbs":["${KAFKA_BACKUP_TOPIC}-1","${KAFKA_BACKUP_TOPIC}-2"]}
    ${response}=  Post Request  backup_daemon_session  /backup  data=${data}  headers=${headers}
    Check Backup Status  ${response}
    [Return]  ${response.content}

Not Evictable Backup
    ${data}=  Set Variable  {"allow_eviction":"False"}
    ${response}=  Post Request  backup_daemon_session  /backup  data=${data}  headers=${headers}
    Check Backup Status  ${response}
    Check Eviction In Backup  ${response}
    [Return]  ${response.content}

Granular Backup By Regex
    ${data}=  Set Variable  {"topic_regex":"${KAFKA_BACKUP_TOPIC}-[1-9]"}
    ${response}=  Post Request  backup_daemon_session  /backup  data=${data}  headers=${headers}
    Check Backup Status  ${response}
    [Return]  ${response.content}

Check Eviction In Backup
    [Arguments]  ${backup_response}
    ${response}=  Get Request  backup_daemon_session  /listbackups/${backup_response.content}
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Strings  ${content['evictable']}  False

Check Backup Status
    [Arguments]  ${backup_response}
    Should Be Equal As Strings  ${backup_response.status_code}  200
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Backup Content  ${backup_response}

Get Backup Timestamp
    [Arguments]  ${backup_id}
    ${response}=  Get Request  backup_daemon_session  /listbackups/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content['ts']}

Get Backup ID With Timestamp
    [Arguments]  ${backup_ts}
    ${ts_dict}=  Create Dictionary  ts=${backup_ts}
    ${response}=  Get Request  backup_daemon_session  /find  json=${ts_dict}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content['id']}


Check Backup Content
    [Arguments]  ${backup_response}
    ${response}=  Get Request  backup_daemon_session  /listbackups/${backup_response.content}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Strings  ${content['failed']}  False

Full Restore
    [Arguments]  ${backup_id}
    ${restore_data}=  Set Variable  {"vault":"${backup_id}"}
    ${response}=  Post Request  backup_daemon_session  /restore  data=${restore_data}  headers=${headers}
    Check Restore Status  ${response}

Full Restore From Timestamp
    [Arguments]  ${backup_ts}
    ${restore_data}=  Set Variable  {"ts":"${backup_ts}"}
    ${response}=  Post Request  backup_daemon_session  /restore  data=${restore_data}  headers=${headers}
    Check Restore Status  ${response}

Granular Restore
    [Arguments]  ${backup_id}  ${topics_list}
    ${restore_data}=  Set Variable  {"vault":"${backup_id}","dbs":${topics_list}}
    ${response}=  Post Request  backup_daemon_session  /restore  data=${restore_data}  headers=${headers}
    Check Restore Status  ${response}

Granular Restore By Regex
    [Arguments]  ${backup_id}
    ${restore_data}=  Set Variable  {"vault":"${backup_id}","topic_regex":"${KAFKA_BACKUP_TOPIC}-[1-9]"}
    ${response}=  Post Request  backup_daemon_session  /restore  data=${restore_data}  headers=${headers}
    Check Restore Status  ${response}

Check Restore Status
    [Arguments]  ${restore_response}
    Should Be Equal As Strings  ${restore_response.status_code}  200
    Sleep  ${BACKUP_RESTORE_TIMEOUT}
    ${response}=  Get Request  backup_daemon_session  /jobstatus/${restore_response.content}
    Should Contain  str(${response.content})  Successful

Delete Backup
    [Arguments]  ${backup_id}
    ${response}=  Post Request  backup_daemon_session  /evict/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  200

Check Backup Absence In Backup Daemon
    [Arguments]  ${backup_id}
    ${response}=  Get Request  backup_daemon_session  /listbackups/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  404

Create Topic With Generated Data
    [Arguments]  ${topic_name}
    Create Topic With Custom Configuration  ${topic_name}
    ${producer} =  Create Kafka Producer
    ${message} =  Create Test Message
    Produce Message  ${producer}  ${topic_name}  ${message}
    ${consumer} =  Create Kafka Consumer  ${topic_name}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Consumed Message  ${consumer}  ${topic_name}  ${message}
    Close Kafka Consumer  ${consumer}

Create Topic With Custom Configuration
    [Arguments]  ${topic_name}  ${replication_factor}=${kafka_brokers_count}  ${partitions}=${kafka_brokers_count}  ${configs}=${properties}
    Create Topic  ${admin}  ${topic_name}  ${replication_factor}  ${partitions}  ${configs}
    Sleep  ${SLEEP}
    Check Topic Exists  ${topic_name}

Check Consumed Message
    [Arguments]  ${consumer}  ${topic_name}  ${message}
    ${receivedMessage} =  Consume Message  ${consumer}  ${topic_name}
    Should Contain  ${receivedMessage}  ${message}

Delete Data
    [Arguments]  ${topic_name}
    Delete Topic  ${admin}  ${topic_name}
    Sleep  ${SLEEP}
    ${list_topics}=  Get Topics List  ${admin}
    Should Not Contain  ${list_topics}  ${topic_name}
    Log  '${topic_name}' topic is deleted

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
    Should Be Equal  ${value}  86400001

*** Test Cases ***
Full Backup And Restore
    [Tags]  kafka  backup  full_backup  full_restore
    Create Topic With Custom Configuration  ${KAFKA_BACKUP_TOPIC}
    ${backup_id}=  Full Backup
    Delete Data  ${KAFKA_BACKUP_TOPIC}
    Full Restore  ${backup_id}
    Check Topic Exists  ${KAFKA_BACKUP_TOPIC}
    Check Topic Configuration  ${KAFKA_BACKUP_TOPIC}  ${kafka_brokers_count}
    [Teardown]  Run Keywords  Delete Topics  AND  Delete Backup  ${backup_id}

Full Backup And Restore From Timestamp
    [Tags]  kafka  backup  full_backup  full_restore
    Create Topic With Custom Configuration  ${KAFKA_BACKUP_TOPIC}
    ${backup_id}=  Full Backup
    ${backup_ts}=  Get Backup Timestamp  ${backup_id}
    Delete Data  ${KAFKA_BACKUP_TOPIC}
    Full Restore From Timestamp  ${backup_ts}
    Check Topic Exists  ${KAFKA_BACKUP_TOPIC}
    Check Topic Configuration  ${KAFKA_BACKUP_TOPIC}  ${kafka_brokers_count}
    [Teardown]  Run Keywords  Delete Topics  AND  Delete Backup  ${backup_id}

Find Backup
    [Tags]  kafka  backup  find_backup
    Create Topic With Generated Data  ${KAFKA_BACKUP_TOPIC}-1
    ${backup_id}=  Full Backup
    ${backup_ts}=  Get Backup Timestamp  ${backup_id}
    ${find_backup_id}=  Get Backup ID With Timestamp  ${backup_ts}
    Should Be Equal As Strings  ${backup_id}  ${find_backup_id}
    [Teardown]  Run Keywords  Delete Topics  AND  Delete Backup  ${backup_id}

Full Backup And Granular Restore
    [Tags]  kafka  backup  full_backup  granular_restore
    Create Topic With Custom Configuration  ${KAFKA_BACKUP_TOPIC}
    ${backup_id}=  Full Backup
    Delete Data  ${KAFKA_BACKUP_TOPIC}
    Granular Restore  ${backup_id}  ["${KAFKA_BACKUP_TOPIC}"]
    Check Topic Exists  ${KAFKA_BACKUP_TOPIC}
    Check Topic Configuration  ${KAFKA_BACKUP_TOPIC}  ${kafka_brokers_count}
    [Teardown]  Run Keywords  Delete Topics  AND  Delete Backup  ${backup_id}

Granular Backup And Restore
    [Tags]  kafka  backup  granular_backup  granular_restore
    Create Topic With Generated Data  ${KAFKA_BACKUP_TOPIC}-1
    Create Topic With Custom Configuration  ${KAFKA_BACKUP_TOPIC}-2
    ${backup_id}=  Granular Backup
    Delete Data  ${KAFKA_BACKUP_TOPIC}-1
    Delete Data  ${KAFKA_BACKUP_TOPIC}-2
    Granular Restore  ${backup_id}  ["${KAFKA_BACKUP_TOPIC}-1", "${KAFKA_BACKUP_TOPIC}-2"]
    Check Topic Exists  ${KAFKA_BACKUP_TOPIC}-1
    Check Topic Exists  ${KAFKA_BACKUP_TOPIC}-2
    Check Topic Configuration  ${KAFKA_BACKUP_TOPIC}-2  ${kafka_brokers_count}
    [Teardown]  Run Keywords  Delete Topics  AND  Delete Backup  ${backup_id}

Delete Backup By ID
    [Tags]  kafka  backup  backup_deletion
    Create Topic With Generated Data  ${KAFKA_BACKUP_TOPIC}-1
    ${backup_id}=  Full Backup
    Delete Backup  ${backup_id}
    Check Backup Absence In Backup Daemon  ${backup_id}
    [Teardown]  Delete Topics

Unauthorized Access
    [Tags]  kafka  backup  unauthorized_access
    ${verify}=  Set Variable If  '${KAFKA_BACKUP_DAEMON_PROTOCOL}' == 'https'  /backupTLS/ca.crt  ${True}
    Create Session  backup_unauthorized  ${KAFKA_BACKUP_DAEMON_PROTOCOL}://${KAFKA_BACKUP_DAEMON_HOST}:${KAFKA_BACKUP_DAEMON_PORT}  verify=${verify}
    ...  disable_warnings=1
    ${response}=  Post Request  backup_unauthorized  /backup
    Should Be Equal As Strings  ${response.status_code}  401

Not Evictable Backup
    [Tags]  kafka  backup  not_evictable
    Create Topic With Custom Configuration  ${KAFKA_BACKUP_TOPIC}
    ${backup_id}=  Not Evictable Backup
    Delete Data  ${KAFKA_BACKUP_TOPIC}
    Granular Restore  ${backup_id}  ["${KAFKA_BACKUP_TOPIC}"]
    Check Topic Exists  ${KAFKA_BACKUP_TOPIC}
    Check Topic Configuration  ${KAFKA_BACKUP_TOPIC}  ${kafka_brokers_count}
    [Teardown]  Run Keywords  Delete Topics  AND  Delete Backup  ${backup_id}

Granular Backup And Restore By Topic Regex
    [Tags]  kafka  backup  granular_backup  granular_restore  restore_by_regex
    Create Topic With Generated Data  ${KAFKA_BACKUP_TOPIC}-1
    Create Topic With Custom Configuration  ${KAFKA_BACKUP_TOPIC}-2
    ${backup_id}=  Granular Backup By Regex
    Delete Data  ${KAFKA_BACKUP_TOPIC}-1
    Delete Data  ${KAFKA_BACKUP_TOPIC}-2
    Granular Restore By Regex  ${backup_id}
    Check Topic Exists  ${KAFKA_BACKUP_TOPIC}-1
    Check Topic Exists  ${KAFKA_BACKUP_TOPIC}-2
    Check Topic Configuration  ${KAFKA_BACKUP_TOPIC}-2  ${kafka_brokers_count}
    [Teardown]  Run Keywords  Delete Topics  AND  Delete Backup  ${backup_id}
