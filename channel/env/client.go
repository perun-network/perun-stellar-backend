package env

import (
	"errors"
	"fmt"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/xdr"
	"perun.network/perun-stellar-backend/wire"
)

type StellarClient struct {
	stellarEnv   *IntegrationTestEnv
	kp           *keypair.Full
	passphrase   string
	perunAddress xdr.ScAddress
	tokenAddress xdr.ScAddress
}

func NewStellarClient(stellarEnv *IntegrationTestEnv, kp *keypair.Full) *StellarClient {
	passphrase := stellarEnv.GetPassPhrase()
	return &StellarClient{
		stellarEnv:   stellarEnv,
		kp:           kp,
		passphrase:   passphrase,
		perunAddress: stellarEnv.perunAddress,
		tokenAddress: stellarEnv.tokenAddress,
	}

}

func (s *StellarClient) GetAccount() *keypair.Full {
	return s.kp
}
func (s *StellarClient) GetHorizonAcc() horizon.Account {
	return s.stellarEnv.AccountDetails(s.kp)
}

func (s *StellarClient) GetPassPhrase() string {
	return s.passphrase
}

func (s *StellarClient) GetPerunAddress() xdr.ScAddress {
	return s.perunAddress
}

func (s *StellarClient) GetTokenAddress() xdr.ScAddress {
	return s.tokenAddress
}

func (s *StellarClient) InvokeAndProcessHostFunction(horizonAcc horizon.Account, fname string, callTxArgs xdr.ScVec, contractAddr xdr.ScAddress, kp *keypair.Full, auth []xdr.SorobanAuthorizationEntry) (xdr.TransactionMeta, error) {

	// Build contract call operation
	fnameXdr := xdr.ScSymbol(fname)

	invokeHostFunctionOp := BuildContractCallOp(horizonAcc, fnameXdr, callTxArgs, contractAddr, auth)

	preFlightOp, minFee := s.stellarEnv.PreflightHostFunctions(&horizonAcc, *invokeHostFunctionOp)

	tx, err := s.stellarEnv.SubmitOperationsWithFee(&horizonAcc, kp, minFee, &preFlightOp)
	if err != nil {
		panic(err)
	}

	fmt.Println("tx from ", fname, ": ", tx)

	// Decode transaction metadata
	txMeta, err := DecodeTxMeta(tx)
	if err != nil {
		return xdr.TransactionMeta{}, errors.New("error while decoding tx meta")
	}

	return txMeta, nil
}

func (s *StellarClient) GetChannelState(chanArgs xdr.ScVec) (wire.Channel, error) {
	contractAddress := s.stellarEnv.GetPerunAddress()
	kp := s.kp
	hz := s.GetHorizonAcc()
	auth := []xdr.SorobanAuthorizationEntry{}
	txMeta, err := s.InvokeAndProcessHostFunction(hz, "get_channel", chanArgs, contractAddress, kp, auth)
	if err != nil {
		return wire.Channel{}, errors.New("error while processing and submitting get_channel tx")
	}

	retVal := txMeta.V3.SorobanMeta.ReturnValue
	var getChan wire.Channel

	err = getChan.FromScVal(retVal)
	if err != nil {
		return wire.Channel{}, errors.New("error while decoding return value")
	}
	return getChan, nil

}
