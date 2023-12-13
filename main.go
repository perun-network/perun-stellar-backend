package main

import (
	"context"
	"log"
	"perun.network/go-perun/wire"
	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/util"
)

const PerunContractPath = "./testdata/perun_soroban_contract.wasm"
const StellarAssetContractPath = "./testdata/perun_soroban_token.wasm"
const tokenDecimals = uint32(7)
const tokenName = "PerunToken"
const tokenSymbol = "PRN"

func main() {

	stellarEnv := env.NewBackendEnv()
	kps, _ := stellarEnv.CreateAccounts(5, "1000000000")
	kpAlice := kps[0]
	kpBob := kps[1]
	kpDeployerPerun := kps[2]
	kpDeployerToken := kps[3]
	tokenContractIDAddress, _ := util.Deploy(stellarEnv, kpDeployerToken, stellarEnv.AccountDetails(kpDeployerToken), StellarAssetContractPath)

	adminScAddr, err := types.MakeAccountAddress(kpDeployerToken)
	if err != nil {
		panic(err)
	}

	tokenParams := channel.NewTokenParams(tokenDecimals, tokenName, tokenSymbol)

	deployerStellarClient := env.NewStellarClient(stellarEnv, kpDeployerToken)

	err = channel.InitTokenContract(context.TODO(), deployerStellarClient, adminScAddr, *tokenParams, tokenContractIDAddress)
	if err != nil {
		panic(err)
	}

	perunContractIDAddress, _ := util.Deploy(stellarEnv, kpDeployerPerun, stellarEnv.AccountDetails(kpDeployerPerun), PerunContractPath)
	stellarEnv.SetPerunAddress(perunContractIDAddress)
	stellarEnv.SetTokenAddress(tokenContractIDAddress)

	// // Generate L2 accounts for the payment channel
	wAlice, accAlice, _ := util.MakeRandPerunWallet()
	wBob, accBob, _ := util.MakeRandPerunWallet()

	assetContractID, err := types.NewStellarAssetFromScAddress(tokenContractIDAddress)
	if err != nil {
		panic(err)
	}

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

	log.Println("Done")

}
