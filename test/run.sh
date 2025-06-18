#!/usr/bin/env bash

set -euo pipefail

log_err() { echo "ERROR: $*" >&2; }

pushd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null || { log_err "Failed to change to script directory."; exit 1; }

# Build Docker image
docker buildx build -t ssh-testserv --load . || { popd > /dev/null; exit 1; }

# Run container in detached mode and capture container ID
CONTAINER_ID=$(docker run -d -p 2222:22 ssh-testserv)
echo "Started container: $CONTAINER_ID"

# Determine which public SSH key to use
if [[ -f "${HOME}/.ssh/id_ed25519.pub" ]]; then
    PUBKEY_PATH="${HOME}/.ssh/id_ed25519.pub"
elif [[ -f "${HOME}/.ssh/id_rsa.pub" ]]; then
    PUBKEY_PATH="${HOME}/.ssh/id_rsa.pub"
else
    log_err "No public SSH key found in ~/.ssh/"
    docker rm -f "$CONTAINER_ID" > /dev/null
    popd > /dev/null
    exit 1
fi

PUBKEY=$(<"$PUBKEY_PATH")

# Inject the public key into the container
docker exec "$CONTAINER_ID" mkdir -p /root/.ssh
docker exec "$CONTAINER_ID" bash -c "echo '$PUBKEY' >> /root/.ssh/authorized_keys"
docker exec "$CONTAINER_ID" chmod 600 /root/.ssh/authorized_keys
docker exec "$CONTAINER_ID" chmod 700 /root/.ssh

echo "SSH key added from '$PUBKEY_PATH'. You can now connect with:"
echo "  ssh -p 2222 root@localhost"

popd > /dev/null
