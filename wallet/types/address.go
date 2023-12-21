package types

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/strkey"
	"perun.network/go-perun/wallet"
)

// Participant is the backend's version of the on-chain participant in the Perun smart contract on stellar.
type Participant struct {
	// Address is the stellar ParticipantAddress of the participant.
	Address keypair.FromAddress
	// PublicKey is the public key of the participant, which is used to verify signatures on channel state.
	PublicKey ed25519.PublicKey
}

func NewParticipant(addr keypair.FromAddress, pk ed25519.PublicKey) *Participant {
	return &Participant{
		Address:   addr,
		PublicKey: pk,
	}
}

// MarshalBinary encodes the participant into binary form.
func (p Participant) MarshalBinary() (data []byte, err error) {
	if len(p.PublicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: %d", len(p.PublicKey))
	}
	binAddr, err := p.Address.MarshalBinary()
	if err != nil {
		return nil, err
	}
	res := make([]byte, ed25519.PublicKeySize+len(binAddr))
	copy(res, p.PublicKey)
	copy(res[ed25519.PublicKeySize:], binAddr)
	return res, nil
}

// UnmarshalBinary decodes the participant from binary form.
func (p *Participant) UnmarshalBinary(data []byte) error {
	if len(data) < ed25519.PublicKeySize {
		return fmt.Errorf("invalid data size: %d", len(data))
	}
	p.PublicKey = data[:ed25519.PublicKeySize]
	p.Address = keypair.FromAddress{}
	return p.Address.UnmarshalBinary(data[ed25519.PublicKeySize:])
}

// String returns the string representation of the participant as [ParticipantAddress string]:[public key hex].
func (p Participant) String() string {
	return p.AddressString() // + ":" + p.PublicKeyString()
}

func (p Participant) Equal(other wallet.Address) bool {
	otherAddress, ok := other.(*Participant)
	if !ok {
		return false
	}
	return p.Address.Equal(&otherAddress.Address) && p.PublicKey.Equal(otherAddress.PublicKey)
}

func (p Participant) AddressString() string {
	return p.Address.Address()
}

func (p Participant) PublicKeyString() string {
	return hex.EncodeToString(p.PublicKey)
}

func ZeroAddress() (*Participant, error) {
	zeros := [32]byte{}
	pk := ed25519.PublicKey(zeros[:])
	addr, err := strkey.Encode(strkey.VersionByteAccountID, pk)
	if err != nil {
		return nil, err
	}
	a := &Participant{}
	err = a.Address.UnmarshalText([]byte(addr))
	a.PublicKey = pk
	return a, err
}

func AsParticipant(address wallet.Address) *Participant {
	p, ok := address.(*Participant)
	if !ok {
		panic("ParticipantAddress has invalid type")
	}
	return p
}

func ToParticipant(address wallet.Address) (*Participant, error) {
	p, ok := address.(*Participant)
	if !ok {
		return nil, fmt.Errorf("address has invalid type")
	}
	return p, nil
}

func PublicKeyFromKeyPair(kp keypair.KP) (ed25519.PublicKey, error) {
	return strkey.Decode(strkey.VersionByteAccountID, kp.Address())
}
