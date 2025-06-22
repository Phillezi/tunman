#!/usr/bin/env bash

BASE_CMD="./bin/tunman open localhost --port 2222 --remote 0.0.0.0:8080 --user root"
PREFIX="0.0.0.0:80"

for i in $(seq -w 0 99); do
    LOCAL_PORT="${PREFIX}${i}"
    #echo "Trying local port $LOCAL_PORT..."

    $BASE_CMD --local "$LOCAL_PORT"
    EXIT_CODE=$?

    # if [ $EXIT_CODE -eq 0 ]; then
    #     #echo "Tunnel opened successfully on local port $LOCAL_PORT"
    #     #exit 0
    # fi
done

