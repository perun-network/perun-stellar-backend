// Copyright 2025 PolyCrypt GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"errors"
	"log"
	"math"
	"testing"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"

	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/event"
	"perun.network/perun-stellar-backend/wire/scval"
)

const (
	tokenDecimals = uint32(7)
	tokenName     = "PerunToken"
	tokenSymbol   = "PRN"
)

// TokenParams contains the parameters for the token contract.
type TokenParams struct {
	decimals uint32
	name     string
	symbol   string
}

// NewTokenParams creates a new TokenParams instance.
func NewTokenParams() *TokenParams {
	return &TokenParams{
		decimals: tokenDecimals,
		name:     tokenName,
		symbol:   tokenSymbol,
	}
}

func (t *TokenParams) GetDecimals() uint32 {
	return t.decimals
}

func (t *TokenParams) GetName() string {
	return t.name
}

func (t *TokenParams) GetSymbol() string {
	return t.symbol
}

// BuildInitTokenArgs creates the arguments for the initialize function of the token contract.
func BuildInitTokenArgs(adminAddr xdr.ScAddress, decimals uint32, tokenName string, tokenSymbol string) (xdr.ScVec, error) {
	adminScAddr, err := scval.WrapScAddress(adminAddr)
	if err != nil {
		panic(err)
	}

	decim := xdr.Uint32(decimals)
	scvaltype := xdr.ScValTypeScvU32
	decimSc, err := xdr.NewScVal(scvaltype, decim)
	if err != nil {
		panic(err)
	}

	tokenNameScString := xdr.ScString(tokenName)
	tokenNameXdr, _ := scval.MustWrapScString(tokenNameScString)

	tokenSymbolString := xdr.ScString(tokenSymbol)
	tokenSymbolXdr, _ := scval.MustWrapScString(tokenSymbolString)

	initTokenArgs := xdr.ScVec{
		adminScAddr,
		decimSc,
		tokenNameXdr,
		tokenSymbolXdr,
	}

	return initTokenArgs, nil
}

// InitTokenContract initializes the token contract.
func InitTokenContract(kp *keypair.Full, contractIDAddress xdr.ScAddress, url string) error {
	cb := NewContractBackendFromKey(kp, nil, url)

	adminScAddr, err := types.MakeAccountAddress(kp)
	if err != nil {
		panic(err)
	}

	tokenParams := NewTokenParams()
	decimals := tokenParams.GetDecimals()
	name := tokenParams.GetName()
	symbol := tokenParams.GetSymbol()

	initArgs, err := BuildInitTokenArgs(adminScAddr, decimals, name, symbol)
	if err != nil {
		panic(err)
	}

	txMeta, err := cb.InvokeSignedTx("initialize", initArgs, contractIDAddress)
	if err != nil {
		return errors.New("error while invoking and processing host function: initialize" + err.Error())
	}

	_, err = event.DecodeEventsPerun(txMeta)
	if err != nil {
		return err
	}

	return nil
}

// GetTokenName gets the name of the token.
func GetTokenName(kp *keypair.Full, contractAddress xdr.ScAddress, url string) error {
	cb := NewContractBackendFromKey(kp, nil, url)
	TokenNameArgs := xdr.ScVec{}

	_, err := cb.InvokeSignedTx("name", TokenNameArgs, contractAddress)
	if err != nil {
		panic(err)
	}

	return nil
}

// BuildGetTokenBalanceArgs creates the arguments for the getTokenBalance function of the token contract.
func BuildGetTokenBalanceArgs(balanceOf xdr.ScAddress) (xdr.ScVec, error) {
	recScAddr, err := scval.WrapScAddress(balanceOf)
	if err != nil {
		panic(err)
	}

	GetTokenBalanceArgs := xdr.ScVec{
		recScAddr,
	}

	return GetTokenBalanceArgs, nil
}

