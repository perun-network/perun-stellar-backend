package wire

import (
	"github.com/stellar/go/xdr"
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
