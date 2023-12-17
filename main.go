package main

import (
	// "context"
	"fmt"

	"github.com/stellar/go/keypair"
	// "github.com/stellar/go/txnbuild"
	// "github.com/stellar/go/xdr"

	"log"

	"perun.network/go-perun/wire"
	// "perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/util"
	// "perun.network/perun-stellar-backend/wallet"
)

const PerunContractPath = "./testdata/perun_soroban_contract.wasm"
const StellarAssetContractPath = "./testdata/perun_soroban_token.wasm"

func main() {

	wAlice, accAlice, kpAlice := util.MakeRandPerunWallet()
	wBob, accBob, kpBob := util.MakeRandPerunWallet()
	_, _, kpDepToken := util.MakeRandPerunWallet()
	_, _, kpDepPerun := util.MakeRandPerunWallet()
	kps := []*keypair.Full{kpAlice, kpBob, kpDepToken, kpDepPerun}

	err := util.CreateFundStellarAccounts(kps, len(kps), "1000000")
	if err != nil {
		panic(err)
	}

	tokenAddr, assetHash := util.Deploy(kpDepToken, StellarAssetContractPath)

	err = util.InitTokenContract(kpDepToken, tokenAddr)
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

	err = util.MintToken(kpDepToken, tokenAddr, int64(10000000000), aliceAddr)
	if err != nil {
		panic(err)
	}

	err = util.MintToken(kpDepToken, tokenAddr, int64(10000000000), bobAddr)
	if err != nil {
		panic(err)
	}

	perunAddr, perunHash := util.Deploy(kpDepPerun, PerunContractPath)
	fmt.Println("assetID, assetHash, perunID, perunHas: ", perunAddr, assetHash, perunHash)

	err = util.MintToken(kpDepToken, tokenAddr, int64(10000000000), perunAddr)
	if err != nil {
		panic(err)
	}

	bus := wire.NewLocalBus()
	alicePerun, err := client.SetupPaymentClient(wAlice, accAlice, kpAlice, tokenAddr, perunAddr, bus)
	if err != nil {
		panic(err)
	}

	bobPerun, err := client.SetupPaymentClient(wBob, accBob, kpBob, tokenAddr, perunAddr, bus)

	if err != nil {
		panic(err)
	}
	alicePerun.OpenChannel(bobPerun.WireAddress(), 100)
	aliceChannel := alicePerun.Channel
	bobChannel := bobPerun.AcceptedChannel()

	aliceChannel.SendPayment(10)
	bobChannel.SendPayment(2)

	aliceChannel.Settle()
	bobChannel.Settle()

	alicePerun.Shutdown()
	bobPerun.Shutdown()

	fmt.Println("Get Balances: ")

	err = util.GetTokenBalance(kpAlice, tokenAddr, aliceAddr)
	if err != nil {
		panic(err)
	}

	err = util.GetTokenBalance(kpBob, tokenAddr, bobAddr)
	if err != nil {
		panic(err)
	}

	log.Println("DONE")
}
