package channel_test

import (
	"context"
	"fmt"
	"perun.network/perun-stellar-backend/util"

	"github.com/stellar/go/xdr"
	//"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	chtest "perun.network/perun-stellar-backend/channel/test"

	_ "perun.network/perun-stellar-backend/wallet/test"
	//"perun.network/perun-stellar-backend/wire"

	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/env"

	"testing"
)

const PerunContractPath = "../testdata/perun_soroban_contract.wasm"

func TestOpenChannel(t *testing.T) {
	itest := env.NewIntegrationEnv(t)

	kps, _ := itest.CreateAccounts(2, "10000000")

	kpAlice := kps[0]
	kpBob := kps[1]

	reqAlice := itest.AccountDetails(kpAlice)
	_ = itest.AccountDetails(kpBob)

	installContractOp := channel.AssembleInstallContractCodeOp(kpAlice.Address(), channel.PerunContractPath)
	preFlightOp, minFee := itest.PreflightHostFunctions(&reqAlice, *installContractOp)
	_ = itest.MustSubmitOperationsWithFee(&reqAlice, kpAlice, 5*minFee, &preFlightOp)

	// Create the contract

	createContractOp := channel.AssembleCreateContractOp(kpAlice.Address(), channel.PerunContractPath, "a1", itest.GetPassPhrase())
	preFlightOp, minFee = itest.PreflightHostFunctions(&reqAlice, *createContractOp)
	_, err := itest.SubmitOperationsWithFee(&reqAlice, kpAlice, minFee, &preFlightOp)

	require.NoError(t, err)

	// contract has been deployed, now invoke 'open' fn on the contract
	contractID := preFlightOp.Ext.SorobanData.Resources.Footprint.ReadWrite[0].MustContractData().Contract.ContractId
	require.NotNil(t, contractID)
	contractIDAddress := xdr.ScAddress{
		Type:       xdr.ScAddressTypeScAddressTypeContract,
		ContractId: contractID,
	}

	perunFirstParams, perunFirstState := chtest.NewParamsState(t)

	openArgs := env.BuildOpenTxArgs(perunFirstParams, perunFirstState)

	invokeHostFunctionOp := env.BuildContractCallOp(reqAlice, "open", openArgs, contractIDAddress)

	preFlightOp, minFee = itest.PreflightHostFunctions(&reqAlice, *invokeHostFunctionOp)

	tx, err := itest.SubmitOperationsWithFee(&reqAlice, kpAlice, minFee, &preFlightOp)
	require.NoError(t, err)

	txMeta, err := env.DecodeTxMeta(tx)
	require.NoError(t, err)

	perunEvents, err := channel.DecodeEvents(txMeta)
	require.NoError(t, err)

	fmt.Println("perunEvents: ", perunEvents)

}

