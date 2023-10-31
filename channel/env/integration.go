package env

import (
	"errors"
	"fmt"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	testenv "github.com/stellar/go/services/horizon/pkg/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"testing"
)

const PtotocolVersion = 20
const EnableSorobanRPC = true
const StandaloneAuth = "Standalone Network ; February 2017"
const SorobanRPCPort = 8080

type IntegrationTestEnv struct {
	testEnv           *testenv.Test
	contractIDAddress xdr.ScAddress
	stellarClient     *StellarClient
}

func NewBackendEnv() *IntegrationTestEnv {
	// Create a dummy testing.T object.
	// This might not be appropriate depending on your use case.
	t := &testing.T{}

	cfg := testenv.Config{
		ProtocolVersion:  PtotocolVersion,
		EnableSorobanRPC: EnableSorobanRPC,
	}
	itest := IntegrationTestEnv{testEnv: testenv.NewTest(t, cfg)}
	return &itest
}

func NewIntegrationEnv(t *testing.T) *IntegrationTestEnv {
	cfg := testenv.Config{ProtocolVersion: PtotocolVersion,
		EnableSorobanRPC: EnableSorobanRPC,
	}
	itest := IntegrationTestEnv{testEnv: testenv.NewTest(t, cfg)}
	return &itest
}

// func (it *IntegrationTestEnv) GetStellarClient() *StellarClient {
// 	return it.stellarClient
// }

func (it *IntegrationTestEnv) CreateAccounts(numAccounts int, initBalance string) ([]*keypair.Full, []txnbuild.Account) {
	kps, accs := it.testEnv.CreateAccounts(numAccounts, initBalance)

	return kps, accs
}

func (it *IntegrationTestEnv) AccountDetails(acc *keypair.Full) horizon.Account {
	accountReq := horizonclient.AccountRequest{AccountID: acc.Address()}
	hzAccount, err := it.testEnv.Client().AccountDetail(accountReq)
	if err != nil {
		panic(err)
	}
	return hzAccount
}

func (it *IntegrationTestEnv) PreflightHostFunctions(sourceAccount txnbuild.Account, function txnbuild.InvokeHostFunction) (txnbuild.InvokeHostFunction, int64) {
	return it.testEnv.PreflightHostFunctions(sourceAccount, function)
}

func (it *IntegrationTestEnv) MustSubmitOperationsWithFee(
	source txnbuild.Account,
	signer *keypair.Full,
	fee int64,
	ops ...txnbuild.Operation,
) horizon.Transaction {
	return it.testEnv.MustSubmitOperationsWithFee(source, signer, fee, ops...)
}

func (it *IntegrationTestEnv) SubmitOperationsWithFee(
	source txnbuild.Account,
	signer *keypair.Full,
	fee int64,
	ops ...txnbuild.Operation,
) (horizon.Transaction, error) {
	return it.testEnv.SubmitOperationsWithFee(source, signer, fee, ops...)
}

func (it *IntegrationTestEnv) Client() *horizonclient.Client {
	return it.testEnv.Client()
}

func (it *IntegrationTestEnv) GetPassPhrase() string {
	return it.testEnv.GetPassPhrase()
}

func (it *IntegrationTestEnv) GetContractIDAddress() xdr.ScAddress {
	return it.contractIDAddress
}

func (it *IntegrationTestEnv) InvokeAndProcessHostFunction(horizonAcc horizon.Account, fname string, callTxArgs xdr.ScVec, contractAddr xdr.ScAddress, kp *keypair.Full) (xdr.TransactionMeta, error) {
	// Build contract call operation
	fnameXdr := xdr.ScSymbol(fname)
	invokeHostFunctionOp := BuildContractCallOp(horizonAcc, fnameXdr, callTxArgs, contractAddr)

	// Preflight host functions
	preFlightOp, minFee := it.PreflightHostFunctions(&horizonAcc, *invokeHostFunctionOp)

	// Submit operations with fee
	tx, err := it.SubmitOperationsWithFee(&horizonAcc, kp, minFee, &preFlightOp)
	if err != nil {
		return xdr.TransactionMeta{}, errors.New("error while submitting operations with fee")
	}

	// Decode transaction metadata
	txMeta, err := DecodeTxMeta(tx)
	if err != nil {
		return xdr.TransactionMeta{}, errors.New("error while decoding tx meta")
	}

	fmt.Println("txMeta: ", txMeta)

	// // Decode events
	// _, err = DecodeSorEvents(txMeta)
	// if err != nil {
	// 	return errors.New("error while decoding events")
	// }

	return txMeta, nil
}

func (it *IntegrationTestEnv) GetChannelState(getChanArgs xdr.ScVec) (xdr.TransactionMeta, error) {
	// query channel state
	kp := it.stellarClient.account

	acc := it.AccountDetails(kp)
	chanState, err := it.InvokeAndProcessHostFunction(acc, "get_channel", getChanArgs, xdr.ScAddress{}, nil)
	if err != nil {
		return xdr.TransactionMeta{}, err
	}
	return chanState, nil
}
