package client

import (
	"errors"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"

	"github.com/stellar/go/xdr"
	"perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/wallet/types"
	"sync"
)

const stellarDefaultChainId = 1

type Sender interface {
	SignSendTx(txnbuild.Transaction) (xdr.TransactionMeta, error)
}

type ContractBackend struct {
	Invoker
	tr      StellarSigner
	chainID int
	cbMutex sync.Mutex
}

func NewContractBackend(trConfig *TransactorConfig) *ContractBackend {
	transactor := NewTransactor(*trConfig)
	return &ContractBackend{
		tr:      *transactor,
		chainID: stellarDefaultChainId,
		cbMutex: sync.Mutex{},
	}

}

type StellarSigner struct {
	keyPair     *keypair.Full
	participant *types.Participant
	account     *wallet.Account
	hzClient    *horizonclient.Client
	sender      Sender
}

type TransactorConfig struct {
	keyPair     *keypair.Full
	participant *types.Participant
	account     *wallet.Account
	sender      Sender
}

func (tc *TransactorConfig) SetKeyPair(kp *keypair.Full) {
	tc.keyPair = kp
}

func (tc *TransactorConfig) SetParticipant(participant *types.Participant) {
	tc.participant = participant
}

func (tc *TransactorConfig) SetAccount(account *wallet.Account) {
	tc.account = account
}

func (tc *TransactorConfig) SetSender(sender Sender) {
	tc.sender = sender
}

func NewTransactor(cfg TransactorConfig) *StellarSigner {
	st := &StellarSigner{}

	if cfg.sender != nil {
		st.sender = cfg.sender
	} else {
		st.sender = &TxSender{}
	}

	if cfg.keyPair != nil {
		st.keyPair = cfg.keyPair
		if txSender, ok := st.sender.(*TxSender); ok {
			txSender.kp = st.keyPair
		}
	}
	if cfg.participant != nil {
		st.participant = cfg.participant
	}
	if cfg.account != nil {
		st.account = cfg.account
	}

	st.hzClient = NewHorizonClient()

	return st
}

func (cb *ContractBackend) GetTransactor() *StellarSigner {
	return &cb.tr
}

func (st *StellarSigner) GetHorizonAccount() (horizon.Account, error) {
	hzAddress, err := st.GetAddress()
	if err != nil {
		return horizon.Account{}, err
	}
	accountReq := horizonclient.AccountRequest{AccountID: hzAddress}
	hzAccount, err := st.hzClient.AccountDetail(accountReq)
	if err != nil {
		return hzAccount, err
	}
	return hzAccount, nil
}

func (st *StellarSigner) GetAddress() (string, error) {
	if st.keyPair != nil {
		return st.keyPair.Address(), nil
	}
	if st.account != nil {
		return (*st.account).Address().String(), nil
	}
	if st.participant != nil {
		return st.participant.AddressString(), nil
	}

	return "", errors.New("transactor cannot retrieve address")
}

func (st *StellarSigner) GetHorizonClient() *horizonclient.Client {
	return st.hzClient
}

func (c *ContractBackend) InvokeSignedTx(fname string, callTxArgs xdr.ScVec, contractAddr xdr.ScAddress) (xdr.TransactionMeta, error) {
	c.cbMutex.Lock()
	defer c.cbMutex.Unlock()
	fnameXdr := xdr.ScSymbol(fname)
	hzAcc, err := c.tr.GetHorizonAccount()
	if err != nil {
		return xdr.TransactionMeta{}, err
	}

	hzClient := c.tr.GetHorizonClient()

	txSender, ok := c.tr.sender.(*TxSender)
	if !ok {
		return xdr.TransactionMeta{}, errors.New("sender is not of type *TxSender")
	}

	txSender.SetHzClient(hzClient)

	invokeHostFunctionOp := BuildContractCallOp(hzAcc, fnameXdr, callTxArgs, contractAddr)
	preFlightOp, _ := PreflightHostFunctions(hzClient, &hzAcc, *invokeHostFunctionOp)
	minFeeCustom := int64(500000)
	txParams := GetBaseTransactionParamsWithFee(&hzAcc, minFeeCustom, &preFlightOp)
	txUnsigned, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return xdr.TransactionMeta{}, err
	}
	txMeta, err := c.tr.sender.SignSendTx(*txUnsigned)

	if err != nil {
		return xdr.TransactionMeta{}, err
	}

	return txMeta, nil
}
