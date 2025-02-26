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

package types

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/strkey"
	"perun.network/go-perun/wallet"
)

const (
	StellarBackendID     = 2
	CCAddressLength      = 20
	StellarAddressLength = 32
)

// Participant is the backend's version of the on-chain participant in the Perun smart contract on stellar.
type Participant struct {
	// Address is the stellar ParticipantAddress of the participant.
	StellarAddress keypair.FromAddress
	// PublicKey is the public key of the participant, which is used to verify signatures on channel state.
	StellarPubKey *ecdsa.PublicKey
	// CCAddr is the cross-chain address of the participant.
	CCAddr [CCAddressLength]byte
}

func NewParticipant(addr keypair.FromAddress, pk *ecdsa.PublicKey, ccAddr [CCAddressLength]byte) *Participant {
	return &Participant{
		StellarAddress: addr,
		StellarPubKey:  pk,
		CCAddr:         ccAddr,
	}
}

// MarshalBinary encodes the participant into binary form.
func (p Participant) MarshalBinary() (data []byte, err error) {
	// Marshal the Stellar public key using secp256k1's raw byte format (uncompressed)
	//nolint:staticcheck
	pubKeyBytes := elliptic.Marshal(p.StellarPubKey.Curve, p.StellarPubKey.X, p.StellarPubKey.Y)

	binAddr, err := p.StellarAddress.MarshalBinary()
	if err != nil {
		return nil, err
	}

	res := make([]byte, len(pubKeyBytes)+len(binAddr)+CCAddressLength)
	copy(res, pubKeyBytes)
	copy(res[len(pubKeyBytes):], binAddr)
	copy(res[len(pubKeyBytes)+len(binAddr):], p.CCAddr[:])

	return res, nil
}

// UnmarshalBinary decodes the participant from binary form.
func (p *Participant) UnmarshalBinary(data []byte) error {
	// Check the minimum length required for the public key
	if len(data) < 65 { //nolint:gomnd
		return fmt.Errorf("invalid data size for public key")
	}

	// Unmarshal the public key (first 65 bytes)
	//nolint:staticcheck
	x, y := elliptic.Unmarshal(secp256k1.S256(), data[:65])
	if x == nil || y == nil {
		return fmt.Errorf("failed to unmarshal ECDSA public key")
	}
	p.StellarPubKey = &ecdsa.PublicKey{
		Curve: secp256k1.S256(),
		X:     x,
		Y:     y,
	}

	// Unmarshal the Stellar address (assuming the next part is for Stellar address)
	offset := 65
	binAddrLength := len(data) - offset - CCAddressLength
	if binAddrLength <= 0 || offset+binAddrLength > len(data) {
		return fmt.Errorf("invalid data size for Stellar address")
	}

	// Unmarshal Stellar address
	if err := p.StellarAddress.UnmarshalBinary(data[offset : offset+binAddrLength]); err != nil {
		return fmt.Errorf("failed to unmarshal Stellar address: %w", err)
	}

	// Unmarshal CCAddr (last part)
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
	pubKeyBytes, _ := x509.MarshalPKIXPublicKey(&p.StellarPubKey)
	return hex.EncodeToString(pubKeyBytes)
}

func (p Participant) BackendID() wallet.BackendID {
	return StellarBackendID
}

func ZeroAddress() (*Participant, error) {
	// Create a zero-value ECDSA public key (X = 0, Y = 0)
	zeroPubKey := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int),
		Y:     new(big.Int),
	}

	stellarAddr, err := keypair.Random() // Generate a random Stellar keypair for the zero address.
	if err != nil {
		return nil, err
	}

	return &Participant{
		StellarAddress: *stellarAddr.FromAddress(),
		StellarPubKey:  &zeroPubKey,
		CCAddr:         [CCAddressLength]byte{},
	}, nil
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
