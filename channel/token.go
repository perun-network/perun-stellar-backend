// Copyright 2023 PolyCrypt GmbH
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

package channel

import (
	"github.com/stellar/go/xdr"
	"perun.network/perun-stellar-backend/wire/scval"
)

const tokenDecimals = uint32(7)
const tokenName = "PerunToken"
const tokenSymbol = "PRN"

type TokenParams struct {
	decimals uint32
	name     string
	symbol   string
}

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
	tokenNameXdr := scval.MustWrapScString(tokenNameScString)

	tokenSymbolString := xdr.ScString(tokenSymbol)
	tokenSymbolXdr := scval.MustWrapScString(tokenSymbolString)

	initTokenArgs := xdr.ScVec{
		adminScAddr,
		decimSc,
		tokenNameXdr,
		tokenSymbolXdr,
	}

	return initTokenArgs, nil
}

// func InitTokenContract(hzClient *env.StellarClient, hzAccount horizon.Account, kpAdmin *keypair.Full, contractAddress xdr.ScAddress) error {

// 	tokenParams := NewTokenParams()
// 	adminScAddr, err := types.MakeAccountAddress(kpAdmin)

// 	if err != nil {
// 		panic(err)
// 	}
// 	initTokenArgs, err := wire.BuildInitTokenArgs(adminScAddr, tokenParams.decimals, tokenParams.name, tokenParams.symbol)

// 	if err != nil {
// 		return errors.New("error while building fund tx")
// 	}
// 	auth := []xdr.SorobanAuthorizationEntry{}

// 	_, err = hzClient.GetHorizonClient().InvokeAndProcessHostFunction(hzAccount, "initialize", initTokenArgs, contractAddress, kpAdmin, auth)
// 	if err != nil {
// 		return errors.New("error while invoking and processing host function for InitTokenContract")
// 	}

// 	return nil
// }

func BuildTokenNameArgs() (xdr.ScVec, error) {

	return xdr.ScVec{}, nil
}

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

// func GetTokenName(ctx context.Context, stellarClient *env.StellarClient, contractAddress xdr.ScAddress) error {

// 	hzAcc := stellarClient.GetHorizonAcc()
// 	kp := stellarClient.GetAccount()
// 	// generate tx to open the channel
// 	TokenNameArgs, err := BuildTokenNameArgs()
// 	if err != nil {
// 		return errors.New("error while building fund tx")
// 	}
// 	auth := []xdr.SorobanAuthorizationEntry{}

// 	_, err = stellarClient.InvokeAndProcessHostFunction(hzAcc, "name", TokenNameArgs, contractAddress, kp, auth)
// 	if err != nil {
// 		return errors.New("error while invoking and processing host function for GetTokenName")
// 	}

// 	return nil
// }

// func MintToken(ctx context.Context, stellarClient *env.StellarClient, mintTo xdr.ScAddress, mintAmount int64, contractAddress xdr.ScAddress) error {

// 	hzAcc := stellarClient.GetHorizonAcc()
// 	kp := stellarClient.GetAccount()
// 	TokenNameArgs, err := BuildMintTokenArgs(mintTo, mintAmount)
// 	if err != nil {
// 		return errors.New("error while building fund tx")
// 	}
// 	auth := []xdr.SorobanAuthorizationEntry{}
// 	_, err = stellarClient.InvokeAndProcessHostFunction(hzAcc, "mint", TokenNameArgs, contractAddress, kp, auth)
// 	if err != nil {
// 		return errors.New("error while invoking and processing host function for GetTokenName")
// 	}

// 	return nil
// }

// func GetTokenBalance(ctx context.Context, stellarClient *env.StellarClient, balanceOf xdr.ScAddress, contractAddress xdr.ScAddress) error {

// 	hzAcc := stellarClient.GetHorizonAcc()
// 	kp := stellarClient.GetAccount()
// 	// generate tx to open the channel
// 	getBalanceArgs, err := BuildGetTokenBalanceArgs(balanceOf)
// 	if err != nil {
// 		return errors.New("error while building fund tx")
// 	}
// 	auth := []xdr.SorobanAuthorizationEntry{}
// 	txMeta, err := stellarClient.InvokeAndProcessHostFunction(hzAcc, "balance", getBalanceArgs, contractAddress, kp, auth)
// 	if err != nil {
// 		return errors.New("error while invoking and processing host function for GetTokenName")
// 	}

// 	_ = txMeta.V3.SorobanMeta.ReturnValue

// 	// rVal := reVal.MustI128()

// 	return nil
// }

// func TransferToken(ctx context.Context, stellarClient *env.StellarClient, from xdr.ScAddress, to xdr.ScAddress, amount128 xdr.Int128Parts, contractAddress xdr.ScAddress) error {

// 	hzAcc := stellarClient.GetHorizonAcc()
// 	kp := stellarClient.GetAccount()
// 	transferArgs, err := BuildTransferTokenArgs(from, to, amount128)
// 	if err != nil {
// 		return errors.New("error while building fund tx")
// 	}
// 	auth := []xdr.SorobanAuthorizationEntry{}
// 	txMeta, err := stellarClient.InvokeAndProcessHostFunction(hzAcc, "transfer", transferArgs, contractAddress, kp, auth)
// 	if err != nil {
// 		return errors.New("error while invoking and processing host function for GetTokenName")
// 	}

// 	_ = txMeta.V3.SorobanMeta.ReturnValue

// 	// rVal := reVal.MustI128()

// 	return nil
// }
