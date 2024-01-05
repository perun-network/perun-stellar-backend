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
	"bytes"
	"errors"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	"perun.network/perun-stellar-backend/wire/scval"
)

const (
	SymbolControlFundedA    = "funded_a"
	SymbolControlFundedB    = "funded_b"
	SymbolControlClosed     = "closed"
	SymbolControlWithdrawnA = "withdrawn_a"
	SymbolControlWithdrawnB = "withdrawn_b"
	SymbolControlDisputed   = "disputed"
	SymbolControlTimestamp  = "timestamp"
)

type Control struct {
	FundedA    bool
	FundedB    bool
	Closed     bool
	WithdrawnA bool
	WithdrawnB bool
	Disputed   bool
	Timestamp  xdr.Uint64
}

func (c Control) ToScVal() (xdr.ScVal, error) {
	fundedA, err := scval.WrapBool(c.FundedA)
	if err != nil {
		return xdr.ScVal{}, err
	}
	fundedB, err := scval.WrapBool(c.FundedB)
	if err != nil {
		return xdr.ScVal{}, err
	}
	closed, err := scval.WrapBool(c.Closed)
	if err != nil {
		return xdr.ScVal{}, err
	}
	withdrawnA, err := scval.WrapBool(c.WithdrawnA)
	if err != nil {
		return xdr.ScVal{}, err
	}
	withdrawnB, err := scval.WrapBool(c.WithdrawnB)
	if err != nil {
		return xdr.ScVal{}, err
	}
	disputed, err := scval.WrapBool(c.Disputed)
	if err != nil {
		return xdr.ScVal{}, err
	}
	timestamp, err := scval.WrapUint64(c.Timestamp)
	if err != nil {
		return xdr.ScVal{}, err
	}

	m, err := MakeSymbolScMap(
		[]xdr.ScSymbol{
			SymbolControlFundedA,
			SymbolControlFundedB,
			SymbolControlClosed,
			SymbolControlWithdrawnA,
			SymbolControlWithdrawnB,
			SymbolControlDisputed,
			SymbolControlTimestamp,
		},
		[]xdr.ScVal{fundedA, fundedB, closed, withdrawnA, withdrawnB, disputed, timestamp},
	)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return scval.WrapScMap(m)
}

func (c *Control) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map")
	}
	if len(*m) != 7 {
		return errors.New("expected map of length 7")
	}
	fundedAVal, err := GetScMapValueFromSymbol(SymbolControlFundedA, *m)
	if err != nil {
		return err
	}
	fundedA, ok := fundedAVal.GetB()
	if !ok {
		return errors.New("expected bool")
	}
	fundedBVal, err := GetScMapValueFromSymbol(SymbolControlFundedB, *m)
	if err != nil {
		return err
	}
	fundedB, ok := fundedBVal.GetB()
	if !ok {
		return errors.New("expected bool")
	}
	closedVal, err := GetScMapValueFromSymbol(SymbolControlClosed, *m)
	if err != nil {
		return err
	}
	closed, ok := closedVal.GetB()
	if !ok {
		return errors.New("expected bool")
	}
	withdrawnAVal, err := GetScMapValueFromSymbol(SymbolControlWithdrawnA, *m)
	if err != nil {
		return err
	}
	withdrawnA, ok := withdrawnAVal.GetB()
	if !ok {
		return errors.New("expected bool")
	}
	withdrawnBVal, err := GetScMapValueFromSymbol(SymbolControlWithdrawnB, *m)
	if err != nil {
		return err
	}
	withdrawnB, ok := withdrawnBVal.GetB()
	if !ok {
		return errors.New("expected bool")
	}
	disputedVal, err := GetScMapValueFromSymbol(SymbolControlDisputed, *m)
	if err != nil {
		return err
	}
	disputed, ok := disputedVal.GetB()
	if !ok {
		return errors.New("expected bool")
	}
	timestampVal, err := GetScMapValueFromSymbol(SymbolControlTimestamp, *m)
	if err != nil {
		return err
	}
	timestamp, ok := timestampVal.GetU64()
	if !ok {
		return errors.New("expected uint64")
	}
	c.FundedA = fundedA
	c.FundedB = fundedB
	c.Closed = closed
	c.WithdrawnA = withdrawnA
	c.WithdrawnB = withdrawnB
	c.Disputed = disputed
	c.Timestamp = timestamp
	return nil
}

func (c Control) EncodeTo(e *xdr3.Encoder) error {
	v, err := c.ToScVal()
	if err != nil {
		return err
	}
	return v.EncodeTo(e)
}

func (c *Control) DecodeFrom(d *xdr3.Decoder) (int, error) {
	var v xdr.ScVal
	i, err := d.Decode(&v)
	if err != nil {
		return i, err
	}
	return i, c.FromScVal(v)
}

func (c Control) MarshalBinary() ([]byte, error) {
	buf := bytes.Buffer{}
	e := xdr3.NewEncoder(&buf)
	err := c.EncodeTo(e)
	return buf.Bytes(), err
}

func (c *Control) UnmarshalBinary(data []byte) error {
	d := xdr3.NewDecoder(bytes.NewReader(data))
	_, err := c.DecodeFrom(d)
	return err
}

func ControlFromScVal(v xdr.ScVal) (Control, error) {
	var p Control
	err := (&p).FromScVal(v)
	return p, err
}
