package env

import (
	"errors"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	testenv "github.com/stellar/go/services/horizon/pkg/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"testing"
)

const PtotocolVersion = 20
const EnableSorobanRPC = true
const StandaloneAuth = "Standalone Network ; February 2017"
const SorobanRPCPort = 8080

type IntegrationTestEnv struct {
	testEnv      *testenv.Test
	perunAddress xdr.ScAddress
	tokenAddress xdr.ScAddress
}

func NewBackendEnv() *IntegrationTestEnv {
	t := &testing.T{}

	cfg := testenv.Config{
		ProtocolVersion:    PtotocolVersion,
		EnableSorobanRPC:   EnableSorobanRPC,
		HorizonEnvironment: map[string]string{"INGEST_DISABLE_STATE_VERIFICATION": "true", "CONNECTION_TIMEOUT": "360000"},
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

func (it *IntegrationTestEnv) SetPerunAddress(perunAddress xdr.ScAddress) {
	it.perunAddress = perunAddress
}

func (it *IntegrationTestEnv) SetTokenAddress(tokenAddress xdr.ScAddress) {
	it.tokenAddress = tokenAddress
}

func (it *IntegrationTestEnv) CreateAccounts(numAccounts int, initBalance string) ([]*keypair.Full, []txnbuild.Account) {
	kps, accs := it.testEnv.CreateAccounts(numAccounts, initBalance)

	return kps, accs
}

func (it *IntegrationTestEnv) CreateAccount(initialBalance string) (*keypair.Full, txnbuild.Account) {
	kps, accts := it.CreateAccounts(1, initialBalance)
	return kps[0], accts[0]
}

func (it *IntegrationTestEnv) CurrentTest() *testing.T {
	return it.testEnv.CurrentTest()
}

func (it *IntegrationTestEnv) Master() *keypair.Full {
	return it.testEnv.Master()
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

func (it *IntegrationTestEnv) GetPerunAddress() xdr.ScAddress {
	return it.perunAddress
}

func AssertContainsBalance(it *IntegrationTestEnv, acct *keypair.Full, issuer, code string, amt xdr.Int64) {
	accountResponse := it.testEnv.MustGetAccount(acct)
	if issuer == "" && code == "" {
		xlmBalance, err := accountResponse.GetNativeBalance()
		if err != nil {
			panic(err)
		}
		assert.NoError(it.testEnv.CurrentTest(), err)
		assert.Equal(it.testEnv.CurrentTest(), amt, amount.MustParse(xlmBalance))
	} else {
		assetBalance := accountResponse.GetCreditBalance(code, issuer)
		assert.Equal(it.testEnv.CurrentTest(), amt, amount.MustParse(assetBalance))
	}
}

func (it *IntegrationTestEnv) InvokeAndProcessHostFunction(horizonAcc horizon.Account, fname string, callTxArgs xdr.ScVec, contractAddr xdr.ScAddress, kp *keypair.Full) (xdr.TransactionMeta, error) {
	// Build contract call operation
	fnameXdr := xdr.ScSymbol(fname)
	auth := []xdr.SorobanAuthorizationEntry{}
	invokeHostFunctionOp := BuildContractCallOp(horizonAcc, fnameXdr, callTxArgs, contractAddr, auth)

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

	return txMeta, nil
}

func (it *IntegrationTestEnv) SubmitOperations(source txnbuild.Account, signer *keypair.Full, ops ...txnbuild.Operation) (horizon.Transaction, error) {
	return it.testEnv.SubmitOperations(source, signer, ops...)
}

func (it *IntegrationTestEnv) MustGetAccount(source *keypair.Full) horizon.Account {
	client := it.Client()
	account, err := client.AccountDetail(horizonclient.AccountRequest{AccountID: source.Address()})
	if err != nil {
		panic(err)
	}
	return account
}

func (it *IntegrationTestEnv) MustEstablishTrustline(
	truster *keypair.Full, account txnbuild.Account, asset txnbuild.Asset,
) (resp horizon.Transaction) {
	txResp, err := it.testEnv.EstablishTrustline(truster, account, asset)
	if err != nil {
		panic(err)
	}
	return txResp
}
