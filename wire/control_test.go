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

package wire_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"perun.network/perun-stellar-backend/wire"
)

func TestControl(t *testing.T) {
	x := []byte{0, 0, 0, 17, 0, 0, 0, 1, 0, 0, 0, 7, 0, 0, 0, 15, 0, 0, 0, 6, 99, 108, 111, 115, 101, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 8, 100, 105, 115, 112, 117, 116, 101, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 8, 102, 117, 110, 100, 101, 100, 95, 97, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 8, 102, 117, 110, 100, 101, 100, 95, 98, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 9, 116, 105, 109, 101, 115, 116, 97, 109, 112, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 11, 119, 105, 116, 104, 100, 114, 97, 119, 110, 95, 97, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 11, 119, 105, 116, 104, 100, 114, 97, 119, 110, 95, 98, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	control := &wire.Control{}
	err := control.UnmarshalBinary(x)
	require.NoError(t, err)
	res, err := control.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, x, res)
}
