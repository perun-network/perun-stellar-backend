package types_test

import (
	"bytes"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"perun.network/perun-stellar-backend/channel/types"
	"testing"
)

func TestMarshalAndUnmarshalBinary(t *testing.T) {
	var hash xdr.Hash
	copy(hash[:], []byte("testhashfortestingonly!testhash"))

	asset := types.NewStellarAsset(hash)

	data, err := asset.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	newAsset := &types.StellarAsset{}
	err = newAsset.UnmarshalBinary(data)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal([]byte(newAsset.ContractID().HexString()), []byte(asset.ContractID().HexString())) {
		t.Fatalf("expected %x, got %x", asset.ContractID(), newAsset.ContractID())
	}
}

func TestMakeAccountAddress(t *testing.T) {
	kp, _ := keypair.Random()

	address, err := types.MakeAccountAddress(kp)
	if err != nil {
		t.Fatal(err)
	}

	if address.Type != xdr.ScAddressTypeScAddressTypeAccount {
		t.Fatalf("expected account address type, got %v", address.Type)
	}
}
