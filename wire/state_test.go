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

package wire_test

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"math/big"
	"perun.network/go-perun/channel"
	ptest "perun.network/go-perun/channel/test"
	_ "perun.network/perun-stellar-backend/channel/test"
	"perun.network/perun-stellar-backend/wire"
	polytest "polycry.pt/poly-go/test"
	"testing"
)

func TestState(t *testing.T) {
	// State XDR generated by soroban contract
	x := []byte{0, 0, 0, 17, 0, 0, 0, 1, 0, 0, 0, 4, 0, 0, 0, 15, 0, 0, 0, 8, 98, 97, 108, 97, 110, 99, 101, 115, 0, 0, 0, 17, 0, 0, 0, 1, 0, 0, 0, 3, 0, 0, 0, 15, 0, 0, 0, 5, 98, 97, 108, 95, 97, 0, 0, 0, 0, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 100, 0, 0, 0, 15, 0, 0, 0, 5, 98, 97, 108, 95, 98, 0, 0, 0, 0, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 200, 0, 0, 0, 15, 0, 0, 0, 5, 116, 111, 107, 101, 110, 0, 0, 0, 0, 0, 0, 18, 0, 0, 0, 1, 48, 207, 108, 223, 126, 81, 182, 30, 80, 205, 206, 164, 29, 186, 161, 49, 19, 220, 169, 154, 82, 230, 27, 46, 112, 182, 98, 53, 61, 128, 204, 23, 0, 0, 0, 15, 0, 0, 0, 10, 99, 104, 97, 110, 110, 101, 108, 95, 105, 100, 0, 0, 0, 0, 0, 13, 0, 0, 0, 32, 135, 93, 16, 248, 119, 202, 15, 58, 94, 69, 71, 246, 71, 210, 225, 253, 36, 204, 170, 27, 91, 210, 4, 152, 129, 94, 12, 28, 183, 59, 230, 206, 0, 0, 0, 15, 0, 0, 0, 9, 102, 105, 110, 97, 108, 105, 122, 101, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 7, 118, 101, 114, 115, 105, 111, 110, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0}
	state := &wire.State{}
	err := state.UnmarshalBinary(x)
	require.NoError(t, err)
	fmt.Println(state)
	res, err := state.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, x, res)
}

func TestStateConversion(t *testing.T) {
	rng := polytest.Prng(t)
	perunFirstState := *ptest.NewRandomState(rng,
		ptest.WithNumParts(2),
		ptest.WithNumAssets(1),
		ptest.WithNumLocked(0),
		ptest.WithoutApp(),
		ptest.WithBalancesInRange(big.NewInt(0).Mul(big.NewInt(1), big.NewInt(100_000_000)), big.NewInt(0).Mul(big.NewInt(1), big.NewInt(100_000_000))),
	)

	stellarFirstState, err := wire.MakeState(perunFirstState)
	require.NoError(t, err)

	perunLastState, err := wire.ToState(stellarFirstState)
	require.NoError(t, err)

	validatePerunStates(t, perunFirstState, perunLastState)

	stellarLastState, err := wire.MakeState(perunLastState)
	require.NoError(t, err)

	checkStellarStateEquality(t, stellarFirstState, stellarLastState)
}

func validatePerunStates(t *testing.T, first, last channel.State) {
	checkAssetsEquality(t, first, last)
	checkNoLockedAmount(t, first)
	checkNoLockedAmount(t, last)
	checkPerunStateEquality(t, first, last)
}

func checkAssetsEquality(t *testing.T, first, last channel.State) {
	for i, asset := range first.Allocation.Assets {
		require.True(t, asset.Equal(last.Allocation.Assets[i]))
	}
}

func checkNoLockedAmount(t *testing.T, state channel.State) {
	if len(state.Allocation.Locked) != 0 {
		t.Fatal("locked amount should be empty")
	}
}

func checkPerunStateEquality(t *testing.T, first, last channel.State) {
	require.Equal(t, first.IsFinal, last.IsFinal)
	require.Equal(t, first.ID, last.ID)
	require.Equal(t, first.Version, last.Version)
}

func checkStellarStateEquality(t *testing.T, first, last wire.State) {
	require.Equal(t, first.Version, last.Version)
	require.Equal(t, first.ChannelID, last.ChannelID)
	require.Equal(t, first.Finalized, last.Finalized)
	require.Equal(t, first.Balances.BalA, last.Balances.BalA)
	require.Equal(t, first.Balances.BalB, last.Balances.BalB)
	require.Equal(t, first.Balances.Token, last.Balances.Token)
}
