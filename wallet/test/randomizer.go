// Copyright 2023 PolyCrypt GmbH
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

package test

import (
	"math/rand"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/wallet/test"
	swallet "perun.network/perun-stellar-backend/wallet"
)

type Randomizer struct {
	Account *swallet.Account
}

// NewRandomAddress implements test.Randomizer
func (r *Randomizer) NewRandomAddress(rng *rand.Rand) wallet.Address {
	return swallet.NewRandomAddress(rng)
}

// NewWallet implements test.Randomizer
func (*Randomizer) NewWallet() test.Wallet {
	panic("unimplemented")
}

// RandomWallet implements test.Randomizer
func (*Randomizer) RandomWallet() test.Wallet {
	panic("unimplemented")
}

var _ test.Randomizer = (*Randomizer)(nil)
