package wire

import (
	"bytes"
	"errors"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/wallet/types"
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
	if err != nil {
		return xdr.ScVal{}, err
	}
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

func MakeParams(params channel.Params) (Params, error) {
	if !params.LedgerChannel {
		return Params{}, errors.New("expected ledger channel")
	}
	if params.VirtualChannel {
		return Params{}, errors.New("expected non-virtual channel")
	}
	if !channel.IsNoApp(params.App) {
		return Params{}, errors.New("expected no app")
	}

	if len(params.Parts) != 2 {
		return Params{}, errors.New("expected exactly two participants")
	}

	participantA, err := types.ToParticipant(params.Parts[0])
	if err != nil {
		return Params{}, err
	}
	a, err := MakeParticipant(*participantA)
	if err != nil {
		return Params{}, err
	}
	participantB, err := types.ToParticipant(params.Parts[1])
	if err != nil {
		return Params{}, err
	}
	b, err := MakeParticipant(*participantB)
	if err != nil {
		return Params{}, err
	}
	nonce := MakeNonce(params.Nonce)
	return Params{
		A:                 a,
		B:                 b,
		Nonce:             nonce,
		ChallengeDuration: xdr.Uint64(params.ChallengeDuration),
	}, nil
}

func MustMakeParams(params channel.Params) Params {
	p, err := MakeParams(params)
	if err != nil {
		panic(err)
	}
	return p
}

func ToParams(params Params) (channel.Params, error) {
	participantA, err := ToParticipant(params.A)
	if err != nil {
		return channel.Params{}, err
	}
	participantB, err := ToParticipant(params.B)
	if err != nil {
		return channel.Params{}, err
	}

	challengeDuration := uint64(params.ChallengeDuration)
	parts := []wallet.Address{&participantA, &participantB}
	app := channel.NoApp()
	nonce := ToNonce(params.Nonce)
	ledgerChannel := true
	virtualChannel := false

	perunParams, err := channel.NewParams(challengeDuration, parts, app, nonce, ledgerChannel, virtualChannel)
	if err != nil {
		return channel.Params{}, err
	}

	return *perunParams, nil
}

func MakeNonce(nonce channel.Nonce) xdr.ScBytes {
	return nonce.FillBytes(make([]byte, NonceLength))
}

func ToNonce(bytes xdr.ScBytes) channel.Nonce {
	return channel.NonceFromBytes(bytes[:])
}
