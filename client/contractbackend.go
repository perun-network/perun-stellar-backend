package client

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/wallet"
	chTypes "perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wallet/types"
	"sync"
)

const stellarDefaultChainId = 1

type Sender interface {
	SignSendTx(txnbuild.Transaction) (xdr.TransactionMeta, error)
	SetHzClient(*horizonclient.Client)
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

func (cb *ContractBackend) GetBalance(cID xdr.ScAddress) (string, error) {
	tr := cb.GetTransactor()
	add, err := tr.GetAddress()
	if err != nil {
		return "", err
	}
	accountId, err := xdr.AddressToAccountId(add)
	if err != nil {
		return "", err
	}
	scAdd, err := xdr.NewScAddress(xdr.ScAddressTypeScAddressTypeAccount, accountId)
	if err != nil {
		return "", err
	}
	TokenNameArgs, err := BuildGetTokenBalanceArgs(scAdd)
	if err != nil {
		return "", err
	}
	tx, err := cb.InvokeSignedTx("balance", TokenNameArgs, cID)
	if err != nil {
		return "", err
	}
	return tx.V3.SorobanMeta.ReturnValue.String(), nil
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

	c.tr.sender.SetHzClient(hzClient)
	invokeHostFunctionOp := BuildContractCallOp(hzAcc, fnameXdr, callTxArgs, contractAddr)
	preFlightOp, minFee := PreflightHostFunctions(hzClient, &hzAcc, *invokeHostFunctionOp)

	txParams := GetBaseTransactionParamsWithFee(&hzAcc, minFee, &preFlightOp)
	txUnsigned, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return xdr.TransactionMeta{}, err
	}
	// txSigned, err := c.tr.createSignedTxFromParams(txParams)
	txMeta, err := c.tr.sender.SignSendTx(*txUnsigned)
	if err != nil {
		return xdr.TransactionMeta{}, err
	}
	// tx, err := hzClient.SubmitTransaction(txSigned)
	// if err != nil {
	// 	return xdr.TransactionMeta{}, err
	// }

	// txMeta, err := DecodeTxMeta(txSigned)
	// if err != nil {
	// 	return xdr.TransactionMeta{}, ErrCouldNotDecodeTxMeta
	// }
	// _ = txMeta.V3.SorobanMeta.ReturnValue

	return txMeta, nil
}

/*
 * StringToScAddress converts a string to a xdr.ScAddress.
 */
func StringToScAddress(s string) (xdr.ScAddress, error) {
	hash, err := StringToHash(s)
	if err != nil {
		return xdr.ScAddress{}, err
	}
	return chTypes.MakeContractAddress(hash)
}

/*
 * StringToHash converts a hex string to a xdr.Hash.
 */
func StringToHash(s string) (xdr.Hash, error) {
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return xdr.Hash{}, fmt.Errorf("failed to decode hex string: %w", err)
	}
	var hash xdr.Hash
	copy(hash[:], bytes)
	return hash, nil
}
