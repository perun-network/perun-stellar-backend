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

package wallet

import (
	"crypto/ecdsa"
	"math/rand"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/pkg/errors"
	"github.com/stellar/go/keypair"
	"perun.network/go-perun/wallet"

	"perun.network/perun-stellar-backend/wallet/types"
)

// Account is used for signing channel state.
type Account struct {
	// privateKey is the private key of the account.
	privateKey ecdsa.PrivateKey
	// ParticipantAddress references the ParticipantAddress of the Participant this account belongs to.
	ParticipantAddress keypair.FromAddress
	// CCAddr is the cross-chain address of the participant.
	CCAddr [types.CCAddressLength]byte
}

// NewAccount creates a new account with the given private key and addresses.
func NewAccount(privateKey string, addr keypair.FromAddress, ccAddr [types.CCAddressLength]byte) *Account {
	privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		panic(errors.Wrap(err, "NewAccount"))
	}
	return &Account{*privateKeyECDSA, addr, ccAddr}
}

// NewRandomAccountWithAddress creates a new account with a random private key and the given address as
// Account.ParticipantAddress.
func NewRandomAccountWithAddress(rng *rand.Rand, addr *keypair.FromAddress) (*Account, error) {
	s, err := ecdsa.GenerateKey(secp256k1.S256(), rng)
	if err != nil {
		return nil, err
	}
	return &Account{privateKey: *s, ParticipantAddress: *addr}, nil
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

// NewRandomAddress creates a new random address.
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
	pubKey, ok := a.privateKey.Public().(*ecdsa.PublicKey) // Ensure correct type
	if !ok {
		panic("unexpected type for ecdsa.PublicKey")
	}
	return types.NewParticipant(a.ParticipantAddress, pubKey, a.CCAddr)
}

// Participant returns the Participant this account belongs to.
func (a Account) Participant() *types.Participant {
	return types.NewParticipant(a.ParticipantAddress, a.privateKey.Public().(*ecdsa.PublicKey), a.CCAddr)
}

// SignData signs the given data with the account's private key.
func (a Account) SignData(data []byte) ([]byte, error) {
	hash := crypto.Keccak256(data)
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	phash := crypto.Keccak256(prefix, hash)

	sig, err := crypto.Sign(phash, &a.privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "SignHash")
	}
	sig[64] += 27
	return sig, nil
}
