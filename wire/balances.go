// Copyright 2024 PolyCrypt GmbH
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
	"bytes"
	"encoding/binary"
	"errors"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	"math/big"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/multi"
	"perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/channel/types"
	wtypes "perun.network/perun-stellar-backend/wallet/types"
	"perun.network/perun-stellar-backend/wire/scval"
	"strconv"
)

var MaxBalance = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1))

// Tokens field:
// multiasset case xdr.ScAddress -> xdr.ScVec{xdr.ScAddress1, xdr.ScAddress2} (Stellar MultiAsset)
// crossasset case xdr.ScAddress -> xdR.ScVec{xdr.ScMap{xdr.ScAddress, Int}, xdr.ScMap{xdr.ScBytes, Int}} for Stellar (xdr.ScAddress), Ethereum (xdr.ScBytes) respectively
type Balances struct {
	BalA   xdr.ScVec //{xdr.Int128Parts, xdr.Int128Parts}
	BalB   xdr.ScVec //{xdr.Int128Parts, xdr.Int128Parts}
	Tokens xdr.ScVec
}

const (
	SymbolBalancesBalA   xdr.ScSymbol = "bal_a"
	SymbolBalancesBalB   xdr.ScSymbol = "bal_b"
	SymbolBalancesTokens xdr.ScSymbol = "tokens"

	SymbolTokensAddress xdr.ScSymbol = "address"
	SymbolTokensChain   xdr.ScSymbol = "chain"
)

func (b Balances) ToScVal() (xdr.ScVal, error) {
	var err error
	balA, err := scval.WrapVec(b.BalA)
	if err != nil {
		return xdr.ScVal{}, err
	}
	balB, err := scval.WrapVec(b.BalB)
	if err != nil {
		return xdr.ScVal{}, err
	}
	tokens, err := scval.WrapVec(b.Tokens)
	if err != nil {
		return xdr.ScVal{}, err
	}
	m, err := MakeSymbolScMap(
		[]xdr.ScSymbol{
			SymbolBalancesBalA,
			SymbolBalancesBalB,
			SymbolBalancesTokens,
		},
		[]xdr.ScVal{balA, balB, tokens},
	)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return scval.WrapScMap(m)
}

func (b *Balances) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map")
	}
	if len(*m) != 3 {
		return errors.New("expected map of length 3")
	}
	balAVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolBalancesBalA), *m)
	if err != nil {
		return err
	}

	balA, ok := balAVal.GetVec()
	if !ok {
		return errors.New("expected vec of i128")
	}

	balBVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolBalancesBalB), *m)
	if err != nil {
		return err
	}

	balB, ok := balBVal.GetVec()
	if !ok {
		return errors.New("expected vec of i128")
	}

	tokenVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolBalancesTokens), *m)
	if err != nil {
		return err
	}
	tokens, ok := tokenVal.GetVec()
	if !ok {
		return errors.New("expected vec of addresses")
	}

	b.BalA = *balA
	b.BalB = *balB
	b.Tokens = *tokens
	return nil
}

func BalancesFromScVal(v xdr.ScVal) (Balances, error) {
	var b Balances
	err := (&b).FromScVal(v)
	return b, err
}

func (b Balances) EncodeTo(e *xdr3.Encoder) error {
	v, err := b.ToScVal()
	if err != nil {
		return err
	}
	_, err = e.Encode(v)
	return err
}

