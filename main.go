package main

import (
	"fmt"
	"github.com/stellar/go/keypair"
	"log"
	"perun.network/go-perun/wire"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/util"
)

const PerunContractPath = "./testdata/perun_soroban_contract.wasm"
const StellarAssetContractPath = "./testdata/perun_soroban_token.wasm"

func main() {
	wAlice, accAlice, kpAlice := util.MakeRandPerunWallet()
	wBob, accBob, kpBob := util.MakeRandPerunWallet()
	_, _, kpDepToken := util.MakeRandPerunWallet()
	_, _, kpDepPerun := util.MakeRandPerunWallet()
	kps := []*keypair.Full{kpAlice, kpBob, kpDepToken, kpDepPerun}

	checkErr(util.CreateFundStellarAccounts(kps, len(kps), "1000000"))

	tokenAddr, _ := util.Deploy(kpDepToken, StellarAssetContractPath)
	checkErr(util.InitTokenContract(kpDepToken, tokenAddr))

	aliceAddr, err := util.MakeAccountAddress(kpAlice)
	checkErr(err)
	bobAddr, err := util.MakeAccountAddress(kpBob)
	checkErr(err)

	checkErr(util.MintToken(kpDepToken, tokenAddr, 1000000, aliceAddr))
	checkErr(util.MintToken(kpDepToken, tokenAddr, 1000000, bobAddr))

	perunAddr, _ := util.Deploy(kpDepPerun, PerunContractPath)

	bus := wire.NewLocalBus()
	alicePerun, err := client.SetupPaymentClient(wAlice, accAlice, kpAlice, tokenAddr, perunAddr, bus)
	checkErr(err)
	bobPerun, err := client.SetupPaymentClient(wBob, accBob, kpBob, tokenAddr, perunAddr, bus)
	checkErr(err)

	alicePerun.OpenChannel(bobPerun.WireAddress(), 100)
	aliceChannel, bobChannel := alicePerun.Channel, bobPerun.AcceptedChannel()

	aliceChannel.SendPayment(150)
	bobChannel.SendPayment(220)

	aliceChannel.Settle()
	bobChannel.Settle()

	alicePerun.Shutdown()
	bobPerun.Shutdown()

	fmt.Println("Get Balances: ")
	checkErr(util.GetTokenBalance(kpAlice, tokenAddr, aliceAddr))
	checkErr(util.GetTokenBalance(kpBob, tokenAddr, bobAddr))

	log.Println("DONE")
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
