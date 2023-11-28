package types_test

import (
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"
	"perun.network/perun-stellar-backend/channel/types"
	"testing"
)

func TestMarshalAndUnmarshalBinary(t *testing.T) {
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
