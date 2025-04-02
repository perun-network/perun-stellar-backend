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
	"errors"
	"math/rand"

	"github.com/stellar/go/keypair"
	"perun.network/go-perun/wallet"
	"polycry.pt/poly-go/sync"

	"perun.network/perun-stellar-backend/wallet/types"
)

// EphemeralWallet is a wallet that stores accounts in memory.
type EphemeralWallet struct {
	lock     sync.Mutex
	accounts map[string]*Account
}

// Unlock unlocks the account associated with the given address.
func (e *EphemeralWallet) Unlock(a wallet.Address) (wallet.Account, error) {
	addr, ok := a.(*types.Participant)
	if !ok {
		return nil, errors.New("incorrect Participant type")
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	account, ok := e.accounts[addr.String()]
	if !ok {
		return nil, errors.New("account not found")
	}
	return account, nil
}

// LockAll locks all accounts.
func (e *EphemeralWallet) LockAll() {}

// IncrementUsage increments the usage counter of the account associated with the given address.
func (e *EphemeralWallet) IncrementUsage(address wallet.Address) {}

// DecrementUsage decrements the usage counter of the account associated with the given address.
func (e *EphemeralWallet) DecrementUsage(address wallet.Address) {}

// AddNewAccount generates a new account and adds it to the wallet.
func (e *EphemeralWallet) AddNewAccount(rng *rand.Rand) (*Account, *keypair.Full, error) {
	acc, kp, err := NewRandomAccount(rng)
	if err != nil {
		return nil, nil, err
	}
	return acc, kp, e.AddAccount(acc)
}

// AddAccount adds the given account to the wallet.
func (e *EphemeralWallet) AddAccount(acc *Account) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	k := types.AsParticipant(acc.Address()).String()
	if _, ok := e.accounts[k]; ok {
		return errors.New("account already exists")
	}
	e.accounts[k] = acc
	return nil
}

// NewEphemeralWallet creates a new EphemeralWallet instance.
func NewEphemeralWallet() *EphemeralWallet {
	return &EphemeralWallet{
		accounts: make(map[string]*Account),
	}
}
