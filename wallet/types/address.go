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

package types

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/strkey"
	"perun.network/go-perun/wallet"
)

const StellarBackendID = 2
const CCAddressLength = 20
const StellarAddressLength = 32

// Participant is the backend's version of the on-chain participant in the Perun smart contract on stellar.
type Participant struct {
	// Address is the stellar ParticipantAddress of the participant.
	StellarAddress keypair.FromAddress
	// PublicKey is the public key of the participant, which is used to verify signatures on channel state.
	StellarPubKey ed25519.PublicKey
	// CCAddr is the cross-chain address of the participant.
	CCAddr [CCAddressLength]byte
}

func NewParticipant(addr keypair.FromAddress, pk ed25519.PublicKey, ccAddr [CCAddressLength]byte) *Participant {
	return &Participant{
		StellarAddress: addr,
		StellarPubKey:  pk,
		CCAddr:         ccAddr,
	}
}

// MarshalBinary encodes the participant into binary form.
func (p Participant) MarshalBinary() (data []byte, err error) {
	if len(p.StellarPubKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Stellar public key size: %d", len(p.StellarPubKey))
	}
	binAddr, err := p.StellarAddress.MarshalBinary()
	if err != nil {
		return nil, err
	}
	res := make([]byte, ed25519.PublicKeySize+len(binAddr)+CCAddressLength)
	copy(res, p.StellarPubKey)
	copy(res[ed25519.PublicKeySize:], binAddr)
	copy(res[ed25519.PublicKeySize+len(binAddr):], p.CCAddr[:])
	return res, nil
}

// UnmarshalBinary decodes the participant from binary form.
func (p *Participant) UnmarshalBinary(data []byte) error {
	if len(data) < ed25519.PublicKeySize {
		return fmt.Errorf("invalid data size: %d", len(data))
	}
	p.StellarPubKey = data[:ed25519.PublicKeySize]
	p.StellarAddress = keypair.FromAddress{}
	p.CCAddr = [CCAddressLength]byte{}
	offset := ed25519.PublicKeySize

	binAddrLength := len(data) - offset - CCAddressLength

	if binAddrLength <= 0 || offset+binAddrLength > len(data) {
		return fmt.Errorf("invalid data size for Stellar address")
	}
	if err := p.StellarAddress.UnmarshalBinary(data[offset : offset+binAddrLength]); err != nil {
		return fmt.Errorf("failed to unmarshal Stellar address: %w", err)
	}

	offset += binAddrLength
	if len(data[offset:]) != CCAddressLength {
		return fmt.Errorf("invalid cross-chain address size")
	}
	copy(p.CCAddr[:], data[offset:])
	return nil
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
	return p.StellarAddress.Equal(&otherAddress.StellarAddress) && p.StellarPubKey.Equal(otherAddress.StellarPubKey) && p.CCAddr == otherAddress.CCAddr
}

func (p Participant) AddressString() string {
	return p.StellarAddress.Address()
}

func (p Participant) PublicKeyString() string {
	return hex.EncodeToString(p.StellarPubKey)
}

func (p Participant) BackendID() wallet.BackendID {
	return StellarBackendID
}

func ZeroAddress() (*Participant, error) {
	zeros := [32]byte{}
	pk := ed25519.PublicKey(zeros[:])
	stellarAddr, err := strkey.Encode(strkey.VersionByteAccountID, pk)
	if err != nil {
		return nil, err
	}
	a := &Participant{}
	err = a.StellarAddress.UnmarshalText([]byte(stellarAddr))
	a.StellarPubKey = pk
	a.CCAddr = [CCAddressLength]byte{}
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
