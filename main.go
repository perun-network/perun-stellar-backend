package main

import (
	"context"
	_ "github.com/stellar/go/txnbuild"
	"perun.network/go-perun/wire"
	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/util"
)

const PerunContractPath = "./testdata/perun_soroban_contract.wasm"
const StellarAssetContractPath = "./testdata/perun_soroban_token.wasm"

func main() {

	stellarEnv := env.NewBackendEnv()

	kps, _ := stellarEnv.CreateAccounts(5, "100000000")
	kpAlice := kps[0]
	kpBob := kps[1]
	kpDeployerPerun := kps[2]
	kpDeployerToken := kps[3]
	_ = kps[4]

	tokenContractIDAddress := util.Deploy(stellarEnv, kpDeployerToken, stellarEnv.AccountDetails(kpDeployerToken), StellarAssetContractPath)

	adminScAddr, err := types.MakeAccountAddress(kpDeployerToken)
	if err != nil {
		panic(err)
	}
	decims := uint32(7)
	tokenName := "PerunToken"
	tokenSymbol := "PRN"

	deployerStellarClient := env.NewStellarClient(stellarEnv, kpDeployerToken)
	// aliceStellarClient := env.NewStellarClient(stellarEnv, kpAlice)

	err = channel.InitTokenContract(context.TODO(), deployerStellarClient, adminScAddr, decims, tokenName, tokenSymbol, tokenContractIDAddress) //tokenContractIDAddress)
	if err != nil {
		panic(err)
	}

	err = channel.GetTokenName(context.TODO(), deployerStellarClient, tokenContractIDAddress)
	if err != nil {
		panic(err)
	}

	aliceAddrXdr, err := types.MakeAccountAddress(kpAlice)
	if err != nil {
		panic(err)
	}
	bobAddrXdr, err := types.MakeAccountAddress(kpBob)
	if err != nil {
		panic(err)
	}

	mintAmount := int64(100000000)

	err = channel.MintToken(context.TODO(), deployerStellarClient, aliceAddrXdr, mintAmount, tokenContractIDAddress)
	if err != nil {
		panic(err)
	}
	err = channel.MintToken(context.TODO(), deployerStellarClient, bobAddrXdr, mintAmount, tokenContractIDAddress)
	if err != nil {
		panic(err)
	}

	err = channel.GetTokenBalance(context.TODO(), deployerStellarClient, aliceAddrXdr, tokenContractIDAddress)
	if err != nil {
		panic(err)
	}
	err = channel.GetTokenBalance(context.TODO(), deployerStellarClient, bobAddrXdr, tokenContractIDAddress)
	if err != nil {
		panic(err)
	}

	perunContractIDAddress := util.Deploy(stellarEnv, kpDeployerPerun, stellarEnv.AccountDetails(kpDeployerPerun), PerunContractPath)
	stellarEnv.SetPerunAddress(perunContractIDAddress)
	stellarEnv.SetTokenAddress(tokenContractIDAddress)

	// Generate L2 accounts for the payment channel
	wAlice, accAlice, _ := util.MakeRandPerunWallet()
	wBob, accBob, _ := util.MakeRandPerunWallet()
	assetCID, err := types.NewStellarAssetFromScAddress(tokenContractIDAddress) //  tokenContractIDAddress //util.NewRandAsset()
	if err != nil {
		panic(err)
	}
	assetContractID := *assetCID

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
