package client

import (
	"context"
	"errors"
	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/jhttp"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"math/big"
	"perun.network/perun-stellar-backend/wire"
	"time"
)

const (
	sorobanRPCLink = "http://localhost:8000/soroban/rpc"
	sorobanTestnet = "https://soroban-testnet.stellar.org"
)

func (st *StellarSigner) createSignedTxFromParams(txParams txnbuild.TransactionParams) (*txnbuild.Transaction, error) {

	txUnsigned, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return nil, err
	}

	tx, err := txUnsigned.Sign(NETWORK_PASSPHRASE, st.keyPair)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func DecodeTxMeta(tx horizon.Transaction) (xdr.TransactionMeta, error) {
	var transactionMeta xdr.TransactionMeta
	err := xdr.SafeUnmarshalBase64(tx.ResultMetaXdr, &transactionMeta)
	if err != nil {
		return xdr.TransactionMeta{}, err
	}

	return transactionMeta, nil
}

func BuildContractCallOp(caller horizon.Account, fName xdr.ScSymbol, callArgs xdr.ScVec, contractIdAddress xdr.ScAddress) *txnbuild.InvokeHostFunction {

	return &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.InvokeContractArgs{
				ContractAddress: contractIdAddress,
				FunctionName:    fName,
				Args:            callArgs,
			},
		},
		SourceAccount: caller.AccountID,
	}
}

type RPCSimulateTxResponse struct {
	Error           string                          `json:"error,omitempty"`
	TransactionData string                          `json:"transactionData"`
	Results         []RPCSimulateHostFunctionResult `json:"results"`
	MinResourceFee  int64                           `json:"minResourceFee,string"`
}

type RPCSimulateHostFunctionResult struct {
	Auth []string `json:"auth"`
	XDR  string   `json:"xdr"`
}

func PreflightHostFunctions(hzClient *horizonclient.Client,
	sourceAccount txnbuild.Account, function txnbuild.InvokeHostFunction,
) (txnbuild.InvokeHostFunction, int64, error) {
	result, transactionData, err := simulateTransaction(hzClient, sourceAccount, &function)
	if err != nil {
		return txnbuild.InvokeHostFunction{}, 0, err
	}

	function.Ext = xdr.TransactionExt{
		V:           1,
		SorobanData: &transactionData,
	}
	var funAuth []xdr.SorobanAuthorizationEntry
	for _, res := range result.Results {
		var decodedRes xdr.ScVal
		err := xdr.SafeUnmarshalBase64(res.XDR, &decodedRes)
		if err != nil {
			return txnbuild.InvokeHostFunction{}, 0, err
		}
		for _, authBase64 := range res.Auth {
			var authEntry xdr.SorobanAuthorizationEntry
			err = xdr.SafeUnmarshalBase64(authBase64, &authEntry)
			if err != nil {
				return txnbuild.InvokeHostFunction{}, 0, err
			}
			funAuth = append(funAuth, authEntry)
		}
	}
	function.Auth = funAuth

	return function, result.MinResourceFee, nil
}

func PreflightHostFunctionsResult(hzClient *horizonclient.Client,
	sourceAccount txnbuild.Account, function txnbuild.InvokeHostFunction, chInfo bool,
) (wire.Channel, string, txnbuild.InvokeHostFunction, int64, error) {
	result, transactionData, err := simulateTransaction(hzClient, sourceAccount, &function)
	if err != nil {
		return wire.Channel{}, "", txnbuild.InvokeHostFunction{}, 0, err
	}

	function.Ext = xdr.TransactionExt{
		V:           1,
		SorobanData: &transactionData,
	}
	var getChan wire.Channel

	if len(result.Results) != 1 {
		return wire.Channel{}, "", function, result.MinResourceFee, errors.New("Invalid number of results")
	}

	var decodedXdr xdr.ScVal
	err = xdr.SafeUnmarshalBase64(result.Results[0].XDR, &decodedXdr)
	if err != nil {
		return wire.Channel{}, "", function, result.MinResourceFee, err
	}
	if chInfo {
		decChanInfo := decodedXdr

		if decChanInfo.Type != xdr.ScValTypeScvMap {
			return getChan, "", function, result.MinResourceFee, errors.New("Invalid channel info type")
		}

		err = getChan.FromScVal(decChanInfo)
		if err != nil {
			return getChan, "", function, result.MinResourceFee, err
		}

		return getChan, "", function, result.MinResourceFee, nil
	} else {
		i128 := decodedXdr.MustI128()
		hi := big.NewInt(int64(i128.Hi))
		lo := big.NewInt(int64(i128.Lo))

		// Combine hi and lo into a single big.Int
		combined := hi.Lsh(hi, 64).Or(hi, lo)
		return wire.Channel{}, combined.String(), function, result.MinResourceFee, nil
	}

}

func simulateTransaction(hzClient *horizonclient.Client,
	sourceAccount txnbuild.Account, op txnbuild.Operation,
) (RPCSimulateTxResponse, xdr.SorobanTransactionData, error) {
	// Before preflighting, make sure soroban-rpc is in sync with Horizon
	root, err := hzClient.Root()
	if err != nil {
		return RPCSimulateTxResponse{}, xdr.SorobanTransactionData{}, err
	}
	err = syncWithSorobanRPC(uint32(root.HorizonSequence))
	if err != nil {
		return RPCSimulateTxResponse{}, xdr.SorobanTransactionData{}, err
	}

	ch := jhttp.NewChannel(sorobanTestnet, nil)
	sorobanRPCClient := jrpc2.NewClient(ch, nil)
	txParams := GetBaseTransactionParamsWithFee(sourceAccount, txnbuild.MinBaseFee, op)
	txParams.IncrementSequenceNum = false
	tx, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return RPCSimulateTxResponse{}, xdr.SorobanTransactionData{}, err
	}
	base64, err := tx.Base64()
	if err != nil {
		return RPCSimulateTxResponse{}, xdr.SorobanTransactionData{}, err
	}
	result := RPCSimulateTxResponse{}
	err = sorobanRPCClient.CallResult(context.Background(), "simulateTransaction", struct {
		Transaction string `json:"transaction"`
	}{base64}, &result)
	if err != nil {
		return RPCSimulateTxResponse{}, xdr.SorobanTransactionData{}, err
	}
	var transactionData xdr.SorobanTransactionData
	err = xdr.SafeUnmarshalBase64(result.TransactionData, &transactionData)
	if err != nil {
		return RPCSimulateTxResponse{}, xdr.SorobanTransactionData{}, err
	}
	return result, transactionData, nil
}
func syncWithSorobanRPC(ledgerToWaitFor uint32) error {
	for j := 0; j < 20; j++ {
		result := struct {
			Sequence uint32 `json:"sequence"`
		}{}
		ch := jhttp.NewChannel(sorobanTestnet, nil)
		sorobanRPCClient := jrpc2.NewClient(ch, nil)
		err := sorobanRPCClient.CallResult(context.Background(), "getLatestLedger", nil, &result)
		if err != nil {
			return err
		}
		if result.Sequence >= ledgerToWaitFor {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return errors.New("Time out waiting for soroban-rpc to sync")
}

func GetBaseTransactionParamsWithFee(source txnbuild.Account, fee int64, ops ...txnbuild.Operation) txnbuild.TransactionParams {
	return txnbuild.TransactionParams{
		SourceAccount:        source,
		Operations:           ops,
		BaseFee:              fee,
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewInfiniteTimeout()},
		IncrementSequenceNum: true,
	}
}
