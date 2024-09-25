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

package wire_test

import (
	"github.com/stretchr/testify/require"
	"math/big"
	"perun.network/go-perun/channel"
	ptest "perun.network/go-perun/channel/test"
	schannel "perun.network/perun-stellar-backend/channel"
	_ "perun.network/perun-stellar-backend/channel/test"
	"perun.network/perun-stellar-backend/wallet/types"
	"perun.network/perun-stellar-backend/wire"

	pkgtest "polycry.pt/poly-go/test"
	"testing"
)

func TestParams(t *testing.T) {
	x := []byte{0, 0, 0, 17, 0, 0, 0, 1, 0, 0, 0, 4, 0, 0, 0, 15, 0, 0, 0, 1, 97, 0, 0, 0, 0, 0, 0, 17, 0, 0, 0, 1, 0, 0, 0, 3, 0, 0, 0, 15, 0, 0, 0, 7, 99, 99, 95, 97, 100, 100, 114, 0, 0, 0, 0, 13, 0, 0, 0, 20, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 0, 0, 0, 15, 0, 0, 0, 12, 115, 116, 101, 108, 108, 97, 114, 95, 97, 100, 100, 114, 0, 0, 0, 18, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 15, 0, 0, 0, 14, 115, 116, 101, 108, 108, 97, 114, 95, 112, 117, 98, 107, 101, 121, 0, 0, 0, 0, 0, 16, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 15, 0, 0, 0, 5, 67, 114, 111, 115, 115, 0, 0, 0, 0, 0, 0, 13, 0, 0, 0, 65, 4, 217, 187, 222, 165, 172, 170, 22, 148, 174, 98, 198, 84, 102, 211, 114, 32, 234, 203, 232, 210, 213, 251, 19, 71, 140, 181, 54, 43, 162, 178, 227, 123, 249, 190, 187, 175, 112, 12, 29, 64, 117, 253, 35, 113, 189, 88, 110, 11, 191, 239, 116, 201, 57, 185, 152, 156, 110, 88, 73, 122, 195, 170, 96, 57, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 1, 98, 0, 0, 0, 0, 0, 0, 17, 0, 0, 0, 1, 0, 0, 0, 3, 0, 0, 0, 15, 0, 0, 0, 7, 99, 99, 95, 97, 100, 100, 114, 0, 0, 0, 0, 13, 0, 0, 0, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 0, 0, 0, 15, 0, 0, 0, 12, 115, 116, 101, 108, 108, 97, 114, 95, 97, 100, 100, 114, 0, 0, 0, 18, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 15, 0, 0, 0, 14, 115, 116, 101, 108, 108, 97, 114, 95, 112, 117, 98, 107, 101, 121, 0, 0, 0, 0, 0, 16, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 15, 0, 0, 0, 5, 67, 114, 111, 115, 115, 0, 0, 0, 0, 0, 0, 13, 0, 0, 0, 65, 4, 74, 48, 63, 166, 12, 25, 126, 14, 205, 31, 30, 194, 187, 132, 103, 73, 247, 159, 221, 230, 206, 253, 252, 137, 115, 32, 32, 6, 138, 91, 47, 90, 191, 123, 252, 160, 24, 141, 7, 181, 217, 98, 211, 15, 127, 23, 16, 183, 186, 102, 236, 94, 225, 232, 207, 164, 116, 10, 129, 204, 118, 218, 64, 47, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 18, 99, 104, 97, 108, 108, 101, 110, 103, 101, 95, 100, 117, 114, 97, 116, 105, 111, 110, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 10, 0, 0, 0, 15, 0, 0, 0, 5, 110, 111, 110, 99, 101, 0, 0, 0, 0, 0, 0, 13, 0, 0, 0, 32, 135, 219, 214, 115, 126, 153, 86, 98, 50, 30, 42, 239, 94, 201, 68, 155, 149, 175, 245, 120, 242, 135, 194, 66, 207, 135, 20, 220, 122, 252, 206, 151}
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
		ptest.WithBackend(2),
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
	lastChanID, err := schannel.Backend.CalcID(&last)
	require.NoError(t, err)
	require.Equal(t, first.ID()[types.StellarBackendID], lastChanID)

	for i := 0; i < numParts; i++ {
		for backendID := range last.Parts[i] {
			require.True(t, last.Parts[i][backendID].Equal(first.Parts[i][backendID]))
		}
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