// BuildTransferTokenArgs creates the arguments for the transferToken function of the token contract.
func BuildTransferTokenArgs(from xdr.ScAddress, to xdr.ScAddress, amount xdr.Int128Parts) (xdr.ScVec, error) {
	fromScAddr, err := scval.WrapScAddress(from)
	if err != nil {
		panic(err)
	}

	toScAddr, err := scval.WrapScAddress(to)
	if err != nil {
		panic(err)
	}

	amountSc, err := scval.WrapInt128Parts(amount)
	if err != nil {
		panic(err)
	}

	GetTokenBalanceArgs := xdr.ScVec{
		fromScAddr,
		toScAddr,
		amountSc,
	}

	return GetTokenBalanceArgs, nil
}

// Deploy deploys the token contract.
func Deploy(t *testing.T, kp *keypair.Full, contractPath string, url string) (xdr.ScAddress, xdr.Hash) {
	deployerCB := NewContractBackendFromKey(kp, nil, url)
	tr := deployerCB.GetTransactor()
	hzClient := tr.GetHorizonClient()
	deployerAccReq := horizonclient.AccountRequest{AccountID: kp.Address()}
	deployerAcc, err := hzClient.AccountDetail(deployerAccReq)

	require.NoError(t, err)

	installContractOpInstall := channel.AssembleInstallContractCodeOp(kp.Address(), contractPath)
	preFlightOp, minFeeInstall, err := client.PreflightHostFunctions(hzClient, &deployerAcc, *installContractOpInstall)

	require.NoError(t, err)

	txParamsInstall := client.GetBaseTransactionParamsWithFee(&deployerAcc, int64(100)+minFeeInstall, &preFlightOp) //nolint:gomnd
	txSignedInstall, err := client.CreateSignedTransactionWithParams([]*keypair.Full{kp}, txParamsInstall, client.NETWORK_PASSPHRASE)
	require.NoError(t, err)

	_, err = hzClient.SubmitTransaction(txSignedInstall)
	var hErr *horizonclient.Error
	if errors.As(err, &hErr) {
		log.Println(hErr.Problem, "fee: ", minFeeInstall)
	}

	require.NoError(t, err)

	createContractOp := channel.AssembleCreateContractOp(kp.Address(), contractPath, "a1", client.NETWORK_PASSPHRASE)
	preFlightOpCreate, minFeeDeploy, err := client.PreflightHostFunctions(hzClient, &deployerAcc, *createContractOp)
	require.NoError(t, err)
	txParamsCreate := client.GetBaseTransactionParamsWithFee(&deployerAcc, int64(100)+minFeeDeploy, &preFlightOpCreate) //nolint:gomnd
	txSignedCreate, err := client.CreateSignedTransactionWithParams([]*keypair.Full{kp}, txParamsCreate, client.NETWORK_PASSPHRASE)

	require.NoError(t, err)

	_, err = hzClient.SubmitTransaction(txSignedCreate)
	require.NoError(t, err)

	contractID := preFlightOpCreate.Ext.SorobanData.Resources.Footprint.ReadWrite[0].MustContractData().Contract.ContractId
	contractHash := preFlightOpCreate.Ext.SorobanData.Resources.Footprint.ReadOnly[0].MustContractCode().Hash
	contractIDAddress := xdr.ScAddress{
		Type:       xdr.ScAddressTypeScAddressTypeContract,
		ContractId: contractID,
	}

	return contractIDAddress, contractHash
}

// MintToken mints a token.
func MintToken(kp *keypair.Full, contractAddr xdr.ScAddress, amount uint64, recipientAddr xdr.ScAddress, url string) error {
	cb := NewContractBackendFromKey(kp, nil, url)

	if amount > math.MaxInt64 {
		return errors.New("amount represents negative number")
	}

	amountTo128Xdr := xdr.Int128Parts{Hi: 0, Lo: xdr.Uint64(amount)}

	amountSc, err := xdr.NewScVal(xdr.ScValTypeScvI128, amountTo128Xdr)
	if err != nil {
		panic(err)
	}
	mintTokenArgs, err := client.BuildMintTokenArgs(recipientAddr, amountSc)
	if err != nil {
		panic(err)
	}
	_, err = cb.InvokeSignedTx("mint", mintTokenArgs, contractAddr)
	if err != nil {
		panic(err)
	}
	return nil
}
