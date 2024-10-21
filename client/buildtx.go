package client

import (
	"github.com/stellar/go/xdr"
	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/wire"
	"perun.network/perun-stellar-backend/wire/scval"
)

func buildOpenTxArgs(params pchannel.Params, state pchannel.State) (xdr.ScVec, error) {
	paramsStellar, err := wire.MakeParams(params)
	if err != nil {
		return xdr.ScVec{}, err
	}
	stateStellar, err := wire.MakeState(state)
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

func buildChanIdTxArgs(chanID pchannel.ID) (xdr.ScVec, error) {

	channelID, err := scval.WrapScBytes(chanID[:])
	if err != nil {
		return xdr.ScVec{}, err
	}

	getChannelArgs := xdr.ScVec{
		channelID,
	}
	return getChannelArgs, nil
}

func buildChanIdxTxArgs(chanID pchannel.ID, withdrawerIdx bool) (xdr.ScVec, error) {

	withdrawerXdrIdx, err := scval.MustWrapBool(withdrawerIdx)
	if err != nil {
		return xdr.ScVec{}, err
	}

	channelIDXdr, err := scval.WrapScBytes(chanID[:])
	if err != nil {
		return xdr.ScVec{}, err
	}

	withdrawArgs := xdr.ScVec{
		channelIDXdr,
		withdrawerXdrIdx,
	}
	return withdrawArgs, nil
}

func buildSignedStateTxArgs(state pchannel.State, sigs []pwallet.Sig) (xdr.ScVec, error) {

	wireState, err := wire.MakeState(state)
	if err != nil {
		return xdr.ScVec{}, err
	}

	sigAXdr, err := scval.WrapScBytes(sigs[0])
	if err != nil {
		return xdr.ScVec{}, err
	}
	sigBXdr, err := scval.WrapScBytes(sigs[1])
	if err != nil {
		return xdr.ScVec{}, err
	}
	xdrState, err := wireState.ToScVal()
	if err != nil {
		return xdr.ScVec{}, err
	}

	signedStateArgs := xdr.ScVec{
		xdrState,
		sigAXdr,
		sigBXdr,
	}
	return signedStateArgs, nil
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
