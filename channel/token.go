package channel

import (
	"context"
	"errors"
	"github.com/stellar/go/xdr"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/wire/scval"
)

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

func BuildMintTokenArgs(mintTo xdr.ScAddress, amount int64) (xdr.ScVec, error) {

	recScAddr, err := scval.WrapScAddress(mintTo)
	if err != nil {
		panic(err)
	}

	amounti64 := xdr.Int64(amount)
	scvaltype := xdr.ScValTypeScvI64

	amountSc, err := xdr.NewScVal(scvaltype, amounti64)
	if err != nil {
		panic(err)
	}

	MintTokenArgs := xdr.ScVec{
		recScAddr,
		amountSc,
	}

	return MintTokenArgs, nil
}

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

func InitTokenContract(ctx context.Context, stellarClient *env.StellarClient, adminAddr xdr.ScAddress, decimals uint32, tokenName, tokenSymbol string, contractAddress xdr.ScAddress) error {

	hzAcc := stellarClient.GetHorizonAcc()
	kp := stellarClient.GetAccount()

	initTokenArgs, err := BuildInitTokenArgs(adminAddr, decimals, tokenName, tokenSymbol)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	auth := []xdr.SorobanAuthorizationEntry{}
	_, err = stellarClient.InvokeAndProcessHostFunction(hzAcc, "initialize", initTokenArgs, contractAddress, kp, auth)
	if err != nil {
		return errors.New("error while invoking and processing host function for InitTokenContract")
	}

	return nil
}

func GetTokenName(ctx context.Context, stellarClient *env.StellarClient, contractAddress xdr.ScAddress) error {

	hzAcc := stellarClient.GetHorizonAcc()
	kp := stellarClient.GetAccount()
	// generate tx to open the channel
	TokenNameArgs, err := BuildTokenNameArgs()
	if err != nil {
		return errors.New("error while building fund tx")
	}
	auth := []xdr.SorobanAuthorizationEntry{}

	_, err = stellarClient.InvokeAndProcessHostFunction(hzAcc, "name", TokenNameArgs, contractAddress, kp, auth)
	if err != nil {
		return errors.New("error while invoking and processing host function for GetTokenName")
	}

	return nil
}

func MintToken(ctx context.Context, stellarClient *env.StellarClient, mintTo xdr.ScAddress, mintAmount int64, contractAddress xdr.ScAddress) error {

	hzAcc := stellarClient.GetHorizonAcc()
	kp := stellarClient.GetAccount()
	TokenNameArgs, err := BuildMintTokenArgs(mintTo, mintAmount)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	auth := []xdr.SorobanAuthorizationEntry{}
	_, err = stellarClient.InvokeAndProcessHostFunction(hzAcc, "mint", TokenNameArgs, contractAddress, kp, auth)
	if err != nil {
		return errors.New("error while invoking and processing host function for GetTokenName")
	}

	return nil
}

func GetTokenBalance(ctx context.Context, stellarClient *env.StellarClient, balanceOf xdr.ScAddress, contractAddress xdr.ScAddress) error {

	hzAcc := stellarClient.GetHorizonAcc()
	kp := stellarClient.GetAccount()
	// generate tx to open the channel
	getBalanceArgs, err := BuildGetTokenBalanceArgs(balanceOf)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	auth := []xdr.SorobanAuthorizationEntry{}
	txMeta, err := stellarClient.InvokeAndProcessHostFunction(hzAcc, "balance", getBalanceArgs, contractAddress, kp, auth)
	if err != nil {
		return errors.New("error while invoking and processing host function for GetTokenName")
	}

	_ = txMeta.V3.SorobanMeta.ReturnValue

	// rVal := reVal.MustI128()

	return nil
}

func TransferToken(ctx context.Context, stellarClient *env.StellarClient, from xdr.ScAddress, to xdr.ScAddress, amount128 xdr.Int128Parts, contractAddress xdr.ScAddress) error {

	hzAcc := stellarClient.GetHorizonAcc()
	kp := stellarClient.GetAccount()
	transferArgs, err := BuildTransferTokenArgs(from, to, amount128)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	auth := []xdr.SorobanAuthorizationEntry{}
	txMeta, err := stellarClient.InvokeAndProcessHostFunction(hzAcc, "transfer", transferArgs, contractAddress, kp, auth)
	if err != nil {
		return errors.New("error while invoking and processing host function for GetTokenName")
	}

	_ = txMeta.V3.SorobanMeta.ReturnValue

	// rVal := reVal.MustI128()

	return nil
}
