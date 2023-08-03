package wire

import (
	"bytes"
	"errors"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	"perun.network/perun-stellar-backend/wire/scval"
)

type Balances struct {
	BalA  xdr.Int128Parts
	BalB  xdr.Int128Parts
	Token xdr.ScAddress
}

const (
	SymbolBalancesBalA  xdr.ScSymbol = "bal_a"
	SymbolBalancesBalB  xdr.ScSymbol = "bal_b"
	SymbolBalancesToken xdr.ScSymbol = "token"
)

func (b Balances) ToScVal() (xdr.ScVal, error) {
	var err error
	balA, err := scval.WrapInt128Parts(b.BalA)
	if err != nil {
		return xdr.ScVal{}, err
	}
	balB, err := scval.WrapInt128Parts(b.BalB)
	if err != nil {
		return xdr.ScVal{}, err
	}
	token, err := scval.WrapScAddress(b.Token)
	if err != nil {
		return xdr.ScVal{}, err
	}
	m, err := MakeSymbolScMap(
		[]xdr.ScSymbol{
			SymbolBalancesBalA,
			SymbolBalancesBalB,
			SymbolBalancesToken,
		},
		[]xdr.ScVal{balA, balB, token},
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
	balA, ok := balAVal.GetI128()
	if !ok {
		return errors.New("expected i128")
	}
	balBVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolBalancesBalB), *m)
	if err != nil {
		return err
	}
	balB, ok := balBVal.GetI128()
	if !ok {
		return errors.New("expected i128")
	}
	tokenVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolBalancesToken), *m)
	if err != nil {
		return err
	}
	token, ok := tokenVal.GetAddress()
	if !ok {
		return errors.New("expected address")
	}
	b.BalA = balA
	b.BalB = balB
	b.Token = token
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
