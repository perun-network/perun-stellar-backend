package client

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

func CreateSignedTransactionWithParams(signers []*keypair.Full, txParams txnbuild.TransactionParams,
) (*txnbuild.Transaction, error) {
	tx, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return nil, err
	}

	for _, signer := range signers {
		tx, err = tx.Sign(NETWORK_PASSPHRASE, signer)
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

type TxSender struct {
	kp       *keypair.Full
	hzClient *horizonclient.Client
}

func NewSender(kp *keypair.Full) Sender {
	return &TxSender{kp: kp}

}

func (s *TxSender) SignSendTx(txUnsigned txnbuild.Transaction) (xdr.TransactionMeta, error) {
	tx, err := txUnsigned.Sign(NETWORK_PASSPHRASE, s.kp)
	if err != nil {
		return xdr.TransactionMeta{}, err

	}

	txSent, err := s.hzClient.SubmitTransaction(tx)
	if err != nil {
		return xdr.TransactionMeta{}, err
	}
	txMeta, err := DecodeTxMeta(txSent)
	if err != nil {
		return xdr.TransactionMeta{}, ErrCouldNotDecodeTxMeta
	}
	_ = txMeta.V3.SorobanMeta.ReturnValue

	return txMeta, nil

}

func (s *TxSender) SetHzClient(hzClient *horizonclient.Client) {
	s.hzClient = hzClient
}
