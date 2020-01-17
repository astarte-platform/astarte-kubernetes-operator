#!/bin/bash

for i in {1..80}
do
    if [[ $(kubectl get deployment -n astarte-test example-astarte-housekeeping-api -o json | jq .status.readyReplicas) == "1" ]]; then
        echo "Housekeeping API Ready!"
        exit 0
    fi

    sleep 10
done

echo "Timed out waiting"
exit 1
