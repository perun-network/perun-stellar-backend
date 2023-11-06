package channel_test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"

	chtest "perun.network/perun-stellar-backend/channel/test"
	"perun.network/perun-stellar-backend/util"

	_ "perun.network/perun-stellar-backend/wallet/test"
	"perun.network/perun-stellar-backend/wire"

	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/env"

	"testing"
)

func TestCloseChannel(t *testing.T) {
	itest := env.NewBackendEnv()

	kps, _ := itest.CreateAccounts(3, "100000000")

	kpAlice := kps[0]
	kpBob := kps[1]
	kpDeployer := kps[2]

	// reqAlice := itest.AccountDetails(kpAlice)
	// reqBob := itest.AccountDetails(kpBob)
	reqDeployer := itest.AccountDetails(kpDeployer)

	//fmt.Println("reqAlice, reqBob: ", reqAlice.Balances, reqBob.Balances)
	contractAddr := util.Deploy(itest, kpDeployer, reqDeployer, PerunContractPath)
	itest.SetContractIDAddress(contractAddr)
	fmt.Println("contractID: ", contractAddr)

	//perunFirstParams, perunFirstState := chtest.NewParamsState(t)
	wAlice, accAlice, _ := util.MakeRandPerunWallet()
	wBob, accBob, _ := util.MakeRandPerunWallet()
	addrAlice := accAlice.Address()
	addrBob := accBob.Address()
	addrList := []pwallet.Address{addrAlice, addrBob}
	perunFirstParams, perunFirstState := chtest.NewParamsWithAddressState(t, addrList)
	fmt.Println("perunFirstParams, perunFirstState: ", perunFirstParams, perunFirstState)

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
	//aliceIdx := false
	//fundArgsAlice, err := env.BuildFundTxArgs(chanID, aliceIdx)
	//require.NoError(t, err)

	// Calling the function
	err := chtest.FundAll(context.TODO(), funders, freqs)
	require.NoError(t, err)
	fmt.Println("Funded all")
	fmt.Println("aliceFunder: ", aliceFunder)
	fmt.Println("aliceFunder: ", wBob, accBob, wAlice)

	hzAliceGetCh := aliceClient.GetHorizonAcc()

	getChannelArgs, err := env.BuildGetChannelTxArgs(chanID)
	require.NoError(t, err)

	txMetaGetChAfterFunding, err := aliceClient.InvokeAndProcessHostFunction(hzAliceGetCh, "get_channel", getChannelArgs, contractAddr, kpAlice)
	require.NoError(t, err)

	retVal := txMetaGetChAfterFunding.V3.SorobanMeta.ReturnValue
	fmt.Println("retVal txMetaGetChAfterFunding: ", retVal)

	var getChanAfterFunding wire.Channel

	err = getChanAfterFunding.FromScVal(retVal)
	require.NoError(t, err)
	fmt.Println("getChanAfterFunding: ", getChanAfterFunding)

	fmt.Println("getChan.Control.FundedA, getChan.Control.FundedA: ", getChanAfterFunding.Control.FundedA, getChanAfterFunding.Control.FundedA)

	// close the channel

	currStellarState := getChanAfterFunding.State

	currStellarState.Finalized = true

	currPerunState, err := wire.ToState(currStellarState)

	require.NoError(t, err)
	fmt.Println("currStellarState: ", currStellarState)

	sigA, err := channel.Backend.Sign(accAlice, &currPerunState)
	require.NoError(t, err)

	sigB, err := channel.Backend.Sign(accBob, &currPerunState)
	require.NoError(t, err)

	sigs := []pwallet.Sig{sigA, sigB}

	// addrAlice := kpAlice.FromAddress()
	// channel.Backend.Sign(addrAlice, currStellarState)

	closeArgs, err := channel.BuildCloseTxArgs(currPerunState, sigs)
	require.NoError(t, err)
	hzAliceGClose := aliceClient.GetHorizonAcc()

	invokeHostFunctionOpClose := env.BuildContractCallOp(hzAliceGClose, "close", closeArgs, contractAddr)

	preFlightOpClose, minFeeClose := itest.PreflightHostFunctions(&hzAliceGClose, *invokeHostFunctionOpClose)

	fmt.Println("minFeeClose: ", minFeeClose)

	txClose, err := itest.SubmitOperationsWithFee(&hzAliceGClose, kpAlice, minFeeClose, &preFlightOpClose)

	require.NoError(t, err)

	txMetaClose, err := env.DecodeTxMeta(txClose)

	require.NoError(t, err)

	perunEventsClose, err := channel.DecodeEvents(txMetaClose)

	require.NoError(t, err)
	fmt.Println("perunEventsClose: ", perunEventsClose)

	// now both users can withdraw their funds

}

