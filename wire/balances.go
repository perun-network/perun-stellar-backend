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

package wire

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"

	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/multi"
	"perun.network/go-perun/wallet"

	"perun.network/perun-stellar-backend/channel/types"
	wtypes "perun.network/perun-stellar-backend/wallet/types"
	"perun.network/perun-stellar-backend/wire/scval"
)

// MaxBalance is the maximum balance that can be represented in the wire format.
var MaxBalance = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1)) //nolint:gomnd

// Balances represents the balances of a channel.
type Balances struct {
	BalA   xdr.ScVec // {xdr.Int128Parts, xdr.Int128Parts}
	BalB   xdr.ScVec // {xdr.Int128Parts, xdr.Int128Parts}
	Tokens []Asset
}

// Asset represents an Asset in the soroban-contract.
type Asset struct {
	Chain          xdr.ScVec     // {xdr.Int128Parts, xdr.Int128Parts}
	StellarAddress xdr.ScAddress // {xdr.Int128Parts, xdr.Int128Parts}
	EthAddress     xdr.ScBytes
}

const (
	SymbolBalancesBalA   xdr.ScSymbol = "bal_a"
	SymbolBalancesBalB   xdr.ScSymbol = "bal_b"
	SymbolBalancesTokens xdr.ScSymbol = "tokens"

	SymbolTokensStellarAddress xdr.ScSymbol = "stellar_address"
	SymbolTokensEthAddress     xdr.ScSymbol = "eth_address"
	SymbolTokensChain          xdr.ScSymbol = "chain"
)

// ToScVal encodes a Asset struct to a xdr.ScVal.
func (a Asset) ToScVal() (xdr.ScVal, error) {
	var err error
	chain, err := scval.WrapVec(a.Chain)
	if err != nil {
		return xdr.ScVal{}, err
	}
	stellarAddr, err := scval.WrapScAddress(a.StellarAddress)
	if err != nil {
		return xdr.ScVal{}, err
	}
	ethAddr, err := scval.WrapScBytes(a.EthAddress)
	if err != nil {
		return xdr.ScVal{}, err
	}
	m, err := MakeSymbolScMap(
		[]xdr.ScSymbol{
			SymbolTokensChain,
			SymbolTokensStellarAddress,
			SymbolTokensEthAddress,
		},
		[]xdr.ScVal{chain, stellarAddr, ethAddr},
	)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return scval.WrapScMap(m)
}

// FromScVal decodes a Asset struct from a xdr.ScVal.
func (a *Asset) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map")
	}
	if len(*m) != 3 { //nolint:gomnd
		return errors.New("expected map of length 3")
	}
	chainVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolTokensChain), *m)
	if err != nil {
		return err
	}
	chain, ok := chainVal.GetVec()
	if !ok {
		return errors.New("expected uint64 for chain")
	}

	stellarAddrVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolTokensStellarAddress), *m)
	if err != nil {
		return err
	}
	stellarAddr, ok := stellarAddrVal.GetAddress()
	if !ok {
		return errors.New("expected address")
	}

	ethAddrVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolTokensEthAddress), *m)
	if err != nil {
		return err
	}
	ethAddr, ok := ethAddrVal.GetBytes()
	if !ok {
		return errors.New("expected vec of bytes")
	}
	a.Chain = *chain
	a.StellarAddress = stellarAddr
	a.EthAddress = ethAddr
	return nil
}

// ToScVal encodes a Balances struct to a xdr.ScVal.
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
	var tokensVec xdr.ScVec
	for _, token := range b.Tokens {
		t, err := token.ToScVal()
		if err != nil {
			return xdr.ScVal{}, err
		}
		tokensVec = append(tokensVec, t)
	}
	tokens, err := scval.WrapVec(tokensVec)
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

// FromScVal decodes a Balances struct from a xdr.ScVal.
func (b *Balances) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map")
	}
	if len(*m) != 3 { //nolint:gomnd
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
	tokensVec, ok := tokenVal.GetVec()
	if !ok {
		return errors.New("expected vec of addresses")
	}
	var tokens []Asset
	for _, tokenVal := range *tokensVec {
		var token Asset
		err := token.FromScVal(tokenVal)
		if err != nil {
			return err
		}
		tokens = append(tokens, token)
	}

	b.BalA = *balA
	b.BalB = *balB
	b.Tokens = tokens
	return nil
}

// BalancesFromScVal decodes a Balances struct from a xdr.ScVal.
func BalancesFromScVal(v xdr.ScVal) (Balances, error) {
	var b Balances
	err := (&b).FromScVal(v)
	return b, err
}

