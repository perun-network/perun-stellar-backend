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

package types_test

import (
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"
	"perun.network/perun-stellar-backend/channel/types"
	"testing"
)

func TestAssetMarshalAndUnmarshalBinary(t *testing.T) {
	var hash xdr.Hash
	copy(hash[:], []byte("testhashfortestingonly!testhash"))

	asset := types.NewStellarAsset(hash)

	data, err := asset.MarshalBinary()
	require.NoError(t, err)

	newAsset := &types.StellarAsset{}
	err = newAsset.UnmarshalBinary(data)
	require.NoError(t, err)

	require.Equal(t, asset.ContractID().HexString(), newAsset.ContractID().HexString(), "Mismatched ContractID. Expected %x, got %x", asset.ContractID(), newAsset.ContractID())
}

func TestMakeAccountAddress(t *testing.T) {
	kp, _ := keypair.Random()

	address, err := types.MakeAccountAddress(kp)
	require.NoError(t, err)

	require.Equal(t, xdr.ScAddressTypeScAddressTypeAccount, address.Type, "Expected account address type, got %v", address.Type)
}
