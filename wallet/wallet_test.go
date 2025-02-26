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

package wallet_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	gptest "perun.network/go-perun/wallet/test"
	pkgtest "polycry.pt/poly-go/test"

	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wallet/types"
)

// TestEphemeralWallet tests the ephemeral wallet implementation.
func TestEphemeralWallet(t *testing.T) {
	rng := pkgtest.Prng(t)
	w := wallet.NewEphemeralWallet()

	acc, _, err := w.AddNewAccount(rng)
	require.NoError(t, err)

	unlockedAccount, err := w.Unlock(acc.Address())
	require.NoError(t, err)
	require.Equal(t, acc.Address(), unlockedAccount.Address())

	msg := []byte("hello world")
	sig, err := unlockedAccount.SignData(msg)
	require.NoError(t, err)

	valid, err := wallet.Backend.VerifySignature(msg, sig, acc.Address())
	require.NoError(t, err)
	require.True(t, valid)
}

func setup(rng *rand.Rand) *gptest.Setup {
	w := wallet.NewEphemeralWallet()
	acc, _, err := w.AddNewAccount(rng)
	if err != nil {
		panic(err)
	}
	acc2, _, err := w.AddNewAccount(rng)
	if err != nil {
		panic(err)
	}
	binAddr2, err := acc2.Address().MarshalBinary()
	if err != nil {
		panic(err)
	}
	z, err := types.ZeroAddress()
	if err != nil {
		panic(err)
	}
	return &gptest.Setup{
		Backend:           wallet.Backend,
		Wallet:            w,
		AddressInWallet:   acc.Address(),
		ZeroAddress:       z,
		DataToSign:        []byte("pls sign me"),
		AddressMarshalled: binAddr2,
	}
}

// TestAddress tests the address implementation.
func TestAddress(t *testing.T) {
	rng := pkgtest.Prng(t)
	gptest.TestAddress(t, setup(rng))
}

// TestSignature tests the signature implementation.
func TestGenericSignatureSizeTest(t *testing.T) {
	rng := pkgtest.Prng(t)
	gptest.GenericSignatureSizeTest(t, setup(rng))
}

// TestAccountWithWalletAndBackend tests the account implementation.
func TestAccountWithWalletAndBackend(t *testing.T) {
	rng := pkgtest.Prng(t)
	gptest.TestAccountWithWalletAndBackend(t, setup(rng))
}
