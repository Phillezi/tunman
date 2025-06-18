#!/usr/bin/env bash

pushd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null || { log_err "Failed to change to script directory."; exit 1; }

docker buildx build -t ssh-testserv --load . || popd > /dev/null

docker run --rm -it -p 2222:22 ssh-testserv || popd > /dev/null

popd > /dev/null