func TestWithdrawChannel(t *testing.T) {
	itest := env.NewBackendEnv()

	kps, _ := itest.CreateAccounts(3, "100000000")

	kpAlice := kps[0]
	kpBob := kps[1]
	kpDeployer := kps[2]

	reqDeployer := itest.AccountDetails(kpDeployer)

	//fmt.Println("reqAlice, reqBob: ", reqAlice.Balances, reqBob.Balances)
	contractAddr := util.Deploy(itest, kpDeployer, reqDeployer, PerunContractPath)
	itest.SetContractIDAddress(contractAddr)

	//perunFirstParams, perunFirstState := chtest.NewParamsState(t)
	_, accAlice, _ := util.MakeRandPerunWallet()
	_, accBob, _ := util.MakeRandPerunWallet()

	addrAlice := accAlice.Address()
	addrBob := accBob.Address()
	addrList := []pwallet.Address{addrAlice, addrBob}
	perunFirstParams, perunFirstState := chtest.NewParamsWithAddressState(t, addrList)

	freqAlice := pchannel.NewFundingReq(perunFirstParams, perunFirstState, 0, perunFirstState.Balances)
	freqBob := pchannel.NewFundingReq(perunFirstParams, perunFirstState, 1, perunFirstState.Balances)
	freqs := []*pchannel.FundingReq{freqAlice, freqBob}

	// Creating the client and funders as pointers
	aliceClient := env.NewStellarClient(itest, kpAlice)
	bobClient := env.NewStellarClient(itest, kpBob)

	aliceFunder := channel.NewFunder(accAlice, kpAlice, aliceClient)
	bobFunder := channel.NewFunder(accBob, kpBob, bobClient)
	funders := []*channel.Funder{aliceFunder, bobFunder}
	chanID := perunFirstParams.ID()

	// Calling the function
	err := chtest.FundAll(context.TODO(), funders, freqs)
	require.NoError(t, err)
	fmt.Println("Channel fully funded")

	hzAliceGetCh := aliceClient.GetHorizonAcc()

	getChannelArgs, err := env.BuildGetChannelTxArgs(chanID)
	require.NoError(t, err)

	txMetaGetChAfterFunding, err := aliceClient.InvokeAndProcessHostFunction(hzAliceGetCh, "get_channel", getChannelArgs, contractAddr, kpAlice)
	require.NoError(t, err)

	retVal := txMetaGetChAfterFunding.V3.SorobanMeta.ReturnValue

	var getChanAfterFunding wire.Channel

	err = getChanAfterFunding.FromScVal(retVal)
	require.NoError(t, err)

	// close the channel

	currStellarState := getChanAfterFunding.State

	currStellarState.Finalized = true

	currPerunState, err := wire.ToState(currStellarState)

	require.NoError(t, err)

	sigA, err := channel.Backend.Sign(accAlice, &currPerunState)
	require.NoError(t, err)

	sigB, err := channel.Backend.Sign(accBob, &currPerunState)
	require.NoError(t, err)

	sigs := []pwallet.Sig{sigA, sigB}

	closeArgs, err := channel.BuildCloseTxArgs(currPerunState, sigs)
	require.NoError(t, err)
	hzAliceGClose := aliceClient.GetHorizonAcc()

	invokeHostFunctionOpClose := env.BuildContractCallOp(hzAliceGClose, "close", closeArgs, contractAddr)

	preFlightOpClose, minFeeClose := itest.PreflightHostFunctions(&hzAliceGClose, *invokeHostFunctionOpClose)

	txClose, err := itest.SubmitOperationsWithFee(&hzAliceGClose, kpAlice, minFeeClose, &preFlightOpClose)

	require.NoError(t, err)

	txMetaClose, err := env.DecodeTxMeta(txClose)

	require.NoError(t, err)

	_, err = channel.DecodeEvents(txMetaClose)
	require.NoError(t, err)

	hzAliceWithdraw := aliceClient.GetHorizonAcc()
	waArgs, err := env.BuildFundTxArgs(chanID, false)
	require.NoError(t, err)
	invokeHostFunctionOpWA := env.BuildContractCallOp(hzAliceWithdraw, "withdraw", waArgs, contractAddr)
	preFlightOpWA, minFeeWA := itest.PreflightHostFunctions(&hzAliceWithdraw, *invokeHostFunctionOpWA)
	txWithdrawAlice, err := itest.SubmitOperationsWithFee(&hzAliceWithdraw, kpAlice, minFeeWA, &preFlightOpWA)
	require.NoError(t, err)

	txMetaWA, err := env.DecodeTxMeta(txWithdrawAlice)

	require.NoError(t, err)
	_, err = channel.DecodeEvents(txMetaWA)
	require.NoError(t, err)

	hzBobWithdraw := bobClient.GetHorizonAcc()
	wbArgs, err := env.BuildFundTxArgs(chanID, true)
	require.NoError(t, err)
	invokeHostFunctionOpWB := env.BuildContractCallOp(hzBobWithdraw, "withdraw", wbArgs, contractAddr)
	preFlightOpWB, minFeeWB := itest.PreflightHostFunctions(&hzBobWithdraw, *invokeHostFunctionOpWB)
	txWithdrawBob, err := itest.SubmitOperationsWithFee(&hzBobWithdraw, kpBob, minFeeWB, &preFlightOpWB)
	require.NoError(t, err)

	txMetaWB, err := env.DecodeTxMeta(txWithdrawBob)

	require.NoError(t, err)
	perunEventsWB, err := channel.DecodeEvents(txMetaWB)
	require.NoError(t, err)

	fmt.Println("perunEventsWB: ", perunEventsWB)

}
