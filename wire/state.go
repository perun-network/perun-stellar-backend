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
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wallet"

	"perun.network/perun-stellar-backend/channel/types"
	wtypes "perun.network/perun-stellar-backend/wallet/types"
	"perun.network/perun-stellar-backend/wire/scval"
)

const ChannelIDLength = 32

const (
	SymbolStateChannelID xdr.ScSymbol = "channel_id"
	SymbolStateBalances  xdr.ScSymbol = "balances"
	SymbolStateVersion   xdr.ScSymbol = "version"
	SymbolStateFinalized xdr.ScSymbol = "finalized"
)

// State represents the state of a channel.
type State struct {
	ChannelID xdr.ScBytes
	Balances  Balances
	Version   xdr.Uint64
	Finalized bool
}

// ToScVal encodes a State to an xdr.ScVal.
func (s State) ToScVal() (xdr.ScVal, error) {
	if len(s.ChannelID) != ChannelIDLength {
		return xdr.ScVal{}, errors.New("invalid channel id length")
	}
	channelID, err := scval.WrapScBytes(s.ChannelID)
	if err != nil {
		return xdr.ScVal{}, err
	}
	balances, err := s.Balances.ToScVal()
	if err != nil {
		return xdr.ScVal{}, err
	}
	version, err := scval.WrapUint64(s.Version)
	if err != nil {
		return xdr.ScVal{}, err
	}
	finalized, err := scval.WrapBool(s.Finalized)
	if err != nil {
		return xdr.ScVal{}, err
	}
	m, err := MakeSymbolScMap(
		[]xdr.ScSymbol{
			SymbolStateChannelID,
			SymbolStateBalances,
			SymbolStateVersion,
			SymbolStateFinalized,
		},
		[]xdr.ScVal{channelID, balances, version, finalized},
	)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return scval.WrapScMap(m)
}

// FromScVal decodes a State from an xdr.ScVal.
func (s *State) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map decoding State")
	}
	if len(*m) != 4 { //nolint:gomnd
		return errors.New("expected map of length 4")
	}
	channelIDVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolStateChannelID), *m)
	if err != nil {
		return err
	}
	channelID, ok := channelIDVal.GetBytes()
	if !ok {
		return errors.New("expected bytes")
	}
	if len(channelID) != ChannelIDLength {
		return errors.New("invalid channel id length")
	}

	balancesVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolStateBalances), *m)
	if err != nil {
		return err
	}
	balances, err := BalancesFromScVal(balancesVal)
	if err != nil {
		return err
	}
	versionVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolStateVersion), *m)
	if err != nil {
		return err
	}
	version, ok := versionVal.GetU64()
	if !ok {
		return errors.New("expected uint64")
	}
	finalizedVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolStateFinalized), *m)
	if err != nil {
		return err
	}
	finalized, ok := finalizedVal.GetB()
	if !ok {
		return errors.New("expected bool")
	}
	s.ChannelID = channelID
	s.Balances = balances
	s.Version = version
	s.Finalized = finalized
	return nil
}

// EncodeTo encodes a State to a xdr.Encoder.
func (s State) EncodeTo(e *xdr3.Encoder) error {
	v, err := s.ToScVal()
	if err != nil {
		return err
	}
	return v.EncodeTo(e)
}

// DecodeFrom decodes a State from a xdr.Decoder.
func (s *State) DecodeFrom(d *xdr3.Decoder) (int, error) {
	var v xdr.ScVal
	i, err := d.Decode(&v)
	if err != nil {
		return i, err
	}
	return i, s.FromScVal(v)
}

// MarshalBinary encodes a State to binary data.
func (s State) MarshalBinary() ([]byte, error) {
	buf := bytes.Buffer{}
	e := xdr3.NewEncoder(&buf)
	err := s.EncodeTo(e)
	return buf.Bytes(), err
}

// UnmarshalBinary decodes a State from binary data.
func (s *State) UnmarshalBinary(data []byte) error {
	d := xdr3.NewDecoder(bytes.NewReader(data))
	_, err := s.DecodeFrom(d)
	return err
}

// StateFromScVal creates a State from a xdr.ScVal.
func StateFromScVal(v xdr.ScVal) (State, error) {
	var s State
	err := (&s).FromScVal(v)
	return s, err
}

// MakeState creates a State from a channel.State.
func MakeState(state channel.State) (State, error) {
	if err := state.Valid(); err != nil {
		return State{}, err
	}
	if !channel.IsNoApp(state.App) {
		return State{}, errors.New("expected NoApp")
	}
	if !channel.IsNoData(state.Data) {
		return State{}, errors.New("expected NoData")
	}
	balances, err := MakeBalances(state.Allocation)
	if err != nil {
		return State{}, err
	}
	return State{
		ChannelID: state.ID[:],
		Balances:  balances,
		Version:   xdr.Uint64(state.Version),
		Finalized: state.IsFinal,
	}, nil
}

