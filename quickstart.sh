#!/bin/bash

set -e

case "$1" in
standalone)
    echo "Using standalone network"
    ARGS="--local"
    ;;
futurenet)
    echo "Using Futurenet network"
    ARGS="--futurenet"
    ;;
*)
    echo "Usage: $0 standalone|futurenet"
    exit 1
    ;;
esac

shift

# Run the soroban-preview container
# Remember to do:
# make build-docker

echo "Creating docker soroban network"
(docker network inspect soroban-network -f '{{.Id}}' 2>/dev/null) \
  || docker network create soroban-network

echo "Searching for a previous soroban-preview docker container"
containerID=$(docker ps --filter="name=soroban-preview" --all --quiet)
if [[ ${containerID} ]]; then
    echo "Start removing soroban-preview container."
    docker rm --force soroban-preview
    echo "Finished removing soroban-preview container."
else
    echo "No previous soroban-preview container was found"
fi

currentDir=$(pwd)
docker run -d \
  --volume ${currentDir}:/workspace \
  --memory="2g" \
  --cpus="2.0" \
  --name soroban-preview \
  -p 8001:8000 \
  --ipc=host \
  --network soroban-network \
  soroban-preview:10

# Run the stellar quickstart image

docker run --rm \
  --name stellar \
  --pull always \
  --network soroban-network \
  -p 8000:8000 \
  docker.io/stellar/quickstart@sha256:d4f752eece1e8780d19f4bd2726845996c413c0f4c4f57d1e4a9ff450442fc29 \
  $ARGS \
  --enable-soroban-rpc \
  --protocol-version 23 \
  --limits testnet \
  "$@" # Pass through args from the CLI
