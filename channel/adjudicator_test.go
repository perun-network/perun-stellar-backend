package channel_test

import (
	"context"
	"github.com/stellar/go/xdr"
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
	reqDeployer := itest.AccountDetails(kpDeployer)

	contractAddr := util.Deploy(itest, kpDeployer, reqDeployer, PerunContractPath)
	itest.SetPerunAddress(contractAddr)

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

	err := chtest.FundAll(context.TODO(), funders, freqs)
	require.NoError(t, err)

	hzAliceGetCh := aliceClient.GetHorizonAcc()

	getChannelArgs, err := env.BuildGetChannelTxArgs(chanID)
	require.NoError(t, err)

	auth := []xdr.SorobanAuthorizationEntry{}
	txMetaGetChAfterFunding, err := aliceClient.InvokeAndProcessHostFunction(hzAliceGetCh, "get_channel", getChannelArgs, contractAddr, kpAlice, auth)
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

	auth = []xdr.SorobanAuthorizationEntry{}
	invokeHostFunctionOpClose := env.BuildContractCallOp(hzAliceGClose, "close", closeArgs, contractAddr, auth)

	preFlightOpClose, minFeeClose := itest.PreflightHostFunctions(&hzAliceGClose, *invokeHostFunctionOpClose)

	txClose, err := itest.SubmitOperationsWithFee(&hzAliceGClose, kpAlice, minFeeClose, &preFlightOpClose)

	require.NoError(t, err)

	txMetaClose, err := env.DecodeTxMeta(txClose)

	require.NoError(t, err)

	_, err = channel.DecodeEventsPerun(txMetaClose)

	require.NoError(t, err)

}

func TestWithdrawChannel(t *testing.T) {
	itest := env.NewBackendEnv()

	kps, _ := itest.CreateAccounts(3, "100000000")

	kpAlice := kps[0]
	kpBob := kps[1]
	kpDeployer := kps[2]

	reqDeployer := itest.AccountDetails(kpDeployer)

	contractAddr := util.Deploy(itest, kpDeployer, reqDeployer, PerunContractPath)
	itest.SetPerunAddress(contractAddr)

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

	hzAliceGetCh := aliceClient.GetHorizonAcc()

	getChannelArgs, err := env.BuildGetChannelTxArgs(chanID)
	require.NoError(t, err)

	auth := []xdr.SorobanAuthorizationEntry{}
	txMetaGetChAfterFunding, err := aliceClient.InvokeAndProcessHostFunction(hzAliceGetCh, "get_channel", getChannelArgs, contractAddr, kpAlice, auth)
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

	auth = []xdr.SorobanAuthorizationEntry{}
	invokeHostFunctionOpClose := env.BuildContractCallOp(hzAliceGClose, "close", closeArgs, contractAddr, auth)

	preFlightOpClose, minFeeClose := itest.PreflightHostFunctions(&hzAliceGClose, *invokeHostFunctionOpClose)

	txClose, err := itest.SubmitOperationsWithFee(&hzAliceGClose, kpAlice, minFeeClose, &preFlightOpClose)

	require.NoError(t, err)

	txMetaClose, err := env.DecodeTxMeta(txClose)

	require.NoError(t, err)

	_, err = channel.DecodeEventsPerun(txMetaClose)
	require.NoError(t, err)

	hzAliceWithdraw := aliceClient.GetHorizonAcc()
	waArgs, err := env.BuildFundTxArgs(chanID, false)
	require.NoError(t, err)
	auth = []xdr.SorobanAuthorizationEntry{}
	invokeHostFunctionOpWA := env.BuildContractCallOp(hzAliceWithdraw, "withdraw", waArgs, contractAddr, auth)
	preFlightOpWA, minFeeWA := itest.PreflightHostFunctions(&hzAliceWithdraw, *invokeHostFunctionOpWA)
	txWithdrawAlice, err := itest.SubmitOperationsWithFee(&hzAliceWithdraw, kpAlice, minFeeWA, &preFlightOpWA)
	require.NoError(t, err)

	txMetaWA, err := env.DecodeTxMeta(txWithdrawAlice)

	require.NoError(t, err)
	_, err = channel.DecodeEventsPerun(txMetaWA)
	require.NoError(t, err)

	hzBobWithdraw := bobClient.GetHorizonAcc()
	wbArgs, err := env.BuildFundTxArgs(chanID, true)
	require.NoError(t, err)
	auth = []xdr.SorobanAuthorizationEntry{}
	invokeHostFunctionOpWB := env.BuildContractCallOp(hzBobWithdraw, "withdraw", wbArgs, contractAddr, auth)
	preFlightOpWB, minFeeWB := itest.PreflightHostFunctions(&hzBobWithdraw, *invokeHostFunctionOpWB)
	txWithdrawBob, err := itest.SubmitOperationsWithFee(&hzBobWithdraw, kpBob, minFeeWB, &preFlightOpWB)
	require.NoError(t, err)

	txMetaWB, err := env.DecodeTxMeta(txWithdrawBob)

	require.NoError(t, err)
	_, err = channel.DecodeEventsPerun(txMetaWB)
	require.NoError(t, err)

}
