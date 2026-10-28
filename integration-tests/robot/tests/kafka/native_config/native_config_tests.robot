*** Variables ***
${KAFKA_API_GROUP}              %{KAFKA_API_GROUP=netcracker.com}
${KAFKA_CR_NAME}                %{KAFKA_CR_NAME=%{KAFKA_HOST}}
${OPERATION_RETRY_COUNT}        12
${OPERATION_RETRY_INTERVAL}     10s

*** Settings ***
Resource  ../../shared/keywords.robot

*** Keywords ***
Skip If Kafka Is Not Managed By Operator
    Skip If  '%{KAFKA_IS_MANAGED_BY_OPERATOR=true}'.lower() in ('false', '0', 'no')
    ...  Kafka is not managed by operator, Kafka custom resource is absent
    Skip If  'EXTERNAL_KAFKA' in os.environ  External Kafka has no Kafka custom resource

Get Kafka CR Spec
    ${status}  ${cr}=  Run Keyword And Ignore Error
    ...  Get Custom Resource  ${KAFKA_API_GROUP}/v1  Kafka  ${KAFKA_OS_PROJECT}  ${KAFKA_CR_NAME}
    Skip If  '${status}' == 'FAIL' and '(404)' in $cr
    ...  Kafka custom resource "${KAFKA_CR_NAME}" is not found in namespace "${KAFKA_OS_PROJECT}"
    Run Keyword If  '${status}' == 'FAIL'
    ...  Fail  Failed to get Kafka custom resource "${KAFKA_CR_NAME}": ${cr}
    RETURN  ${cr}[spec]

Check Broker Properties
    [Arguments]  ${expected}
    ${brokers}=  Get Broker Properties
    Should Not Be Empty  ${brokers}  No Kafka brokers are found
    FOR  ${broker_id}  ${properties}  IN  &{brokers}
        Dictionary Should Contain Sub Dictionary  ${properties}  ${expected}
        ...  msg=Configuration of broker ${broker_id} does not match Kafka CR
    END

*** Test Cases ***
Test Kafka CR Properties Are Applied To Brokers
    [Tags]  kafka  kafka_crud  native_config
    [Documentation]  Reads the Kafka custom resource and verifies via Admin API that every property set via
    ...  `config` and every `CONF_KAFKA_*` entry of `environmentVariables` is applied to all Kafka brokers.
    ...  When the same property is set in both sections, the `environmentVariables` value is expected.
    Skip If Kafka Is Not Managed By Operator
    ${spec}=  Get Kafka CR Spec
    ${config}=  Set Variable  ${{ $spec.get('config') or {} }}
    ${environment_properties}=  Normalize Environment Variables  ${{ $spec.get('environmentVariables') }}
    ${expected}=  Create Dictionary  &{config}  &{environment_properties}
    Skip If  not $expected  Kafka CR has neither config nor CONF_KAFKA_* environmentVariables to verify
    Log  Expected Kafka properties: ${expected}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Broker Properties  ${expected}