func TestFundChannel(t *testing.T) {
	itest := env.NewBackendEnv()

	kps, _ := itest.CreateAccounts(3, "1000000000")

	kpAlice := kps[0]
	kpBob := kps[1]
	kpDeployer := kps[2]

	reqAlice := itest.AccountDetails(kpAlice)
	reqBob := itest.AccountDetails(kpBob)
	reqDeployer := itest.AccountDetails(kpDeployer)

	fmt.Println("reqAlice, reqBob: ", reqAlice.Balances, reqBob.Balances)
	contractAddr := util.Deploy(itest, kpDeployer, reqDeployer, PerunContractPath)
	itest.SetContractIDAddress(contractAddr)
	fmt.Println("contractID: ", contractAddr)

	perunFirstParams, perunFirstState := chtest.NewParamsState(t)

	fmt.Println("perunFirstParams, perunFirstState: ", perunFirstParams, perunFirstState)

	wAlice, accAlice, _ := util.MakeRandPerunWallet()
	wBob, accBob, _ := util.MakeRandPerunWallet()

	freqAlice := pchannel.NewFundingReq(perunFirstParams, perunFirstState, 0, perunFirstState.Balances)
	fmt.Println("freqAlice: ", freqAlice)
	freqBob := pchannel.NewFundingReq(perunFirstParams, perunFirstState, 1, perunFirstState.Balances)
	fmt.Println("freqBob: ", freqBob)
	freqs := []*pchannel.FundingReq{freqAlice, freqBob}
	fmt.Println("freqs: ", freqs)

	// Creating the client and funders as pointers
	aliceClient := env.NewStellarClient(itest, kpAlice)
	bobClient := env.NewStellarClient(itest, kpBob)

	aliceFunder := channel.NewFunder(accAlice, kpAlice, aliceClient)
	bobFunder := channel.NewFunder(accBob, kpBob, bobClient)
	funders := []*channel.Funder{aliceFunder, bobFunder}
	fmt.Println("funders: ", funders)
	chanID := perunFirstParams.ID()
	aliceIdx := false
	fundArgsAlice, err := env.BuildFundTxArgs(chanID, aliceIdx)
	require.NoError(t, err)
	fmt.Println("chanID: ", chanID)

	fmt.Println("fundArgs outside: ", contractAddr, kpAlice, reqAlice, chanID, fundArgsAlice, aliceIdx)
	//fmt.Println("funderchan args: ", contractAddress, kp, hzAcc, chanId, fundTxArgs, funderIdx)

	//fmt.Println("funderchan args: ", contractAddress, kp, hzAcc, chanId, funderIdx)

	// Calling the function
	err = chtest.FundAll(context.TODO(), funders, freqs)
	require.NoError(t, err)
	fmt.Println("Funded all")
	fmt.Println("aliceFunder: ", aliceFunder)
	fmt.Println("aliceFunder: ", wBob, accBob, wAlice)

	// err = aliceFunder.OpenChannel(context.TODO(), perunFirstParams, perunFirstState)
	// require.NoError(t, err)
	//chanOpened, err := aliceFunder.GetChannelState(context.TODO(), perunFirstParams, perunFirstState)
	//require.NoError(t, err)

	//fmt.Println("chanOpened state: ", chanOpened)

	// openArgs := env.BuildOpenTxArgs(perunFirstParams, perunFirstState)
	// fnameXdr := xdr.ScSymbol("open")
	// hzAlice := aliceClient.GetHorizonAcc()
	// hzBob := bobClient.GetHorizonAcc()

	// //invokeHostFunctionOp := env.BuildContractCallOp(reqAlice, fnameXdr, openArgs, contractAddr)
	// invokeHostFunctionOp := env.BuildContractCallOp(hzAlice, fnameXdr, openArgs, contractAddr)
	// preFlightOp, minFeeOpen := itest.PreflightHostFunctions(&hzAlice, *invokeHostFunctionOp)

	// _, err = itest.SubmitOperationsWithFee(&hzAlice, kpAlice, minFeeOpen, &preFlightOp)
	// require.NoError(t, err)

	// // check for Channel State

	// getChannelArgs, err := env.BuildGetChannelTxArgs(chanID)
	// require.NoError(t, err)
	// invokeHostFunctionOpGetChOpen := env.BuildContractCallOp(hzAlice, "get_channel", getChannelArgs, contractAddr)
	// preFlightOpGetChOpen, minFeeGetChOpen := itest.PreflightHostFunctions(&hzAlice, *invokeHostFunctionOpGetChOpen)
	// txGetChOpen, err := itest.SubmitOperationsWithFee(&hzAlice, kpAlice, minFeeGetChOpen, &preFlightOpGetChOpen)
	// require.NoError(t, err)

	// txMetaGetChOpen, err := env.DecodeTxMeta(txGetChOpen)
	// require.NoError(t, err)
	// fmt.Println("txMetaGetChOpen: ", txMetaGetChOpen)
	// retValOpen := txMetaGetChOpen.V3.SorobanMeta.ReturnValue
	// fmt.Println("retVal: ", retValOpen)

	// var getChanOpen wire.Channel

	// err = getChanOpen.FromScVal(retValOpen)
	// require.NoError(t, err)
	// fmt.Println("getChanOpen: ", getChanOpen)
	// // check get state

	// invokeHostFunctionOpFund := env.BuildContractCallOp(hzAlice, "fund", fundArgsAlice, contractAddr)

	// preFlightOpFund, minFeeFund := itest.PreflightHostFunctions(&hzAlice, *invokeHostFunctionOpFund)

	// txFund, err := itest.SubmitOperationsWithFee(&hzAlice, kpAlice, minFeeFund, &preFlightOpFund)
	// fmt.Println("minFeeFund: ", minFeeFund)
	// require.NoError(t, err)

	// txMetaFund, err := env.DecodeTxMeta(txFund)
	// require.NoError(t, err)

	// perunEventsFund, err := channel.DecodeEvents(txMetaFund)
	// require.NoError(t, err)
	// fmt.Println("perunEventsFund: ", perunEventsFund)

	// bobIdx := true
	// fundArgsBob, err := env.BuildFundTxArgs(chanID, bobIdx)
	// require.NoError(t, err)
	// invokeHostFunctionOpFund2 := env.BuildContractCallOp(hzBob, "fund", fundArgsBob, contractAddr)

	// preFlightOpFund2, minFeeFund2 := itest.PreflightHostFunctions(&hzBob, *invokeHostFunctionOpFund2)
	// fmt.Println("minFeeFund2: ", minFeeFund2)

	// txFund2, err := itest.SubmitOperationsWithFee(&reqBob, kpBob, minFeeFund2, &preFlightOpFund2)

	// require.NoError(t, err)
	// txMetaFund2, err := env.DecodeTxMeta(txFund2)
	// require.NoError(t, err)
	// fmt.Println("txMetaFund2: ", txMetaFund2)
	// perunEventsFund2, err := channel.DecodeEvents(txMetaFund2)
	// require.NoError(t, err)
	// fmt.Println("perunEventsFund2: ", perunEventsFund2)

	// // qurey channel state

	// // getChannelArgs, err := env.BuildGetChannelTxArgs(chanID)
	// // require.NoError(t, err)

	// invokeHostFunctionOpGetCh := env.BuildContractCallOp(hzAlice, "get_channel", getChannelArgs, contractAddr)
	// preFlightOpGetCh, minFeeGetCh := itest.PreflightHostFunctions(&hzAlice, *invokeHostFunctionOpGetCh)
	// txGetCh, err := itest.SubmitOperationsWithFee(&hzAlice, kpAlice, minFeeGetCh, &preFlightOpGetCh)
	// require.NoError(t, err)

	// txMetaGetCh, err := env.DecodeTxMeta(txGetCh)
	// require.NoError(t, err)
	// fmt.Println("txMetaGetCh: ", txMetaGetCh)
	// retVal := txMetaGetCh.V3.SorobanMeta.ReturnValue
	// fmt.Println("retVal: ", retVal)

	// var getChan wire.Channel

	// err = getChan.FromScVal(retVal)
	// require.NoError(t, err)
	// fmt.Println("getChan: ", getChan)
}
