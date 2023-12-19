<h1 align="center">
    <a href="https://perun.network/"><img src=".assets/go-perun.png" alt="Perun" width="30%"></a>
</h1>


# [Perun](https://perun.network/) Stellar backend

This repository contains the [Stellar](https://stellar.org/) backend for the [go-perun](https://github.com/perun-network/go-perun) channel library. It provides the necessary components to run our secure peer-to-peer Perun Payment Channels on the Stellar blockchain, using a [Soroban](https://soroban.stellar.org/) smart contract implementation. The payment channel implementation connects our Perun state machine with the [Perun contract](https://github.com/perun-network/perun-soroban-contract), which implements the Perun Payment Channel logic on the Stellar blockchain. This project is financed through the Stellar Community Fund grants program.

In the following sections, we will describe how to run our Payment Channels on a local instance of the Stellar blockchain.

## [Setup](#setup)

1. We the Stellar Go SDK, which is located [here](https://github.com/stellar/go). Our payment channels use the integration test environment of the Stellar Go SDK, which supports Soroban functionality. You can find more information on the test environment [here](https://github.com/perun-network/go/tree/master/services/horizon/internal/docs).

However, we will summarize the necessary steps to run the Stellar Go client with Soroban support here:

Clone the forked Stellar Go SDK:

```sh
git clone https://github.com/perun-network/go
```

Build the horizon binary:

```sh
cd go/services/horizon/
go build -o stellar-horizon && go install
```
Add the stellar-horizon executable in you PATH in your ~/.bashrc file. You can find the exact description of the process [here](https://github.com/perun-network/go/blob/master/services/horizon/internal/docs/GUIDE_FOR_DEVELOPERS.md#building-horizon).

To run the Postgres database server, you start a docker container in the docker folder:

```sh
docker-compose -f ./docker/docker-compose.yml up horizon-postgres
```

After using the Stellar backend, you can stop the Postgres database server with:

```sh
docker-compose -f ./docker/docker-compose.yml down
```

2. To use the integration test environment of the Stellar Go SDK, you need to set the correct environment variables:

```sh
export HORIZON_INTEGRATION_TESTS_ENABLED="true"
export HORIZON_INTEGRATION_TESTS_CORE_MAX_SUPPORTED_PROTOCOL="20"
export HORIZON_INTEGRATION_TESTS_ENABLE_SOROBAN_RPC="true"
export HORIZON_INTEGRATION_TESTS_DOCKER_IMG="stellar/stellar-core:19.13.1-1481.3acf6dd26.focal"
export HORIZON_INTEGRATION_TESTS_SOROBAN_RPC_DOCKER_IMG="stellar/soroban-rpc:20.0.0-rc3-39"
```

Note that these settings are required to support the Soroban smart contract functionality using the Stellar Go SDK. You can find the exact specifications on the test environment [here](https://github.com/stellar/go/blob/master/.github/workflows/horizon.yml). 

3. Running the payment channels:

Having set up the working environment in the previous steps, you can clone the repository and run the payment channel tests:
    
```sh
git clone https://github.com/perun-network/perun-stellar-backend
cd perun-stellar-backend
go test ./...
```

To run the simple payment channel demo, you can run the following command:

```sh
go run main.go
```

## Copyright

Copyright 2023 PolyCrypt GmbH. Use of the source code is governed by the Apache 2.0 license that can be found in the [LICENSE file](LICENSE).