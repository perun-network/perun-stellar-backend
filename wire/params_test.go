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
	"github.com/stretchr/testify/require"
	"math/big"
	"perun.network/go-perun/channel"
	ptest "perun.network/go-perun/channel/test"
	schannel "perun.network/perun-stellar-backend/channel"

	_ "perun.network/perun-stellar-backend/wallet/test"
	"perun.network/perun-stellar-backend/wire"

	pkgtest "polycry.pt/poly-go/test"
	"testing"
)

func TestParams(t *testing.T) {
	x := []byte{0, 0, 0, 17, 0, 0, 0, 1, 0, 0, 0, 4, 0, 0, 0, 15, 0, 0, 0, 1, 97, 0, 0, 0, 0, 0, 0, 17, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 15, 0, 0, 0, 4, 97, 100, 100, 114, 0, 0, 0, 18, 0, 0, 0, 1, 100, 102, 1, 62, 144, 25, 203, 254, 77, 51, 254, 154, 48, 112, 147, 73, 240, 64, 244, 179, 161, 243, 111, 26, 76, 81, 122, 190, 16, 11, 6, 86, 0, 0, 0, 15, 0, 0, 0, 6, 112, 117, 98, 107, 101, 121, 0, 0, 0, 0, 0, 13, 0, 0, 0, 32, 161, 254, 118, 204, 233, 36, 234, 44, 212, 63, 191, 30, 184, 43, 254, 124, 172, 122, 80, 159, 130, 160, 115, 195, 41, 12, 49, 89, 190, 53, 92, 50, 0, 0, 0, 15, 0, 0, 0, 1, 98, 0, 0, 0, 0, 0, 0, 17, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 15, 0, 0, 0, 4, 97, 100, 100, 114, 0, 0, 0, 18, 0, 0, 0, 1, 150, 139, 245, 155, 89, 98, 50, 60, 35, 101, 150, 241, 125, 46, 207, 112, 191, 2, 195, 252, 181, 164, 244, 5, 38, 183, 40, 224, 96, 159, 6, 162, 0, 0, 0, 15, 0, 0, 0, 6, 112, 117, 98, 107, 101, 121, 0, 0, 0, 0, 0, 13, 0, 0, 0, 32, 182, 104, 55, 79, 130, 121, 159, 228, 251, 195, 211, 137, 145, 92, 106, 55, 206, 137, 219, 47, 233, 99, 114, 77, 172, 109, 60, 143, 113, 160, 141, 192, 0, 0, 0, 15, 0, 0, 0, 18, 99, 104, 97, 108, 108, 101, 110, 103, 101, 95, 100, 117, 114, 97, 116, 105, 111, 110, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 10, 0, 0, 0, 15, 0, 0, 0, 5, 110, 111, 110, 99, 101, 0, 0, 0, 0, 0, 0, 13, 0, 0, 0, 32, 64, 194, 171, 150, 181, 52, 147, 0, 150, 145, 33, 44, 252, 12, 67, 66, 219, 135, 26, 235, 252, 58, 248, 99, 218, 239, 18, 73, 247, 124, 196, 67}
	p := &wire.Params{}
	err := p.UnmarshalBinary(x)
	require.NoError(t, err)
	res, err := p.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, x, res)
}

func TestParamsConversion(t *testing.T) {
	rng := pkgtest.Prng(t)

	numParts := 2

	perunFirstParams := *ptest.NewRandomParams(rng, ptest.WithNumLocked(0).Append(
		ptest.WithNumParts(numParts),
		ptest.WithBalancesInRange(big.NewInt(0), big.NewInt(1<<60)),
		ptest.WithLedgerChannel(true),
		ptest.WithVirtualChannel(false),
		ptest.WithNumAssets(1),
		ptest.WithoutApp(),
	))

	stellarFirstParams, err := wire.MakeParams(perunFirstParams)
	require.NoError(t, err)

	perunLastParams, err := wire.ToParams(stellarFirstParams)
	require.NoError(t, err)

	checkPerunParamsEquality(t, perunFirstParams, perunLastParams, numParts)

	stellarLastParams, err := wire.MakeParams(perunLastParams)
	require.NoError(t, err)

	checkStellarParamsEquality(t, stellarFirstParams, stellarLastParams)
}

func checkPerunParamsEquality(t *testing.T, first, last channel.Params, numParts int) {
	require.Equal(t, first.ID(), schannel.Backend.CalcID(&last))

	for i := 0; i < numParts; i++ {
		require.True(t, last.Parts[i].Equal(first.Parts[i]))
	}

	require.Equal(t, first.ChallengeDuration, last.ChallengeDuration)
	require.Equal(t, first.Nonce, last.Nonce)
	require.Equal(t, first.App, last.App)
	require.Equal(t, first.LedgerChannel, last.LedgerChannel)
	require.Equal(t, first.VirtualChannel, last.VirtualChannel)
}

func checkStellarParamsEquality(t *testing.T, first, last wire.Params) {
	require.Equal(t, first.A, last.A)
	require.Equal(t, first.B, last.B)
	require.Equal(t, first.ChallengeDuration, last.ChallengeDuration)
	require.Equal(t, first.Nonce, last.Nonce)
}
