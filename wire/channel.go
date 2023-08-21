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

type Channel struct {
	Params  Params
	State   State
	Control Control
}

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
	return scval.WrapScMap(m)
}

func (c *Channel) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map")
	}
	if len(*m) != 3 {
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

func (c Channel) EncodeTo(e *xdr3.Encoder) error {
	v, err := c.ToScVal()
	if err != nil {
		return err
	}
	return v.EncodeTo(e)
}

func (c *Channel) DecodeFrom(d *xdr3.Decoder) (int, error) {
	var v xdr.ScVal
	i, err := d.Decode(&v)
	if err != nil {
		return i, err
	}
	return i, c.FromScVal(v)
}

func (c Channel) MarshalBinary() ([]byte, error) {
	buf := bytes.Buffer{}
	e := xdr3.NewEncoder(&buf)
	err := c.EncodeTo(e)
	return buf.Bytes(), err
}

func (c *Channel) UnmarshalBinary(data []byte) error {
	d := xdr3.NewDecoder(bytes.NewReader(data))
	_, err := c.DecodeFrom(d)
	return err
}

func ChannelFromScVal(v xdr.ScVal) (Channel, error) {
	var p Channel
	err := (&p).FromScVal(v)
	return p, err
}

func MakeChannel(p Params, s State, c Control) Channel {
	return Channel{
		Params:  p,
		State:   s,
		Control: c,
	}
}
