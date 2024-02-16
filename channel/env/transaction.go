// Copyright 2023 PolyCrypt GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package env

import (
	"context"
	"errors"
	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/jhttp"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/wire"
	"perun.network/perun-stellar-backend/wire/scval"
	"strconv"
	"sync"
	"time"
)

var sharedMtx sync.Mutex

const sorobanRPCPort = 8000

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
) (txnbuild.InvokeHostFunction, int64) {
	result, transactionData := simulateTransaction(hzClient, sourceAccount, &function)

	function.Ext = xdr.TransactionExt{
		V:           1,
		SorobanData: &transactionData,
	}
	var funAuth []xdr.SorobanAuthorizationEntry
	for _, res := range result.Results {
		var decodedRes xdr.ScVal
		err := xdr.SafeUnmarshalBase64(res.XDR, &decodedRes)
		if err != nil {
			panic(err)
		}
		for _, authBase64 := range res.Auth {
			var authEntry xdr.SorobanAuthorizationEntry
			err = xdr.SafeUnmarshalBase64(authBase64, &authEntry)
			if err != nil {
				panic(err)
			}
			funAuth = append(funAuth, authEntry)
		}
	}
	function.Auth = funAuth

	return function, result.MinResourceFee
}

func simulateTransaction(hzClient *horizonclient.Client,
	sourceAccount txnbuild.Account, op txnbuild.Operation,
) (RPCSimulateTxResponse, xdr.SorobanTransactionData) {
	// Before preflighting, make sure soroban-rpc is in sync with Horizon
	root, err := hzClient.Root()
	if err != nil {
		panic(err)
	}
	syncWithSorobanRPC(uint32(root.HorizonSequence))

	ch := jhttp.NewChannel("http://localhost:"+strconv.Itoa(sorobanRPCPort)+"/soroban/rpc", nil)
	sorobanRPCClient := jrpc2.NewClient(ch, nil)
	txParams := GetBaseTransactionParamsWithFee(sourceAccount, txnbuild.MinBaseFee, op)
	txParams.IncrementSequenceNum = false
	tx, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		panic(err)
	}
	base64, err := tx.Base64()
	if err != nil {
		panic(err)
	}
	result := RPCSimulateTxResponse{}
	err = sorobanRPCClient.CallResult(context.Background(), "simulateTransaction", struct {
		Transaction string `json:"transaction"`
	}{base64}, &result)
	if err != nil {
		panic(err)
	}
	var transactionData xdr.SorobanTransactionData
	err = xdr.SafeUnmarshalBase64(result.TransactionData, &transactionData)
	if err != nil {
		panic(err)
	}
	return result, transactionData
}
func syncWithSorobanRPC(ledgerToWaitFor uint32) {
	for j := 0; j < 20; j++ {
		result := struct {
			Sequence uint32 `json:"sequence"`
		}{}
		ch := jhttp.NewChannel("http://localhost:"+strconv.Itoa(sorobanRPCPort)+"/soroban/rpc", nil) ///soroban/rpc:
		sorobanRPCClient := jrpc2.NewClient(ch, nil)
		err := sorobanRPCClient.CallResult(context.Background(), "getLatestLedger", nil, &result)
		if err != nil {
			panic(err)
		}
		if result.Sequence >= ledgerToWaitFor {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	panic("Time out waiting for soroban-rpc to sync")
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

func BuildOpenTxArgs(params *pchannel.Params, state *pchannel.State) (xdr.ScVec, error) {
	paramsStellar, err := wire.MakeParams(*params)
	if err != nil {
		return xdr.ScVec{}, err
	}
	stateStellar, err := wire.MakeState(*state)
	if err != nil {
		return xdr.ScVec{}, err
	}
	paramsXdr, err := paramsStellar.ToScVal()
	if err != nil {
		return xdr.ScVec{}, err
	}
	stateXdr, err := stateStellar.ToScVal()
	if err != nil {
		return xdr.ScVec{}, err
	}
	openArgs := xdr.ScVec{
		paramsXdr,
		stateXdr,
	}
	return openArgs, nil
}

func BuildMintTokenArgs(mintTo xdr.ScAddress, amount xdr.ScVal) (xdr.ScVec, error) {

	mintToSc, err := scval.WrapScAddress(mintTo)
	if err != nil {
		return xdr.ScVec{}, err
	}

	MintTokenArgs := xdr.ScVec{
		mintToSc,
		amount,
	}

	return MintTokenArgs, nil
}

func BuildGetTokenBalanceArgs(balanceOf xdr.ScAddress) (xdr.ScVec, error) {

	balanceOfSc, err := scval.WrapScAddress(balanceOf)
	if err != nil {
		return xdr.ScVec{}, err
	}

	GetTokenBalanceArgs := xdr.ScVec{
		balanceOfSc,
	}

	return GetTokenBalanceArgs, nil
}

func BuildFundTxArgs(chanID pchannel.ID, funderIdx bool) (xdr.ScVec, error) {

	chanIDStellar := chanID[:]
	var chanid xdr.ScBytes
	copy(chanid[:], chanIDStellar)
	channelID, err := scval.WrapScBytes(chanIDStellar)
	if err != nil {
		return xdr.ScVec{}, err
	}

	userIdStellar, err := scval.WrapBool(funderIdx)
	if err != nil {
		return xdr.ScVec{}, err
	}

	fundArgs := xdr.ScVec{
		channelID,
		userIdStellar,
	}
	return fundArgs, nil
}

func BuildGetChannelTxArgs(chanID pchannel.ID) (xdr.ScVec, error) {

	chanIDStellar := chanID[:]
	var chanid xdr.ScBytes
	copy(chanid[:], chanIDStellar)
	channelID, err := scval.WrapScBytes(chanIDStellar)
	if err != nil {
		return xdr.ScVec{}, err
	}

	getChannelArgs := xdr.ScVec{
		channelID,
	}
	return getChannelArgs, nil
}

func BuildForceCloseTxArgs(chanID pchannel.ID) (xdr.ScVec, error) {

	chanIDStellar := chanID[:]
	var chanid xdr.ScBytes
	copy(chanid[:], chanIDStellar)
	channelID, err := scval.WrapScBytes(chanIDStellar)
	if err != nil {
		return xdr.ScVec{}, err
	}

	getChannelArgs := xdr.ScVec{
		channelID,
	}
	return getChannelArgs, nil
}

func (s *StellarClient) InvokeAndProcessHostFunction(fname string, callTxArgs xdr.ScVec, contractAddr xdr.ScAddress) (xdr.TransactionMeta, error) {
	sharedMtx.Lock()
	defer sharedMtx.Unlock()
	fnameXdr := xdr.ScSymbol(fname)
	hzAcc, err := s.GetHorizonAccount()
	if err != nil {
		return xdr.TransactionMeta{}, err
	}
	hzClient := s.GetHorizonClient()

	invokeHostFunctionOp := BuildContractCallOp(hzAcc, fnameXdr, callTxArgs, contractAddr)
	preFlightOp, minFee := PreflightHostFunctions(hzClient, &hzAcc, *invokeHostFunctionOp)

	txParams := GetBaseTransactionParamsWithFee(&hzAcc, minFee, &preFlightOp)
	txSigned, err := s.CreateSignedTxFromParams(txParams)
	if err != nil {
		return xdr.TransactionMeta{}, err
	}
	tx, err := hzClient.SubmitTransaction(txSigned)
	if err != nil {
		return xdr.TransactionMeta{}, err
	}
	// Decode transaction metadata
	txMeta, err := DecodeTxMeta(tx)
	if err != nil {
		return xdr.TransactionMeta{}, errors.New("error while decoding tx meta")
	}
	_ = txMeta.V3.SorobanMeta.ReturnValue

	return txMeta, nil
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
func DecodeTxMeta(tx horizon.Transaction) (xdr.TransactionMeta, error) {
	var transactionMeta xdr.TransactionMeta
	err := xdr.SafeUnmarshalBase64(tx.ResultMetaXdr, &transactionMeta)
	if err != nil {
		return xdr.TransactionMeta{}, err
	}

	return transactionMeta, nil
}
