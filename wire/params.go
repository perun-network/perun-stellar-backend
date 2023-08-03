package wire

import (
	"bytes"
	"errors"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	"perun.network/perun-stellar-backend/wire/scval"
)

const NonceLength = 32
const (
	SymbolParamsA                 = "a"
	SymbolParamsB                 = "b"
	SymbolParamsNonce             = "nonce"
	SymbolParamsChallengeDuration = "challenge_duration"
)

type Params struct {
	A                 Participant
	B                 Participant
	Nonce             xdr.ScBytes
	ChallengeDuration xdr.Uint64
}

func (p Params) ToScVal() (xdr.ScVal, error) {
	if len(p.Nonce) != NonceLength {
		return xdr.ScVal{}, errors.New("invalid nonce length")
	}
	a, err := p.A.ToScVal()
	if err != nil {
		return xdr.ScVal{}, err
	}
	b, err := p.B.ToScVal()
	if err != nil {
		return xdr.ScVal{}, err
	}
	nonce, err := scval.WrapScBytes(p.Nonce)
	if err != nil {
		return xdr.ScVal{}, err
	}
	challengeDuration, err := scval.WrapUint64(p.ChallengeDuration)
	if err != nil {
		return xdr.ScVal{}, err
	}
	m, err := MakeSymbolScMap(
		[]xdr.ScSymbol{
			SymbolParamsA,
			SymbolParamsB,
			SymbolParamsNonce,
			SymbolParamsChallengeDuration,
		},
		[]xdr.ScVal{a, b, nonce, challengeDuration},
	)
	return scval.WrapScMap(m)
}

func (p *Params) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map")
	}
	if len(*m) != 4 {
		return errors.New("expected map of length 4")
	}
	aVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolParamsA), *m)
	if err != nil {
		return err
	}
	a, err := ParticipantFromScVal(aVal)
	if err != nil {
		return err
	}
	bVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolParamsB), *m)
	if err != nil {
		return err
	}
	b, err := ParticipantFromScVal(bVal)
	if err != nil {
		return err
	}
	nonceVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolParamsNonce), *m)
	if err != nil {
		return err
	}
	nonce, ok := nonceVal.GetBytes()
	if !ok {
		return errors.New("expected bytes")
	}
	if len(nonce) != NonceLength {
		return errors.New("invalid nonce length")
	}
	challengeDurationVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolParamsChallengeDuration), *m)
	if err != nil {
		return err
	}
	challengeDuration, ok := challengeDurationVal.GetU64()
	if !ok {
		return errors.New("expected uint64")
	}
	p.A = a
	p.B = b
	p.Nonce = nonce
	p.ChallengeDuration = challengeDuration
	return nil
}

func (p Params) EncodeTo(e *xdr3.Encoder) error {
	v, err := p.ToScVal()
	if err != nil {
		return err
	}
	return v.EncodeTo(e)
}

func (p *Params) DecodeFrom(d *xdr3.Decoder) (int, error) {
	var v xdr.ScVal
	i, err := d.Decode(&v)
	if err != nil {
		return i, err
	}
	return i, p.FromScVal(v)
}

func (p Params) MarshalBinary() ([]byte, error) {
	buf := bytes.Buffer{}
	e := xdr3.NewEncoder(&buf)
	err := p.EncodeTo(e)
	return buf.Bytes(), err
}

func (p *Params) UnmarshalBinary(data []byte) error {
	d := xdr3.NewDecoder(bytes.NewReader(data))
	_, err := p.DecodeFrom(d)
	return err
}

func ParamsFromScVal(v xdr.ScVal) (Params, error) {
	var p Params
	err := (&p).FromScVal(v)
	return p, err
}
