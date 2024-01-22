<h1 align="center">
    <a href="https://perun.network/"><img src=".assets/go-perun.png" alt="Perun" width="30%"></a>
</h1>


# [Perun](https://perun.network/) Stellar backend

This repository contains the [Stellar](https://stellar.org/) backend for the [go-perun](https://github.com/perun-network/go-perun) channel library. It provides the necessary components to run our secure peer-to-peer Perun Payment Channels on the Stellar blockchain, using a [Soroban](https://soroban.stellar.org/) smart contract implementation. The payment channel implementation connects our Perun state machine with the [Perun contract](https://github.com/perun-network/perun-soroban-contract), which implements the Perun Payment Channel logic on the Stellar blockchain. To connect to the Stellar blockchain, we use the Horizon client service of the Stellar Go SDK, which is located [here](https://github.com/stellar/go). 

This project is financed through the Stellar Community Fund grants program.

In the following sections, we will describe how to run our Payment Channels on a local instance of the Stellar blockchain.

## [Setup](#setup)

1. Clone this repository:

```
git clone https://github.com/perun-network/perun-stellar-backend
cd perun-stellar-backend
```


2.  To run a local Stellar blockchain with Soroban smart contract support, you initialize the docker images defined in the ```quickstart.sh``` script. Docker needs to be installed to perform this step:

```sh

./quickstart.sh standalone
```

Note that this backend is customized to run on a local Stellar blockchain (standalone), but can be easily adapted to run on a public testnet.


3. Running the payment channel tests:

To make sure that your setup is correct, you can run the payment channel tests after cloning this repository:
  
```sh

go test ./...
```

# Payment Channel Demo

A demonstrator of Perun payment channels on the Stellar blockchain can be found [here](https://github.com/perun-network/perun-stellar-demo).

# Copyright

Copyright 2024 PolyCrypt GmbH. Use of the source code is governed by the Apache 2.0 license that can be found in the [LICENSE file](LICENSE).