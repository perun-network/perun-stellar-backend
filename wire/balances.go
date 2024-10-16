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
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wire/scval"
)

var MaxBalance = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1))

type Balances struct {
	BalA   xdr.ScVec //{xdr.Int128Parts, xdr.Int128Parts}
	BalB   xdr.ScVec //{xdr.Int128Parts, xdr.Int128Parts}
	Tokens xdr.ScVec // multiasset xdr.ScAddress -> xdr.ScVec{xdr.ScAddress1, xdr.ScAddress2}
}

const (
	SymbolBalancesBalA   xdr.ScSymbol = "bal_a"
	SymbolBalancesBalB   xdr.ScSymbol = "bal_b"
	SymbolBalancesTokens xdr.ScSymbol = "tokens"
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

func MakeBalances(alloc channel.Allocation) (Balances, error) {
	if err := alloc.Valid(); err != nil {
		return Balances{}, err
	}
	if len(alloc.Locked) != 0 {
		return Balances{}, errors.New("expected no locked funds")
	}
	assets := alloc.Assets
	var tokens xdr.ScVec
	for i, ast := range assets {
		_, ok := ast.(*types.StellarAsset)
		if !ok {
			return Balances{}, errors.New("expected stellar asset")
		}
		sa, err := types.ToStellarAsset(assets[i])
		if err != nil {
			return Balances{}, err
		}
		token, err := sa.MakeScAddress()
		if err != nil {
			return Balances{}, err
		}

		tokenVal := scval.MustWrapScAddress(token)

		tokens = append(tokens, tokenVal)
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

func ToAllocation(b Balances) (*channel.Allocation, error) {

	var balsPart xdr.ScVal

	var stAssets []channel.Asset

	var alloc *channel.Allocation

	// iterate for asset addresses inside allocation

	for i, val := range b.Tokens {

		balsPart = b.BalA[i]
		token, ok := val.GetAddress()
		if !ok {
			return nil, errors.New("expected address")
		}

		stAsset, err := types.NewStellarAssetFromScAddress(token)
		if err != nil {
			return nil, err
		}

		stAssets = append(stAssets, stAsset)

	}

	alloc = channel.NewAllocation(2, stAssets...)

	for i, _ := range b.Tokens {
		balsPartVec := *balsPart.MustVec()

		for _, val := range balsPartVec {
			bal, ok := val.GetI128()
			if !ok {
				return nil, errors.New("expected i128")
			}

			balInt, err := ToBigInt(bal)
			if err != nil {
				return nil, err
			}

			alloc.SetBalance(channel.Index(i), stAssets[i], balInt)

		}

	}

	if err := alloc.Valid(); err != nil {
		return nil, err
	}
	return alloc, nil
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

func makeAllocation(asset channel.Asset, balA, balB *big.Int) (*channel.Allocation, error) {
	alloc := channel.NewAllocation(2, asset)
	alloc.SetBalance(0, asset, balA)
	alloc.SetBalance(1, asset, balB)
	alloc.Locked = make([]channel.SubAlloc, 0)

	if err := alloc.Valid(); err != nil {
		return nil, err
	}

	return alloc, nil
}

func makeAllocationMulti(assets []channel.Asset, balsA, balsB []*big.Int) (*channel.Allocation, error) {

	if len(balsA) != len(balsB) {
		return nil, errors.New("expected equal number of balances")
	}

	if len(assets) != len(balsA) {
		return nil, errors.New("expected equal number of assets and balances")
	}

	numParts := 2

	alloc := channel.NewAllocation(numParts, assets...)

	for i, _ := range assets {
		alloc.Balances[i] = []*big.Int{balsA[i], balsB[i]}
	}

	alloc.Locked = make([]channel.SubAlloc, 0)

	if err := alloc.Valid(); err != nil {
		return nil, err
	}

	return alloc, nil
}