func scBytesToByteArray(bytesXdr xdr.ScBytes) ([types.HashLenXdr]byte, error) {
	if len(bytesXdr) != types.HashLenXdr {
		return [types.HashLenXdr]byte{}, fmt.Errorf("expected length of %d bytes, got %d", types.HashLenXdr, len(bytesXdr))
	}
	var arr [types.HashLenXdr]byte
	copy(arr[:], bytesXdr[:types.HashLenXdr])
	return arr, nil
}

//nolint:unused
func scMapToMap(mapXdr xdr.ScMap) (map[wallet.BackendID][types.HashLenXdr]byte, error) {
	result := make(map[wallet.BackendID][types.HashLenXdr]byte)

	for _, entry := range mapXdr {
		backendID, err := parseBackendID(entry.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse BackendID from key: %w", err)
		}

		bytesXdr := entry.Val.MustBytes()

		var arr [types.HashLenXdr]byte
		copy(arr[:], bytesXdr)

		result[backendID] = arr
	}

	return result, nil
}

//nolint:unused
func parseBackendID(key xdr.ScVal) (wallet.BackendID, error) {
	sym := key.MustSym()

	backendID, err := strconv.Atoi(string(sym))
	if err != nil {
		return 0, fmt.Errorf("failed to convert symbol to BackendID: %w", err)
	}

	return wallet.BackendID(backendID), nil
}

// ToState converts a channel.State to a State.
func ToState(stellarState State) (channel.State, error) {
	ChanID, err := scBytesToByteArray(stellarState.ChannelID)
	if err != nil {
		return channel.State{}, err
	}

	var balsABigInt []*big.Int
	var balsBBigInt []*big.Int

	balsA := stellarState.Balances.BalA
	for _, scVal := range balsA { // iterate for balance within asset
		valA := scVal.MustI128()
		balAPerun, err := ToBigInt(valA)
		if err != nil {
			return channel.State{}, err
		}
		balsABigInt = append(balsABigInt, balAPerun)
	}

	balsB := stellarState.Balances.BalB
	for _, scVal := range balsB { // iterate for balance within asset
		valB := scVal.MustI128()
		balBPerun, err := ToBigInt(valB)
		if err != nil {
			return channel.State{}, err
		}
		balsBBigInt = append(balsBBigInt, balBPerun)
	}

	Assets, err := convertAssets(stellarState.Balances.Tokens)
	if err != nil {
		return channel.State{}, err
	}

	Alloc, err := makeAllocationMulti(Assets, balsABigInt, balsBBigInt)
	if err != nil {
		return channel.State{}, err
	}

	PerunState := channel.State{
		ID:         ChanID,
		Version:    uint64(stellarState.Version),
		Allocation: *Alloc,
		IsFinal:    stellarState.Finalized,
		App:        channel.NoApp(),
		Data:       channel.NoData(),
	}

	if PerunState.Valid() != nil {
		return channel.State{}, err
	}

	return PerunState, nil
}

func convertAssets(tokens []Asset) ([]channel.Asset, error) {
	var assets []channel.Asset

	for _, val := range tokens {
		defaultAddr := xdr.ScAddress{}
		if val.StellarAddress != defaultAddr {
			var steAsset types.StellarAsset
			err := steAsset.FromScAddress(val.StellarAddress)
			if err != nil {
				return nil, err
			}
			assets = append(assets, &steAsset)
		} else {
			ethAsset, err := createEthAsset(val.Chain, val.EthAddress)
			if err != nil {
				return nil, err
			}
			assets = append(assets, ethAsset)
		}
	}
	return assets, nil
}

func createEthAsset(chain xdr.ScVec, address xdr.ScBytes) (channel.Asset, error) {
	bytes, err := address.MarshalBinary()
	if err != nil || len(bytes) != 20 {
		return nil, errors.New("invalid byte length for EthAsset")
	}
	var ethAddrArray [20]byte
	copy(ethAddrArray[:], bytes)
	newEthAddr := common.BytesToAddress(ethAddrArray[:])
	ethAddr := wtypes.EthAddress(newEthAddr)
	chainID := new(big.Int)
	chainID.SetUint64(uint64(chain[0].MustU64()))
	ethAsset := types.MakeEthAsset(chainID, &ethAddr)
	return &ethAsset, nil
}
