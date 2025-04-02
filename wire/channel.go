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

	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"

	"perun.network/perun-stellar-backend/wire/scval"
)

const (
	SymbolChannelParams  = "params"
	SymbolChannelState   = "state"
	SymbolChannelControl = "control"
)

// Channel represents a channel on the soroban-contract.
type Channel struct {
	Params  Params
	State   State
	Control Control
}

// ToScVal converts a Channel to an xdr.ScVal.
func (c Channel) ToScVal() (xdr.ScVal, error) {
	params, err := c.Params.ToScVal()
	if err != nil {
		return xdr.ScVal{}, err
	}
	state, err := c.State.ToScVal()
	if err != nil {
		return xdr.ScVal{}, err
	}
	control, err := c.Control.ToScVal()
	if err != nil {
		return xdr.ScVal{}, err
	}
	m, err := MakeSymbolScMap(
		[]xdr.ScSymbol{
			SymbolChannelParams,
			SymbolChannelState,
			SymbolChannelControl,
		},
		[]xdr.ScVal{params, state, control},
	)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return scval.WrapScMap(m)
}

// FromScVal converts an xdr.ScVal to a Channel.
func (c *Channel) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map")
	}
	if len(*m) != 3 { //nolint:gomnd
		return errors.New("expected map of length 3")
	}
	paramsVal, err := GetScMapValueFromSymbol(SymbolChannelParams, *m)
	if err != nil {
		return err
	}
	params, err := ParamsFromScVal(paramsVal)
	if err != nil {
		return err
	}
	stateVal, err := GetScMapValueFromSymbol(SymbolChannelState, *m)
	if err != nil {
		return err
	}
	state, err := StateFromScVal(stateVal)
	if err != nil {
		return err
	}
	controlVal, err := GetScMapValueFromSymbol(SymbolChannelControl, *m)
	if err != nil {
		return err
	}
	control, err := ControlFromScVal(controlVal)
	if err != nil {
		return err
	}
	c.Params = params
	c.State = state
	c.Control = control
	return nil
}

// EncodeTo encodes a Channel to an xdr.Encoder.
func (c Channel) EncodeTo(e *xdr3.Encoder) error {
	v, err := c.ToScVal()
	if err != nil {
		return err
	}
	return v.EncodeTo(e)
}

// DecodeFrom decodes a Channel from an xdr.Decoder.
func (c *Channel) DecodeFrom(d *xdr3.Decoder) (int, error) {
	var v xdr.ScVal
	i, err := d.Decode(&v)
	if err != nil {
		return i, err
	}
	return i, c.FromScVal(v)
}

// MarshalBinary encodes a Channel to a binary format.
func (c Channel) MarshalBinary() ([]byte, error) {
	buf := bytes.Buffer{}
	e := xdr3.NewEncoder(&buf)
	err := c.EncodeTo(e)
	return buf.Bytes(), err
}

// UnmarshalBinary decodes a Channel from a binary format.
func (c *Channel) UnmarshalBinary(data []byte) error {
	d := xdr3.NewDecoder(bytes.NewReader(data))
	_, err := c.DecodeFrom(d)
	return err
}

// ChannelFromScVal converts an xdr.ScVal to a Channel.
func ChannelFromScVal(v xdr.ScVal) (Channel, error) {
	var p Channel
	err := (&p).FromScVal(v)
	return p, err
}

// MakeChannel creates a new Channel.
func MakeChannel(p Params, s State, c Control) Channel {
	return Channel{
		Params:  p,
		State:   s,
		Control: c,
	}
}
