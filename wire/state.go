package wire

import (
	"bytes"
	"errors"
	"fmt"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wire/scval"
)

const ChannelIDLength = 32

const (
	SymbolStateChannelID xdr.ScSymbol = "channel_id"
	SymbolStateBalances  xdr.ScSymbol = "balances"
	SymbolStateVersion   xdr.ScSymbol = "version"
	SymbolStateFinalized xdr.ScSymbol = "finalized"
)

type State struct {
	ChannelID xdr.ScBytes
	Balances  Balances
	Version   xdr.Uint64
	Finalized bool
}

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

func ToState(stellarState State) (channel.State, error) {
	ChanID, err := scBytesToByteArray(stellarState.ChannelID)
	if err != nil {
		return channel.State{}, err
	}

	BalA, err := ToBigInt(stellarState.Balances.BalA)
	if err != nil {
		return channel.State{}, err
	}
	BalB, err := ToBigInt(stellarState.Balances.BalB)
	if err != nil {
		return channel.State{}, err
	}

	Assets, err := convertAsset(stellarState.Balances.Token)
	if err != nil {
		return channel.State{}, err
	}

	Alloc, err := makeAllocation(Assets, BalA, BalB)
	if err != nil {
		return channel.State{}, err
	}

	PerunState := channel.State{ID: ChanID,
		Version:    uint64(stellarState.Version),
		Allocation: Alloc,
		IsFinal:    stellarState.Finalized,
		App:        channel.NoApp(),
		Data:       channel.NoData(),
	}

	if PerunState.Valid() != nil {
		return channel.State{}, err
	}

	return PerunState, nil
}

func convertAsset(contractID xdr.ScAddress) ([]channel.Asset, error) {
	stellarAsset, err := types.NewStellarAssetFromScAddress(contractID)
	if err != nil {
		return nil, err
	}
	return []channel.Asset{stellarAsset}, nil
}
