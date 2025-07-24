*** Variables ***
${UNCLEAN_LEADER_ELECTION_TOPIC}    unclean-leader-election-test-topic
${FIND_METRIC_RETRY_TIME}           5min
${FIND_METRIC_RETRY_INTERVAL}       10s

*** Settings ***
Library  MonitoringLibrary  host=%{PROMETHEUS_URL}
...                         username=%{PROMETHEUS_USER}
...                         password=%{PROMETHEUS_PASSWORD}
Resource  ../../shared/keywords.robot

*** Keywords ***
Check JMX Metrics Exist In Prometheus
    ${data}=  Get Metric Values  kafka_server_ReplicaManager_Count_total
    Should Not Be Empty  ${data}
    Should Contain  str(${data})  %{KAFKA_OS_PROJECT}

Check Unclean Leader Election Metric Exists In Prometheus
    ${data}=  Get Metric Values  kafka_cluster_unclean_election_topics{namespace="%{KAFKA_OS_PROJECT}"}
    Should Not Be Empty  ${data}
    Should Be True      "${UNCLEAN_LEADER_ELECTION_TOPIC}" in """${data['result']}"""

*** Test Cases ***
Check JMX Metrics
    [Tags]  kafka  prometheus  jmx_metrics
    Wait Until Keyword Succeeds  ${FIND_METRIC_RETRY_TIME}  ${FIND_METRIC_RETRY_INTERVAL}
    ...  Check JMX Metrics Exist In Prometheus

Check Unclean Leader Election Metric
    [Tags]  kafka  prometheus  unclean_leader_election_metric
    ${admin} =  Create Admin Client
    &{configs}=  Create Dictionary  unclean.leader.election.enable=true
    Create Topic  ${admin}  ${UNCLEAN_LEADER_ELECTION_TOPIC}  ${3}  ${1}  ${configs}
    Wait Until Keyword Succeeds  ${FIND_METRIC_RETRY_TIME}  ${FIND_METRIC_RETRY_INTERVAL}
    ...  Check Unclean Leader Election Metric Exists In Prometheus
    [Teardown]  Delete Topic  ${admin}  ${UNCLEAN_LEADER_ELECTION_TOPIC}