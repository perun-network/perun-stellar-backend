package env

import (
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/wire"
	"perun.network/perun-stellar-backend/wire/scval"
)

func BuildContractCallOp(caller horizon.Account, fName xdr.ScSymbol, callArgs xdr.ScVec, contractIdAddress xdr.ScAddress, auth []xdr.SorobanAuthorizationEntry) *txnbuild.InvokeHostFunction {

	return &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.InvokeContractArgs{
				ContractAddress: contractIdAddress,
				FunctionName:    fName,
				Args:            callArgs,
			},
		},
		Auth:          auth,
		SourceAccount: caller.AccountID,
	}
}

func BuildOpenTxArgs(params *pchannel.Params, state *pchannel.State) xdr.ScVec {
	paramsStellar, err := wire.MakeParams(*params)
	if err != nil {
		panic(err)
	}
	stateStellar, err := wire.MakeState(*state)
	if err != nil {
		panic(err)
	}
	paramsXdr, err := paramsStellar.ToScVal()
	if err != nil {
		panic(err)
	}
	stateXdr, err := stateStellar.ToScVal()
	if err != nil {
		panic(err)
	}
	openArgs := xdr.ScVec{
		paramsXdr,
		stateXdr,
	}
	return openArgs
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
		panic(err)
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

func DecodeTxMeta(tx horizon.Transaction) (xdr.TransactionMeta, error) {
	var transactionMeta xdr.TransactionMeta
	err := xdr.SafeUnmarshalBase64(tx.ResultMetaXdr, &transactionMeta)
	if err != nil {
		return xdr.TransactionMeta{}, err
	}

	return transactionMeta, nil
}

func stellarAssetContractID(stellarEnv *IntegrationTestEnv, asset xdr.Asset) xdr.Hash {
	contractID, err := asset.ContractID(stellarEnv.GetPassPhrase())
	if err != nil {
		panic(err)
	}
	return contractID
}

func i128Param(hi int64, lo uint64) xdr.ScVal {
	i128 := &xdr.Int128Parts{
		Hi: xdr.Int64(hi),
		Lo: xdr.Uint64(lo),
	}
	return xdr.ScVal{
		Type: xdr.ScValTypeScvI128,
		I128: i128,
	}
}

func AccountAddressParam(accountID string) xdr.ScVal {
	address := xdr.ScAddress{
		Type:      xdr.ScAddressTypeScAddressTypeAccount,
		AccountId: xdr.MustAddressPtr(accountID),
	}
	return xdr.ScVal{
		Type:    xdr.ScValTypeScvAddress,
		Address: &address,
	}
}
