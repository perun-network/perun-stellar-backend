package env

import (
	//"context"
	"errors"
	"fmt"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/xdr"
	//pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/wire"
)

type StellarClient struct {
	stellarEnv        *IntegrationTestEnv
	kp                *keypair.Full
	passphrase        string
	contractIDAddress xdr.ScAddress
}

func NewStellarClient(stellarEnv *IntegrationTestEnv, kp *keypair.Full) *StellarClient {
	passphrase := stellarEnv.GetPassPhrase()
	return &StellarClient{
		stellarEnv:        stellarEnv,
		kp:                kp,
		passphrase:        passphrase,
		contractIDAddress: stellarEnv.contractIDAddress,
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

func (s *StellarClient) GetContractIDAddress() xdr.ScAddress {
	return s.contractIDAddress
}

func (s *StellarClient) InvokeAndProcessHostFunction(horizonAcc horizon.Account, fname string, callTxArgs xdr.ScVec, contractAddr xdr.ScAddress, kp *keypair.Full) (xdr.TransactionMeta, error) {

	// Build contract call operation
	fnameXdr := xdr.ScSymbol(fname)
	fmt.Println("contractAddr in InvokeAndProcessHostFunction: ", contractAddr)
	invokeHostFunctionOp := BuildContractCallOp(horizonAcc, fnameXdr, callTxArgs, contractAddr)

	// Preflight host functions
	fmt.Println("horizonAcc in InvokeAndProcessHostFunction: ", horizonAcc)
	preFlightOp, minFee := s.stellarEnv.PreflightHostFunctions(&horizonAcc, *invokeHostFunctionOp)
	fmt.Println("minfee for", fname, ": ", minFee)
	// Submit operations with fee
	tx, err := s.stellarEnv.SubmitOperationsWithFee(&horizonAcc, kp, minFee, &preFlightOp)
	if err != nil {
		panic(err)
		//return xdr.TransactionMeta{}, errors.New("error while submitting operations with fee")
	}

	fmt.Println("tx from ", fname, ": ", tx)

	// Decode transaction metadata
	txMeta, err := DecodeTxMeta(tx)
	if err != nil {
		return xdr.TransactionMeta{}, errors.New("error while decoding tx meta")
	}

	fmt.Println("txMeta from ", fname, ": ", txMeta)

	// // Decode events
	// _, err = DecodeSorEvents(txMeta)
	// if err != nil {
	// 	return errors.New("error while decoding events")
	// }

	return txMeta, nil
}

func (s *StellarClient) GetChannelState(chanArgs xdr.ScVec) (wire.Channel, error) {
	fmt.Println("in GetChannelState")
	contractAddress := s.stellarEnv.GetContractIDAddress()
	kp := s.kp
	hz := s.GetHorizonAcc()

	txMeta, err := s.InvokeAndProcessHostFunction(hz, "get_channel", chanArgs, contractAddress, kp)
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
