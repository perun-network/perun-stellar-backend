package channel_test

import (
	"fmt"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	chtest "perun.network/perun-stellar-backend/channel/test"

	//client "perun.network/perun-stellar-backend/client"
	_ "perun.network/perun-stellar-backend/wallet/test"
	"perun.network/perun-stellar-backend/wire"

	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/env"

	"testing"
)

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

	//clientTx, err := itest.Client().TransactionDetail(tx.Hash)
	//fmt.Println("clientTx: ", clientTx)
	//require.NoError(t, err)

	//assert.Equal(t, tx.Hash, clientTx.Hash)
	//var txResult xdr.TransactionResult
	//err = xdr.SafeUnmarshalBase64(clientTx.ResultXdr, &txResult)
	//require.NoError(t, err)

	//opResults, ok := txResult.OperationResults()

	// assert.True(t, ok)
	// assert.Equal(t, len(opResults), 1)
	// invokeHostFunctionResult, ok := opResults[0].MustTr().GetInvokeHostFunctionResult()
	// assert.True(t, ok)
	//assert.Equal(t, invokeHostFunctionResult.Code, xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess)

	txMeta, err := env.DecodeTxMeta(tx)
	require.NoError(t, err)

	perunEvents, err := channel.DecodeEvents(txMeta)
	require.NoError(t, err)

	fmt.Println("perunEvents: ", perunEvents)

}

func TestFundChannel(t *testing.T) {
	itest := env.NewBackendEnv()

	kps, _ := itest.CreateAccounts(2, "10000000")

	kpAlice := kps[0]
	kpBob := kps[1]

	reqAlice := itest.AccountDetails(kpAlice)
	reqBob := itest.AccountDetails(kpBob)

	fmt.Println("reqAlice, reqBob: ", reqAlice.Balances, reqBob.Balances)

	installContractOp := channel.AssembleInstallContractCodeOp(kpAlice.Address(), channel.PerunContractPath)
	preFlightOp, minFee := itest.PreflightHostFunctions(&reqAlice, *installContractOp)
	_ = itest.MustSubmitOperationsWithFee(&reqAlice, kpAlice, minFee, &preFlightOp)

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

	// invoke open-channel function

	perunFirstParams, perunFirstState := chtest.NewParamsState(t)

	openArgs := env.BuildOpenTxArgs(perunFirstParams, perunFirstState)

	invokeHostFunctionOp := env.BuildContractCallOp(reqAlice, "open", openArgs, contractIDAddress)

	preFlightOp, minFee = itest.PreflightHostFunctions(&reqAlice, *invokeHostFunctionOp)

	tx, err := itest.SubmitOperationsWithFee(&reqAlice, kpAlice, minFee, &preFlightOp)
	require.NoError(t, err)

	clientTx, err := itest.Client().TransactionDetail(tx.Hash)
	fmt.Println("clientTx: ", clientTx)
	require.NoError(t, err)

	assert.Equal(t, tx.Hash, clientTx.Hash)
	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(clientTx.ResultXdr, &txResult)
	require.NoError(t, err)

	opResults, ok := txResult.OperationResults()

	assert.True(t, ok)
	assert.Equal(t, len(opResults), 1)
	invokeHostFunctionResult, ok := opResults[0].MustTr().GetInvokeHostFunctionResult()
	assert.True(t, ok)
	assert.Equal(t, invokeHostFunctionResult.Code, xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess)

	txMeta, err := env.DecodeTxMeta(tx)
	require.NoError(t, err)

	perunEvents, err := channel.DecodeEvents(txMeta)
	require.NoError(t, err)

	fmt.Println("perunEvents: ", perunEvents)

	// fund the channel after it is open

	chanID := perunFirstParams.ID()
	aliceIdx := false

	fundArgsAlice, err := env.BuildFundTxArgs(chanID, aliceIdx)
	require.NoError(t, err)

	invokeHostFunctionOpFund := env.BuildContractCallOp(reqAlice, "fund", fundArgsAlice, contractIDAddress)

	preFlightOpFund, minFeeFund := itest.PreflightHostFunctions(&reqAlice, *invokeHostFunctionOpFund)

	txFund, err := itest.SubmitOperationsWithFee(&reqAlice, kpAlice, minFeeFund, &preFlightOpFund)
	fmt.Println("minFeeFund: ", minFeeFund)
	require.NoError(t, err)

	txMetaFund, err := env.DecodeTxMeta(txFund)
	require.NoError(t, err)

	perunEventsFund, err := channel.DecodeEvents(txMetaFund)
	require.NoError(t, err)
	fmt.Println("perunEventsFund: ", perunEventsFund)

	bobIdx := true
	fundArgsBob, err := env.BuildFundTxArgs(chanID, bobIdx)
	require.NoError(t, err)
	invokeHostFunctionOpFund2 := env.BuildContractCallOp(reqBob, "fund", fundArgsBob, contractIDAddress)

	preFlightOpFund2, minFeeFund2 := itest.PreflightHostFunctions(&reqBob, *invokeHostFunctionOpFund2)
	fmt.Println("minFeeFund2: ", minFeeFund2)

	txFund2, err := itest.SubmitOperationsWithFee(&reqBob, kpBob, minFeeFund2, &preFlightOpFund2)

	require.NoError(t, err)
	txMetaFund2, err := env.DecodeTxMeta(txFund2)
	require.NoError(t, err)
	fmt.Println("txMetaFund2: ", txMetaFund2)
	perunEventsFund2, err := channel.DecodeEvents(txMetaFund2)
	require.NoError(t, err)
	fmt.Println("perunEventsFund2: ", perunEventsFund2)

	// qurey channel state

	getChannelArgs, err := env.BuildGetChannelTxArgs(chanID)
	require.NoError(t, err)

	invokeHostFunctionOpGetCh := env.BuildContractCallOp(reqAlice, "get_channel", getChannelArgs, contractIDAddress)
	preFlightOpGetCh, minFeeGetCh := itest.PreflightHostFunctions(&reqAlice, *invokeHostFunctionOpGetCh)
	txGetCh, err := itest.SubmitOperationsWithFee(&reqAlice, kpAlice, minFeeGetCh, &preFlightOpGetCh)
	require.NoError(t, err)

	txMetaGetCh, err := env.DecodeTxMeta(txGetCh)
	require.NoError(t, err)
	fmt.Println("txMetaGetCh: ", txMetaGetCh)
	retVal := txMetaGetCh.V3.SorobanMeta.ReturnValue
	fmt.Println("retVal: ", retVal)

	var getChan wire.Channel

	err = getChan.FromScVal(retVal)
	require.NoError(t, err)
	fmt.Println("getChan: ", getChan)
}