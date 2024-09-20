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

package wallet

import (
	"crypto/ed25519"
	"errors"
	"github.com/stellar/go/keypair"
	"math/rand"
	"perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/wallet/types"
)

// Account is used for signing channel state.
type Account struct {
	// privateKey is the private key of the account.
	privateKey ed25519.PrivateKey
	// ParticipantAddress references the ParticipantAddress of the Participant this account belongs to.
	ParticipantAddress keypair.FromAddress
	// CCAddr is the cross-chain address of the participant.
	CCAddr [types.CCAddressLength]byte
}

// NewRandomAccountWithAddress creates a new account with a random private key and the given address as
// Account.ParticipantAddress.
func NewRandomAccountWithAddress(rng *rand.Rand, addr *keypair.FromAddress) (*Account, error) {
	_, s, err := ed25519.GenerateKey(rng)
	if err != nil {
		return nil, err
	}
	return &Account{privateKey: s, ParticipantAddress: *addr, CCAddr: [types.CCAddressLength]byte{}}, nil
}

// NewRandomAccount creates a new account with a random private key. It also creates a random key pair, using its
// address as the account'privateKey Account.ParticipantAddress.
func NewRandomAccount(rng *rand.Rand) (*Account, *keypair.Full, error) {
	kp, err := keypair.Random()
	if err != nil {
		return nil, nil, err
	}
	acc, err := NewRandomAccountWithAddress(rng, kp.FromAddress())
	if err != nil {
		return nil, nil, err
	}
	return acc, kp, nil
}

func NewRandomAddress(rng *rand.Rand) wallet.Address {
	kp, err := keypair.Random()
	if err != nil {
		panic(err)
	}
	acc, err := NewRandomAccountWithAddress(rng, kp.FromAddress())
	if err != nil {
		panic(err)
	}

	addr := acc.Address()

	return addr
}

// Address returns the Participant this account belongs to.
func (a Account) Address() wallet.Address {
	return types.NewParticipant(a.ParticipantAddress, a.privateKey.Public().(ed25519.PublicKey), a.CCAddr)
}

func (a Account) Participant() *types.Participant {
	return types.NewParticipant(a.ParticipantAddress, a.privateKey.Public().(ed25519.PublicKey), a.CCAddr)
}

// SignData signs the given data with the account's private key.
func (a Account) SignData(data []byte) ([]byte, error) {
	if len(a.privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key size")
	}
	return ed25519.Sign(a.privateKey, data), nil
}
