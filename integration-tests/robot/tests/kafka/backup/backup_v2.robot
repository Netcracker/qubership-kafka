*** Variables ***
${KAFKA_BACKUP_DAEMON_HOST}          %{BACKUP_DAEMON_HOST}
${KAFKA_BACKUP_DAEMON_PORT}          %{BACKUP_DAEMON_PORT}
${KAFKA_BACKUP_DAEMON_PROTOCOL}      %{BACKUP_DAEMON_PROTOCOL}
${KAFKA_BACKUP_DAEMON_USER}          %{BACKUP_DAEMON_USER}
${KAFKA_BACKUP_DAEMON_PASSWORD}      %{BACKUP_DAEMON_PASSWORD}
${KAFKA_OS_PROJECT}                  %{KAFKA_OS_PROJECT}
${KAFKA_BACKUP_V2_TOPIC}             kafka_backup_v2_topic
${BACKUP_STORAGE_NAME}               s3
${BACKUP_BLOB_PATH}                  /backup-storage/v2/kafka
${BACKUP_BLOB_PATH_ALIAS_TEST}       /backup-storage/v2-alias-default/kafka
${S3_ALIASES_SECRET_NAME}            %{S3_ALIASES_SECRET_NAME=kafka-backup-daemon-s3-aliases}
${S3_DEFAULT_ALIAS_NAME}             %{S3_DEFAULT_ALIAS_NAME=default}
${S3_KEY_SECRET}                     %{S3_KEY_SECRET}
${RETRY_TIME}                        300s
${RETRY_INTERVAL}                    10s
${SLEEP}                             5s

*** Settings ***
Library  RequestsLibrary
Library  OperatingSystem
Resource  ../../shared/keywords.robot
Suite Setup  Prepare

*** Keywords ***
Prepare
    ${auth}=  Create List  ${KAFKA_BACKUP_DAEMON_USER}  ${KAFKA_BACKUP_DAEMON_PASSWORD}
    ${verify}=  Set Variable If  '${KAFKA_BACKUP_DAEMON_PROTOCOL}' == 'https'  /backupTLS/ca.crt  ${True}
    Create Session  backup_daemon_v2_session  ${KAFKA_BACKUP_DAEMON_PROTOCOL}://${KAFKA_BACKUP_DAEMON_HOST}:${KAFKA_BACKUP_DAEMON_PORT}  auth=${auth}  verify=${verify}
    ${admin}=  Create Admin Client
    Set Suite Variable  ${admin}
    ${kafka_brokers_count}=  Get Brokers Count  ${admin}
    Set Suite Variable  ${kafka_brokers_count}
    &{headers}=  Create Dictionary  Content-Type=application/json  Accept=application/json
    Set Suite Variable  ${headers}
    Delete Data  ${KAFKA_BACKUP_V2_TOPIC}

Convert Json ${json} To Type
    ${json_dictionary}=  Evaluate  json.loads('''${json}''')  json
    [Return]  ${json_dictionary}

Get Track Id
    [Arguments]  ${response_content}
    ${content}=  Convert Json ${response_content} To Type
    ${track_id}=  Evaluate  $content.get('backupId') or $content.get('restoreId') or $content.get('trackId') or $content.get('id')
    Should Not Be Equal  ${track_id}  ${None}
    RETURN  ${track_id}

Create Backup V2
    [Arguments]  ${topic_name}  ${blob_path}=${BACKUP_BLOB_PATH}
    ${storage_name}=  Get Backup Storage Name
    ${data}=  Set Variable  {"storageName":"${storage_name}","blobPath":"${blob_path}","databases":["${topic_name}"]}
    ${response}=  Post Request  backup_daemon_v2_session  /api/v1/backup  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${backup_id}=  Get Track Id  ${response.content}
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Backup Status V2  ${backup_id}
    RETURN  ${backup_id}

Check Backup Status V2
    [Arguments]  ${backup_id}
    ${response}=  Get Request  backup_daemon_v2_session  /api/v1/backup/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    ${status}=  Evaluate  str($content.get("status", "")).lower()
    Should Be Equal As Strings  ${status}  completed

Restore Backup V2
    [Arguments]  ${backup_id}  ${topic_name}  ${blob_path}=${BACKUP_BLOB_PATH}
    ${storage_name}=  Get Backup Storage Name
    ${data}=  Set Variable  {"storageName":"${storage_name}","blobPath":"${blob_path}","databases":[{"previousDatabaseName":"${topic_name}","databaseName":"${topic_name}"}],"dryRun":false}
    ${response}=  Post Request  backup_daemon_v2_session  /api/v1/restore/${backup_id}  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${restore_id}=  Get Track Id  ${response.content}
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Restore Status V2  ${restore_id}
    RETURN  ${restore_id}