// EncodeTo encodes the Balances struct to a xdr.Encoder.
func (b Balances) EncodeTo(e *xdr3.Encoder) error {
	v, err := b.ToScVal()
	if err != nil {
		return err
	}
	_, err = e.Encode(v)
	return err
}

// DecodeFrom decodes the Balances struct from a xdr.Decoder.
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

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (b *Balances) UnmarshalBinary(data []byte) error {
	d := xdr3.NewDecoder(bytes.NewReader(data))
	_, err := b.DecodeFrom(d)
	return err
}

func extractAndConvertLedgerID(asset channel.Asset) (uint64, error) {
	multiAsset, ok := asset.(multi.Asset)
	if !ok {
		return 0, errors.New("invalid Asset format")
	}
	id := multiAsset.LedgerBackendID().LedgerID()
	if id == nil {
		return 0, errors.New("invalid LedgerID")
	}
	lidMapKey := id.MapKey()
	lidval, err := strconv.ParseUint(string(lidMapKey), 10, 64)
	if err != nil {
		log.Println("Could not parse ledgerID")
		return 0, errors.New("invalid Ledger ID format")
	}

	return lidval, nil
}

// MakeTokens converts a list of channel.Assets to a list of wire.Assets.
//
//nolint:funlen
func MakeTokens(assets []channel.Asset) ([]Asset, error) {
	var tokens []Asset

	for _, ast := range assets {
		lidvalXdr, err := extractAndConvertLedgerID(ast)
		if err != nil {
			return nil, err
		}
		lidvalXdrValue := xdr.Uint64(lidvalXdr)
		lidval, err := scval.WrapUint64(lidvalXdrValue)
		if err != nil {
			return nil, err
		}
		lidvec := xdr.ScVec{lidval}

		var tokenStellarAddrVal xdr.ScAddress
		var tokenEthAddrVal xdr.ScBytes

		switch asset := ast.(type) {
		case *types.StellarAsset:
			sa, err := types.ToStellarAsset(asset)
			if err != nil {
				return nil, err
			}
			tokenStellarAddrVal, err = sa.MakeScAddress()
			if err != nil {
				return nil, err
			}
			defAddr := make([]byte, 20) //nolint:gomnd
			tokenEthAddrVal = defAddr
			if err != nil {
				return nil, err
			}

		case *types.EthAsset:
			tokenEthAddrVal, err = asset.AssetHolder.MarshalBinary()
			if err != nil {
				return nil, err
			}

			tokenStellarAddrVal, err = randomScAddress()
			if err != nil {
				return nil, err
			}

		default:
			// Assume that Asset it an ethereum asset
			ethAddress := asset.Address()
			// Check if the string is a valid length (20 byte)
			if len(ethAddress) != 20 { //nolint:gomnd
				return nil, errors.New("unexpected asset type")
			}
			tokenEthAddrVal = ethAddress
			tokenStellarAddrVal, err = randomScAddress()
			if err != nil {
				return nil, err
			}
		}

		token := Asset{
			Chain:          lidvec,
			StellarAddress: tokenStellarAddrVal,
			EthAddress:     tokenEthAddrVal,
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// MakeBalances converts a channel.Allocation to a Balances.
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
	if numParts < 2 { //nolint:gomnd
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
//
//nolint:gomnd
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

// ToBigInt converts xdr.Int128Parts to a big.Int.
//
//nolint:gomnd
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

	backendIDs := make([]wallet.BackendID, wtypes.StellarBackendID)

	alloc := channel.NewAllocation(numParts, backendIDs, assets...)

	for i := range assets {
		alloc.Balances[i] = []*big.Int{balsA[i], balsB[i]}
	}

	alloc.Locked = make([]channel.SubAlloc, 0)

	if err := alloc.Valid(); err != nil {
		return nil, err
	}

	return alloc, nil
}

// Generates a random 32-byte slice.
func random32Bytes() ([32]byte, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return [32]byte{}, fmt.Errorf("error generating random bytes: %w", err)
	}
	return b, nil
}

// randomScAddress generates a random Stellar address as sending an uninitialized address fails.
func randomScAddress() (xdr.ScAddress, error) {
	contractIDBytes, err := random32Bytes()
	if err != nil {
		return xdr.ScAddress{}, fmt.Errorf("error generating random contract ID: %w", err)
	}

	// Return the random xdr.ScAddress for a contract
	return xdr.ScAddress{
		Type:       xdr.ScAddressTypeScAddressTypeContract,
		ContractId: (*xdr.Hash)(&contractIDBytes),
	}, nil
}
