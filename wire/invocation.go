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
