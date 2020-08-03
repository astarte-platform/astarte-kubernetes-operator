#!/bin/bash

for i in {1..80}
do
    if [[ $(kubectl get astarte -n astarte-test example-astarte -o json | jq -r .status.health) == "green" ]]; then
        echo "Astarte Ready!"
        exit 0
    fi

    sleep 10
done

echo "Timed out waiting"
exit 1
