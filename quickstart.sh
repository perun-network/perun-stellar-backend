#!/bin/sh

docker run --rm -it \
    -p "8000:8000" \
    --name stellar \
    stellar/quickstart:soroban-dev \
    --testnet \
    --enable-soroban-rpc