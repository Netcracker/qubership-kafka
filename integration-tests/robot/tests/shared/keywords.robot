*** Variables ***
${KAFKA_BOOTSTRAP_SERVERS}      %{KAFKA_BOOTSTRAP_SERVERS}
${KAFKA_HOST}                   %{KAFKA_HOST}

*** Settings ***
Library  String
Library  Collections
Library  ./lib/KafkaLibrary.py  bootstrap_servers=${KAFKA_BOOTSTRAP_SERVERS}
...                             namespace=%{KAFKA_OS_PROJECT}
...                             host=${KAFKA_HOST}
...                             port=%{KAFKA_PORT}
...                             username=%{KAFKA_USER}
...                             password=%{KAFKA_PASSWORD}
...                             enable_ssl=%{KAFKA_ENABLE_SSL}
Library  PlatformLibrary  managed_by_operator=%{KAFKA_IS_MANAGED_BY_OPERATOR}

