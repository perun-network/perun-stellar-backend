package wire

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	assettypes "perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wallet/types"

	"perun.network/perun-stellar-backend/wire/scval"
)

const (
	PubKeyLength                         = 32
	SymbolParticipantAddr   xdr.ScSymbol = "addr"
	SymbolParticipantPubKey xdr.ScSymbol = "pubkey"
)

type Participant struct {
	Addr   xdr.ScAddress
	PubKey xdr.ScBytes
}

func (p Participant) ToScVal() (xdr.ScVal, error) {
	addr, err := scval.WrapScAddress(p.Addr)
	if err != nil {
		return xdr.ScVal{}, err
	}
	if len(p.PubKey) != PubKeyLength {
		return xdr.ScVal{}, errors.New("invalid public key length")
	}
	pubKey, err := scval.WrapScBytes(p.PubKey)
	if err != nil {
		return xdr.ScVal{}, err
	}
	m, err := MakeSymbolScMap(
		[]xdr.ScSymbol{
			SymbolParticipantAddr,
			SymbolParticipantPubKey,
		},
		[]xdr.ScVal{addr, pubKey},
	)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return scval.WrapScMap(m)
}

func (p *Participant) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map")
	}
	if len(*m) != 2 {
		return errors.New("expected map of length 2")
	}
	addrVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolParticipantAddr), *m)
	if err != nil {
		return err
	}
	addr, ok := addrVal.GetAddress()
	if !ok {
		return errors.New("expected address")
	}
	pubKeyVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolParticipantPubKey), *m)
	if err != nil {
		return err
	}
	pubKey, ok := pubKeyVal.GetBytes()
	if !ok {
		return errors.New("expected bytes")
	}
	if len(pubKey) != PubKeyLength {
		return errors.New("invalid public key length")
	}
	p.Addr = addr
	p.PubKey = pubKey
	return nil
}

func (p Participant) EncodeTo(e *xdr3.Encoder) error {
	v, err := p.ToScVal()
	if err != nil {
		return err
	}
	return v.EncodeTo(e)
}

func (p *Participant) DecodeFrom(d *xdr3.Decoder) (int, error) {
	var v xdr.ScVal
	i, err := d.Decode(&v)
	if err != nil {
		return i, err
	}
	return i, p.FromScVal(v)
}

func (p Participant) MarshalBinary() ([]byte, error) {
	buf := bytes.Buffer{}
	e := xdr3.NewEncoder(&buf)
	err := p.EncodeTo(e)
	return buf.Bytes(), err
}

func (p *Participant) UnmarshalBinary(data []byte) error {
	d := xdr3.NewDecoder(bytes.NewReader(data))
	_, err := p.DecodeFrom(d)
	return err
}

func ParticipantFromScVal(v xdr.ScVal) (Participant, error) {
	var p Participant
	err := (&p).FromScVal(v)
	return p, err
}

func MakeParticipant(participant types.Participant) (Participant, error) {
	addr, err := assettypes.MakeAccountAddress(&participant.Address)
	if err != nil {
		return Participant{}, err
	}
	if len(participant.PublicKey) != PubKeyLength {
		return Participant{}, errors.New("invalid public key length")
	}
	pubKey := xdr.ScBytes(participant.PublicKey)
	return Participant{
		Addr:   addr,
		PubKey: pubKey,
	}, nil
}

func ToParticipant(participant Participant) (types.Participant, error) {
	kp, err := assettypes.ToAccountAddress(participant.Addr)
	if err != nil {
		return types.Participant{}, err
	}
	if len(participant.PubKey) != ed25519.PublicKeySize {
		return types.Participant{}, errors.New("invalid public key length")
	}
	return *types.NewParticipant(kp, ed25519.PublicKey(participant.PubKey)), nil
}
