package env

import (
	"errors"
	"fmt"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/xdr"
)

type StellarClient struct {
	stellarEnv        *IntegrationTestEnv
	account           *keypair.Full
	hzAccount         horizon.Account
	passphrase        string
	contractIDAddress xdr.ScAddress
}

func NewStellarClient(stellarEnv *IntegrationTestEnv, account *keypair.Full) *StellarClient {
	passphrase := stellarEnv.GetPassPhrase()
	return &StellarClient{
		stellarEnv: stellarEnv,
		account:    account,
		passphrase: passphrase,
	}
}

func (s StellarClient) GetAccount() *keypair.Full {
	return s.account
}
func (s StellarClient) GetHorizonAcc() horizon.Account {
	return s.hzAccount
}

func (s StellarClient) GetPassPhrase() string {
	return s.passphrase
}

func (s StellarClient) GetContractIDAddress() xdr.ScAddress {
	return s.contractIDAddress
}

func (s StellarClient) InvokeAndProcessHostFunction(horizonAcc horizon.Account, fname string, callTxArgs xdr.ScVec, contractAddr xdr.ScAddress, kp *keypair.Full) (xdr.TransactionMeta, error) {
	// Build contract call operation
	fnameXdr := xdr.ScSymbol(fname)
	invokeHostFunctionOp := BuildContractCallOp(horizonAcc, fnameXdr, callTxArgs, contractAddr)

	// Preflight host functions
	preFlightOp, minFee := s.stellarEnv.PreflightHostFunctions(&horizonAcc, *invokeHostFunctionOp)

	// Submit operations with fee
	tx, err := s.stellarEnv.SubmitOperationsWithFee(&horizonAcc, kp, minFee, &preFlightOp)
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
