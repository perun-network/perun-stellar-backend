package main

import (
	"perun.network/go-perun/wire"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/util"
)

const PerunContractPath = "./testdata/perun_soroban_contract.wasm"

func main() {

	// Create a Backend to interact with the Stellar network: integration test environment from the stellar/go sdk
	stellarEnv := env.NewBackendEnv()

	// Create two Stellar L1 accounts
	kps, _ := stellarEnv.CreateAccounts(3, "10000000")

	kpAlice := kps[0]
	kpBob := kps[1]
	kpDeployer := kps[2]

	_ = stellarEnv.AccountDetails(kpAlice)
	_ = stellarEnv.AccountDetails(kpBob)
	hzDeployer := stellarEnv.AccountDetails(kpDeployer)

	// Deploy the contract

	contractIDAddress := util.Deploy(stellarEnv, kpDeployer, hzDeployer, PerunContractPath)
	stellarEnv.SetContractIDAddress(contractIDAddress)
	//fmt.Println("Deployed contractIDAddress: ", contractIDAddress)

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
