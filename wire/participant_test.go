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
	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wire"
	pkgtest "polycry.pt/poly-go/test"
	"testing"
)

func TestParticipant(t *testing.T) {
	// Participant XDR generated by soroban contract
	x := []byte{0, 0, 0, 17, 0, 0, 0, 1, 0, 0, 0, 3, 0, 0, 0, 15, 0, 0, 0, 7, 99, 99, 95, 97, 100, 100, 114, 0, 0, 0, 0, 13, 0, 0, 0, 20, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 0, 0, 0, 15, 0, 0, 0, 12, 115, 116, 101, 108, 108, 97, 114, 95, 97, 100, 100, 114, 0, 0, 0, 18, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 15, 0, 0, 0, 14, 115, 116, 101, 108, 108, 97, 114, 95, 112, 117, 98, 107, 101, 121, 0, 0, 0, 0, 0, 16, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 15, 0, 0, 0, 5, 67, 114, 111, 115, 115, 0, 0, 0, 0, 0, 0, 13, 0, 0, 0, 65, 4, 68, 167, 147, 222, 2, 138, 40, 253, 81, 242, 178, 183, 89, 162, 56, 44, 87, 96, 8, 140, 200, 120, 202, 31, 111, 39, 232, 91, 78, 124, 190, 31, 66, 113, 44, 231, 72, 50, 102, 242, 214, 164, 158, 151, 39, 14, 118, 239, 215, 222, 73, 156, 215, 173, 211, 108, 63, 11, 140, 171, 161, 154, 65, 138, 0, 0, 0}
	p := &wire.Participant{}
	err := p.UnmarshalBinary(x)
	require.NoError(t, err)
	res, err := p.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, x, res)
}

func TestParticipantConversion(t *testing.T) {
	rng := pkgtest.Prng(t)
	acc, _, err := wallet.NewRandomAccount(rng)
	require.NoError(t, err)
	p := *acc.Participant()
	wp, err := wire.MakeParticipant(p)
	require.NoError(t, err)
	res, err := wire.ToParticipant(wp)
	require.NoError(t, err)
	require.True(t, p.Equal(&res))
}
