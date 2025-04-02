package client

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

// CreateSignedTransactionWithParams creates a signed transaction with the given signers and transaction parameters.
func CreateSignedTransactionWithParams(signers []*keypair.Full, txParams txnbuild.TransactionParams, passphrase string,
) (*txnbuild.Transaction, error) {
	tx, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return nil, err
	}

	for _, signer := range signers {
		tx, err = tx.Sign(passphrase, signer)
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

// TxSender is a struct that implements the Sender interface.
type TxSender struct {
	kp       *keypair.Full
	hzClient *horizonclient.Client
}

// NewSender creates a new TxSender.
func NewSender(kp *keypair.Full, hzClient *horizonclient.Client) Sender {
	return &TxSender{kp: kp, hzClient: hzClient}
}

// SetHzClient sets the horizon client.
func (s *TxSender) SetHzClient(hzClient *horizonclient.Client) {
	s.hzClient = hzClient
}

// SignSendTx signs and sends the transaction.
func (s *TxSender) SignSendTx(txUnsigned txnbuild.Transaction) (xdr.TransactionMeta, error) {
	var passphrase string
	if s.hzClient.HorizonURL == horizonClientURL {
		passphrase = NETWORK_PASSPHRASE
	} else {
		passphrase = NETWORK_PASSPHRASETestNet
	}
	tx, err := txUnsigned.Sign(passphrase, s.kp)
	if err != nil {
		return xdr.TransactionMeta{}, err
	}

	txSent, err := s.hzClient.SubmitTransaction(tx)
	if err != nil {
		return xdr.TransactionMeta{}, err
	}
	txMeta, err := DecodeTxMeta(txSent, s.hzClient)
	if err != nil {
		return xdr.TransactionMeta{}, ErrCouldNotDecodeTxMeta
	}
	_ = txMeta.V3.SorobanMeta.ReturnValue

	return txMeta, nil
}
