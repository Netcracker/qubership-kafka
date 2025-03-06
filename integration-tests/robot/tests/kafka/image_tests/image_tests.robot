*** Variables ***
${MONITORED_IMAGES}         %{MONITORED_IMAGES}

*** Settings ***
Resource  ../../shared/keywords.robot


*** Test Cases ***
Test Hardcoded Images
  [Tags]  kafka  kafka_images
  ${stripped_resources}=  Strip String  ${MONITORED_IMAGES}  characters=,  mode=right
  @{list_resources} =  Split String	${stripped_resources} 	,
  FOR  ${resource}  IN  @{list_resources}
    ${type}  ${name}  ${container_name}  ${image}=  Split String	${resource}
    ${resource_image}=  Get Resource Image  ${type}  ${name}  %{OS_PROJECT}  ${container_name}
    Should Be Equal  ${resource_image}  ${image}
  END