func (b *Balances) DecodeFrom(d *xdr3.Decoder) (int, error) {
	var v xdr.ScVal
	n, err := d.Decode(&v)
	if err != nil {
		return n, err
	}
	return n, b.FromScVal(v)
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (b Balances) MarshalBinary() ([]byte, error) {
	buf := bytes.Buffer{}
	e := xdr3.NewEncoder(&buf)
	err := b.EncodeTo(e)
	return buf.Bytes(), err
}

func (b *Balances) UnmarshalBinary(data []byte) error {
	d := xdr3.NewDecoder(bytes.NewReader(data))
	_, err := b.DecodeFrom(d)
	return err
}

func extractAndConvertLedgerID(asset interface{}) (uint64, error) {
	var lidMapKey multi.LedgerIDMapKey

	switch asset := asset.(type) {
	case *types.StellarAsset:
		lidMapKey = asset.LedgerID().MapKey()
	case *types.EthAsset:
		lidMapKey = asset.LedgerID().MapKey()
	default:
		return 0, errors.New("unknown asset type")
	}

	lidval, err := strconv.ParseUint(string(lidMapKey), 10, 64)
	if err != nil {
		return 0, errors.New("invalid Ledger ID format")
	}

	return lidval, nil
}

func MakeTokens(assets []channel.Asset) (xdr.ScVec, error) {
	var tokens xdr.ScVec

	for _, ast := range assets {
		lidvalXdr, err := extractAndConvertLedgerID(ast)
		if err != nil {
			return xdr.ScVec{}, err
		}
		lidvalXdrValue := xdr.Uint64(lidvalXdr)

		var tokenAddrVal xdr.ScVal
		var tokenAddrSymbol xdr.ScSymbol

		switch asset := ast.(type) {
		case *types.StellarAsset:
			tokenAddrSymbol = "Stellar"

			sa, err := types.ToStellarAsset(asset)
			if err != nil {
				return xdr.ScVec{}, err
			}
			token, err := sa.MakeScAddress()
			if err != nil {
				return xdr.ScVec{}, err
			}

			tokenAddrVal, err = scval.MustWrapScAddress(token)
			if err != nil {
				return xdr.ScVec{}, err
			}

		case *types.EthAsset:
			tokenAddrSymbol = "Eth"

			tokenAddrVal, err = scval.MustWrapScBytes(asset.EthAddress().Bytes())
			if err != nil {
				return xdr.ScVec{}, err
			}

		default:
			return xdr.ScVec{}, errors.New("unexpected asset type")
		}

		tokenAddrSymVal := scval.MustWrapScSymbol(tokenAddrSymbol)

		tokenAddrVecVal, err := scval.WrapVec(xdr.ScVec{tokenAddrSymVal, tokenAddrVal})
		if err != nil {
			return xdr.ScVec{}, err
		}

		tokenChainVal, err := scval.MustWrapScUint64(lidvalXdrValue)
		if err != nil {
			return xdr.ScVec{}, err
		}

		tokenMap, err := MakeSymbolScMap(
			[]xdr.ScSymbol{SymbolTokensAddress, SymbolTokensChain},
			[]xdr.ScVal{tokenAddrVecVal, tokenChainVal},
		)
		if err != nil {
			return xdr.ScVec{}, err
		}

		tokenMapVal, err := scval.WrapScMap(tokenMap)
		if err != nil {
			return xdr.ScVec{}, err
		}

		tokens = append(tokens, tokenMapVal)
	}

	tokenCrossSym := xdr.ScSymbol("Cross")
	tokenCrossSymVal := scval.MustWrapScSymbol(tokenCrossSym)

	tokensVecVal, err := scval.WrapVec(tokens)
	if err != nil {
		return xdr.ScVec{}, err
	}

	return xdr.ScVec{tokenCrossSymVal, tokensVecVal}, nil
}

func MakeBalances(alloc channel.Allocation) (Balances, error) {
	if err := alloc.Valid(); err != nil {
		return Balances{}, err
	}
	if len(alloc.Locked) != 0 {
		return Balances{}, errors.New("expected no locked funds")
	}
	assets := alloc.Assets
	tokens, err := MakeTokens(assets)
	if err != nil {
		return Balances{}, err
	}

	numParts := alloc.NumParts()
	if numParts < 2 {
		return Balances{}, errors.New("expected at least two parts")
	}

	bals := alloc.Balances

	balPartVecs := make([]xdr.ScVec, numParts)

	var balScVal xdr.ScVal

	for _, balsAsset := range bals {
		for j, val := range balsAsset {
			xdrBalPart, err := MakeInt128Parts(val)
			if err != nil {
				return Balances{}, err
			}

			balScVal, err = scval.WrapInt128Parts(xdrBalPart)
			if err != nil {
				return Balances{}, err
			}

			if j < numParts {
				balPartVecs[j] = append(balPartVecs[j], balScVal)
			} else {
				return Balances{}, errors.New("unexpected number of parts in balance asset")
			}
		}
	}

	// Assign the first two parts to BalA and BalB for backward compatibility
	var balAPartVec, balBPartVec xdr.ScVec
	if numParts > 0 {
		balAPartVec = balPartVecs[0]
	}
	if numParts > 1 {
		balBPartVec = balPartVecs[1]
	}

	return Balances{
		BalA:   balAPartVec,
		BalB:   balBPartVec,
		Tokens: tokens,
	}, nil
}

// MakeInt128Parts converts a big.Int to xdr.Int128Parts.
// It returns an error if the big.Int is negative or too large.
func MakeInt128Parts(i *big.Int) (xdr.Int128Parts, error) {
	if i.Sign() < 0 {
		return xdr.Int128Parts{}, errors.New("expected non-negative balance")
	}
	if i.Cmp(MaxBalance) > 0 {
		return xdr.Int128Parts{}, errors.New("balance too large")
	}
	b := make([]byte, 16)
	b = i.FillBytes(b)
	hi := binary.BigEndian.Uint64(b[:8])
	lo := binary.BigEndian.Uint64(b[8:])
	return xdr.Int128Parts{
		Hi: xdr.Int64(hi),
		Lo: xdr.Uint64(lo),
	}, nil
}

func ToBigInt(i xdr.Int128Parts) (*big.Int, error) {
	b := make([]byte, 16)
	binary.BigEndian.PutUint64(b[:8], uint64(i.Hi))
	binary.BigEndian.PutUint64(b[8:], uint64(i.Lo))
	return new(big.Int).SetBytes(b), nil
}

func makeAllocationMulti(assets []channel.Asset, balsA, balsB []*big.Int) (*channel.Allocation, error) {

	if len(balsA) != len(balsB) {
		return nil, errors.New("expected equal number of balances")
	}

	if len(assets) != len(balsA) {
		return nil, errors.New("expected equal number of assets and balances")
	}

	numParts := 2

	// TODO might be mixed backends
	backendIDs := make([]wallet.BackendID, wtypes.StellarBackendID)

	alloc := channel.NewAllocation(numParts, backendIDs, assets...)

	for i, _ := range assets {
		alloc.Balances[i] = []*big.Int{balsA[i], balsB[i]}
	}

	alloc.Locked = make([]channel.SubAlloc, 0)

	if err := alloc.Valid(); err != nil {
		return nil, err
	}

	return alloc, nil
}
