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
	"errors"
	"fmt"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	"math/big"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wire/scval"
	"strconv"
)

const ChannelIDLength = 32

const (
	SymbolStateChannelID xdr.ScSymbol = "channel_id"
	SymbolStateBalances  xdr.ScSymbol = "balances"
	SymbolStateVersion   xdr.ScSymbol = "version"
	SymbolStateFinalized xdr.ScSymbol = "finalized"
)

type State struct {
	ChannelID xdr.ScMap
	Balances  Balances
	Version   xdr.Uint64
	Finalized bool
}

func (s State) ToScVal() (xdr.ScVal, error) {
	if len(s.ChannelID) != 2 {
		return xdr.ScVal{}, errors.New("invalid channel id length")
	}

	channelID, err := scval.WrapScMap(s.ChannelID)
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

func (s *State) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map")
	}
	if len(*m) != 4 {
		return errors.New("expected map of length 4")
	}
	channelIDVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolStateChannelID), *m)
	if err != nil {
		return err
	}

	channelID, ok := channelIDVal.GetMap()
	if !ok {
		return errors.New("expected map")
	}

	for _, v := range *channelID {
		if len(*v.Key.Bytes) != ChannelIDLength {
			return errors.New("invalid channel id length")
		}
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
	s.ChannelID = *channelID
	s.Balances = balances
	s.Version = version
	s.Finalized = finalized
	return nil
}

func (s State) EncodeTo(e *xdr3.Encoder) error {
	v, err := s.ToScVal()
	if err != nil {
		return err
	}
	return v.EncodeTo(e)
}

func (s *State) DecodeFrom(d *xdr3.Decoder) (int, error) {
	var v xdr.ScVal
	i, err := d.Decode(&v)
	if err != nil {
		return i, err
	}
	return i, s.FromScVal(v)
}

func (s State) MarshalBinary() ([]byte, error) {
	buf := bytes.Buffer{}
	e := xdr3.NewEncoder(&buf)
	err := s.EncodeTo(e)
	return buf.Bytes(), err
}

func (s *State) UnmarshalBinary(data []byte) error {
	d := xdr3.NewDecoder(bytes.NewReader(data))
	_, err := s.DecodeFrom(d)
	return err
}

func StateFromScVal(v xdr.ScVal) (State, error) {
	var s State
	err := (&s).FromScVal(v)
	return s, err
}

func MakeState(state channel.State) (State, error) {
	// TODO: Put these checks into a compatibility layer

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

	chanIdMap, err := MakeChannelId(&state)
	if err != nil {
		return State{}, err
	}

	return State{
		ChannelID: chanIdMap,
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

func scMapToMap(MapXdr xdr.ScMap) (map[wallet.BackendID][types.HashLenXdr]byte, error) {
	result := make(map[wallet.BackendID][types.HashLenXdr]byte)

	for _, entry := range MapXdr {
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

func parseBackendID(key xdr.ScVal) (wallet.BackendID, error) {
	sym := key.MustSym()

	backendID, err := strconv.Atoi(string(sym))
	if err != nil {
		return 0, fmt.Errorf("failed to convert symbol to BackendID: %w", err)
	}

	return wallet.BackendID(backendID), nil
}

func ToState(stellarState State) (channel.State, error) {
	ChanID, err := scMapToMap(stellarState.ChannelID)
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

	PerunState := channel.State{ID: ChanID,
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

func convertAsset(contractID xdr.ScAddress) (channel.Asset, error) {
	stellarAsset, err := types.NewStellarAssetFromScAddress(contractID)
	if err != nil {
		return nil, err
	}
	return stellarAsset, nil
}

func convertAssets(contractIDs xdr.ScVec) ([]channel.Asset, error) {

	var assets []channel.Asset

	for _, val := range contractIDs {
		contractID, ok := val.GetAddress()
		if !ok {
			return nil, errors.New("could not turn value into address")
		}
		if contractID.Type != xdr.ScAddressTypeScAddressTypeContract {
			return nil, errors.New("invalid address type")
		}
		asset, err := convertAsset(contractID)
		if err != nil {
			return nil, err
		}
		// return asset, nil

		assets = append(assets, asset)

	}

	return assets, nil
}
