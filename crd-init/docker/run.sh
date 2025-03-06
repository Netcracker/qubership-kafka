#!/bin/bash

echo "Running crd-init.py script. Script logs are below"
python crd-init.py
result=$?
if [[ $result != 0 ]] ; then
    echo "Something went wrong with script. Please, check logs above"
    exit 1
fi

echo "Sleep ${SLEEP_AFTER_COMPLETION} after script completion"
sleep ${SLEEP_AFTER_COMPLETION}