Check Restore Status V2
    [Arguments]  ${restore_id}
    ${response}=  Get Request  backup_daemon_v2_session  /api/v1/restore/${restore_id}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    ${status}=  Evaluate  str($content.get("status", "")).lower()
    Should Be Equal As Strings  ${status}  completed

Delete Backup V2
    [Arguments]  ${backup_id}  ${blob_path}=${BACKUP_BLOB_PATH}
    ${response}=  Delete Request  backup_daemon_v2_session  /api/v1/backup/${backup_id}?blobPath=${blob_path}
    Should Be Equal As Strings  ${response.status_code}  200

Delete Restore V2 If Exists
    [Arguments]  ${restore_id}  ${blob_path}=${BACKUP_BLOB_PATH}
    Run Keyword If  "${restore_id}" != "${None}"  Run Keyword And Ignore Error  Delete Request  backup_daemon_v2_session  /api/v1/restore/${restore_id}?blobPath=${blob_path}

Delete Backup V2 If Exists
    [Arguments]  ${backup_id}  ${blob_path}=${BACKUP_BLOB_PATH}
    Run Keyword If  "${backup_id}" != "${None}"  Run Keyword And Ignore Error  Delete Backup V2  ${backup_id}  ${blob_path}

Ensure S3 Aliases Config Available
    ${secret_exists}=  Run Keyword And Return Status  Check Secret  ${S3_ALIASES_SECRET_NAME}  ${KAFKA_OS_PROJECT}
    Pass Execution If  not ${secret_exists}  S3 aliases secret is absent, skip alias routing test
    ${secret}=  Check Secret  ${S3_ALIASES_SECRET_NAME}  ${KAFKA_OS_PROJECT}
    ${has_alias_config}=  Evaluate  bool($secret.data) and 's3_aliases.json' in $secret.data and bool($secret.data['s3_aliases.json'])
    Pass Execution If  not ${has_alias_config}  S3 aliases config is empty, skip alias routing test

Get Default S3 Alias Config
    ${secret}=  Check Secret  ${S3_ALIASES_SECRET_NAME}  ${KAFKA_OS_PROJECT}
    ${aliases_base64}=  Set Variable  ${secret.data['s3_aliases.json']}
    ${aliases_json}=  Evaluate  base64.b64decode($aliases_base64).decode("utf-8")  modules=base64
    ${aliases}=  Convert Json ${aliases_json} To Type
    ${default_alias_name}=  Evaluate  next((name for name, cfg in $aliases.items() if cfg.get("default") is True), None)
    ${default_alias_name}=  Run Keyword If  "${default_alias_name}" == "${None}"  Set Variable  ${S3_DEFAULT_ALIAS_NAME}  ELSE  Set Variable  ${default_alias_name}
    ${default_alias}=  Evaluate  $aliases.get($default_alias_name)
    Should Not Be Equal  ${default_alias}  ${None}
    RETURN  ${default_alias}

Get Backup Storage Name
    ${secret_exists}=  Run Keyword And Return Status  Check Secret  ${S3_ALIASES_SECRET_NAME}  ${KAFKA_OS_PROJECT}
    Run Keyword If  not ${secret_exists}  Return From Keyword  ${BACKUP_STORAGE_NAME}
    ${secret}=  Check Secret  ${S3_ALIASES_SECRET_NAME}  ${KAFKA_OS_PROJECT}
    ${has_alias_config}=  Evaluate  bool($secret.data) and 's3_aliases.json' in $secret.data and bool($secret.data['s3_aliases.json'])
    Run Keyword If  not ${has_alias_config}  Return From Keyword  ${BACKUP_STORAGE_NAME}
    ${aliases_base64}=  Set Variable  ${secret.data['s3_aliases.json']}
    ${aliases_json}=  Evaluate  base64.b64decode($aliases_base64).decode("utf-8")  modules=base64
    ${aliases}=  Convert Json ${aliases_json} To Type
    ${default_alias_name}=  Evaluate  next((name for name, cfg in $aliases.items() if cfg.get("default") is True), None)
    ${storage_name}=  Run Keyword If  "${default_alias_name}" == "${None}"  Set Variable  ${S3_DEFAULT_ALIAS_NAME}  ELSE  Set Variable  ${default_alias_name}
    RETURN  ${storage_name}

