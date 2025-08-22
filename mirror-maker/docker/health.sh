#!/bin/bash

check_process_state() {
  local state=$(ps x -o pid -o stat -o comm | grep java | awk '{print $2}')
  if [[ ${state} == S* ]] || [[ ${state} == R* ]]; then
     echo "Kafka mirror maker is OK"
     exit 0
  fi
  echo "Process for Kafka Mirror Maker is frozen. State=${state}"
  exit 1
}

live_check() {
  check_process_state
}

ready_check() {
  check_process_state
}

case $1 in
  live)
    live_check
    exit $?
    ;;
  ready)
    ready_check
    exit $?
    ;;
esac

live_check