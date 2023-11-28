package channel_test

import (
	"context"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"
	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/env"
	chtest "perun.network/perun-stellar-backend/channel/test"
	"perun.network/perun-stellar-backend/util"
	_ "perun.network/perun-stellar-backend/wallet/test"
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

	_, err = channel.DecodeEvents(txMeta)
	require.NoError(t, err)

}

func TestFundChannel(t *testing.T) {
	itest := env.NewBackendEnv()

	kps, _ := itest.CreateAccounts(3, "1000000000")

	kpAlice := kps[0]
	kpBob := kps[1]
	kpDeployer := kps[2]

	_ = itest.AccountDetails(kpAlice)
	_ = itest.AccountDetails(kpBob)
	reqDeployer := itest.AccountDetails(kpDeployer)

	contractAddr := util.Deploy(itest, kpDeployer, reqDeployer, PerunContractPath)
	itest.SetContractIDAddress(contractAddr)

	perunFirstParams, perunFirstState := chtest.NewParamsState(t)

	_, accAlice, _ := util.MakeRandPerunWallet()
	_, accBob, _ := util.MakeRandPerunWallet()

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
	aliceIdx := false
	_, err := env.BuildFundTxArgs(chanID, aliceIdx)
	require.NoError(t, err)

	// Calling the function
	err = chtest.FundAll(context.TODO(), funders, freqs)
	require.NoError(t, err)

}
