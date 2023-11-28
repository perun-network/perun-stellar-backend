package main

import (
	"fmt"
	"perun.network/go-perun/wire"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/util"
)

const PerunContractPath = "./testdata/perun_soroban_contract.wasm"
const StellarAssetContractPath = "./testdata/perun_soroban_token.wasm"

func main() {

	// Create a Backend to interact with the Stellar network: integration test environment from the stellar/go sdk
	stellarEnv := env.NewBackendEnv()

	// Create two Stellar L1 accounts
	kps, _ := stellarEnv.CreateAccounts(4, "100000000")

	kpAlice := kps[0]
	kpBob := kps[1]
	kpDeployerPerun := kps[2]
	kpDeployerToken := kps[3]

	realAssetContractID := util.Deploy(stellarEnv, kpDeployerToken, stellarEnv.AccountDetails(kpDeployerToken), StellarAssetContractPath)

	fmt.Println("Deployed Real Asset Contract ID: ", realAssetContractID)
	// Deploy the Perun contract
	contractIDAddress := util.Deploy(stellarEnv, kpDeployerPerun, stellarEnv.AccountDetails(kpDeployerPerun), PerunContractPath)
	stellarEnv.SetContractIDAddress(contractIDAddress)

	// Generate L2 accounts for the payment channel
	wAlice, accAlice, _ := util.MakeRandPerunWallet()
	wBob, accBob, _ := util.MakeRandPerunWallet()
	assetContractID := util.NewRandAsset()
	bus := wire.NewLocalBus()
	alicePerun, err := client.SetupPaymentClient(stellarEnv, wAlice, accAlice, kpAlice, assetContractID, bus)
	if err != nil {
		panic(err)
	}

	bobPerun, err := client.SetupPaymentClient(stellarEnv, wBob, accBob, kpBob, assetContractID, bus)
	if err != nil {
		panic(err)
	}

	alicePerun.OpenChannel(bobPerun.WireAddress(), 1000)
	aliceChannel := alicePerun.Channel
	bobChannel := bobPerun.AcceptedChannel()

	aliceChannel.Settle()
	bobChannel.Settle()

	alicePerun.Shutdown()
	bobPerun.Shutdown()

}