Check Backup Exists In Default Alias Bucket
    [Arguments]  ${backup_id}  ${blob_path}=${BACKUP_BLOB_PATH}
    ${default_alias}=  Get Default S3 Alias Config
    ${s3_url}=  Evaluate  $default_alias.get("s3Url") or $default_alias.get("storageServerUrl")
    ${s3_bucket}=  Evaluate  $default_alias.get("bucketName") or $default_alias.get("storageBucket")
    ${s3_key_id}=  Evaluate  $default_alias.get("accessKeyId") or $default_alias.get("storageUsername")
    ${s3_key_secret}=  Evaluate  $default_alias.get("accessKeySecret") or $S3_KEY_SECRET
    Should Not Be Empty  ${s3_url}
    Should Not Be Empty  ${s3_bucket}
    Should Not Be Empty  ${s3_key_id}
    Should Not Be Empty  ${s3_key_secret}
    Import Library  S3BackupLibrary  url=${s3_url}  bucket=${s3_bucket}  key_id=${s3_key_id}  key_secret=${s3_key_secret}  WITH NAME  DefaultAliasS3
    ${backup_file_exist}=  DefaultAliasS3.Check Backup Exists  path=${blob_path}  backup_id=${backup_id}
    Should Be True  ${backup_file_exist}

Check Backup Does Not Exist In Default Alias Bucket
    [Arguments]  ${backup_id}  ${blob_path}
    ${default_alias}=  Get Default S3 Alias Config
    ${s3_url}=  Evaluate  $default_alias.get("s3Url") or $default_alias.get("storageServerUrl")
    ${s3_bucket}=  Evaluate  $default_alias.get("bucketName") or $default_alias.get("storageBucket")
    ${s3_key_id}=  Evaluate  $default_alias.get("accessKeyId") or $default_alias.get("storageUsername")
    ${s3_key_secret}=  Evaluate  $default_alias.get("accessKeySecret") or $S3_KEY_SECRET
    Import Library  S3BackupLibrary  url=${s3_url}  bucket=${s3_bucket}  key_id=${s3_key_id}  key_secret=${s3_key_secret}  WITH NAME  DefaultAliasS3
    ${backup_file_exist}=  DefaultAliasS3.Check Backup Exists  path=${blob_path}  backup_id=${backup_id}
    Should Not Be True  ${backup_file_exist}

Create Topic With Generated Data
    [Arguments]  ${topic_name}
    Create Topic  ${admin}  ${topic_name}  ${kafka_brokers_count}  ${kafka_brokers_count}
    Sleep  ${SLEEP}
    ${producer}=  Create Kafka Producer
    ${message}=  Create Test Message
    Produce Message  ${producer}  ${topic_name}  ${message}
    ${consumer}=  Create Kafka Consumer  ${topic_name}
    Wait Until Keyword Succeeds  10x  3s
    ...  Check Consumed Message  ${consumer}  ${topic_name}  ${message}
    Close Kafka Consumer  ${consumer}

Delete Data
    [Arguments]  ${topic_name}
    Delete Topic  ${admin}  ${topic_name}
    Sleep  ${SLEEP}

Check Topic Exists
    [Arguments]  ${topic_name}
    ${list_topics}=  Get Topics List  ${admin}
    Should Contain  ${list_topics}  ${topic_name}

Check Consumed Message
    [Arguments]  ${consumer}  ${topic_name}  ${message}
    ${receivedMessage}=  Consume Message  ${consumer}  ${topic_name}
    Should Contain  ${receivedMessage}  ${message}

*** Test Cases ***
V2 Backup Uses Default S3 Alias Container
    [Tags]  kafka  backup  backup_v2  backup_v2_aliases
    ${backup_id}=  Set Variable  ${None}
    ${restore_id}=  Set Variable  ${None}
    Ensure S3 Aliases Config Available
    Create Topic With Generated Data  ${KAFKA_BACKUP_V2_TOPIC}
    ${backup_id}=  Create Backup V2  ${KAFKA_BACKUP_V2_TOPIC}  ${BACKUP_BLOB_PATH_ALIAS_TEST}
    Check Backup Exists In Default Alias Bucket  ${backup_id}  ${BACKUP_BLOB_PATH_ALIAS_TEST}
    Check Backup Does Not Exist In Default Alias Bucket  ${backup_id}  ${BACKUP_BLOB_PATH}
    Delete Data  ${KAFKA_BACKUP_V2_TOPIC}
    ${restore_id}=  Restore Backup V2  ${backup_id}  ${KAFKA_BACKUP_V2_TOPIC}  ${BACKUP_BLOB_PATH_ALIAS_TEST}
    Check Topic Exists  ${KAFKA_BACKUP_V2_TOPIC}
    Delete Backup V2  ${backup_id}  ${BACKUP_BLOB_PATH_ALIAS_TEST}
    [Teardown]  Run Keywords  Delete Data  ${KAFKA_BACKUP_V2_TOPIC}  AND  Delete Restore V2 If Exists  ${restore_id}  ${BACKUP_BLOB_PATH_ALIAS_TEST}  AND  Delete Backup V2 If Exists  ${backup_id}  ${BACKUP_BLOB_PATH_ALIAS_TEST}
