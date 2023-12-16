package main

import (
	"context"
	"fmt"
	"log"
	"perun.network/go-perun/wire"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/util"
)

const PerunContractPath = "./testdata/perun_soroban_contract.wasm"
const StellarAssetContractPath = "./testdata/perun_soroban_token.wasm"

func main() {

	kps, _ := util.CreateFundNewRandomStellarKP(4, "1000000")
	kpAlice := kps[0]
	kpBob := kps[1]
	kpDeployerToken := kps[2]
	kpDeployerPerun := kps[3]

	tokenAddr, assetHash := util.Deploy(kpDeployerToken, StellarAssetContractPath)
	// util.MintAsset(kpDeployerToken, tokenAddr, kpAlice.Address(), 1000000)
	// util.MintAsset(kpDeployerToken, tokenAddr, kpBob.Address(), 1000000)

	err := util.InitTokenContract(kpDeployerToken, tokenAddr)
	if err != nil {
		panic(err)
	}

	aliceAddr, err := util.MakeAccountAddress(kpAlice)
	if err != nil {
		panic(err)
	}

	bobAddr, err := util.MakeAccountAddress(kpBob)
	if err != nil {
		panic(err)
	}

	err = util.MintToken(kpDeployerToken, tokenAddr, int64(10000000000), aliceAddr)
	if err != nil {
		panic(err)
	}

	err = util.MintToken(kpDeployerToken, tokenAddr, int64(10000000000), bobAddr)
	if err != nil {
		panic(err)
	}

	perunAddr, perunHash := util.Deploy(kpDeployerPerun, PerunContractPath)
	fmt.Println("assetID, assetHash, perunID, perunHas: ", perunAddr, assetHash, perunHash)

	// // Generate L2 accounts for the payment channel
	wAlice, accAlice, _ := util.MakeRandPerunWallet()
	wBob, accBob, _ := util.MakeRandPerunWallet()

	bus := wire.NewLocalBus()
	alicePerun, err := client.SetupPaymentClient(wAlice, accAlice, kpAlice, tokenAddr, perunAddr, bus)
	if err != nil {
		panic(err)
	}

	bobPerun, err := client.SetupPaymentClient(wBob, accBob, kpBob, tokenAddr, perunAddr, bus)

	if err != nil {
		panic(err)
	}
	alicePerun.OpenChannel(bobPerun.WireAddress(), 10)
	aliceChannel := alicePerun.Channel
	bobChannel := bobPerun.AcceptedChannel()

	aliceChannel.Settle()
	bobChannel.Settle()

	alicePerun.Shutdown()
	bobPerun.Shutdown()

	log.Println("Done")
	// // initialize the contract

	cl := env.NewStellarClient(kpAlice)

	err = cl.TestInteractContract(context.TODO(), kpAlice, tokenAddr, perunAddr)
	if err != nil {
		panic(err)
	}

	log.Println("DONE")
}
