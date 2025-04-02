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

package types

import (
	"encoding/hex"
	"errors"
	"log"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/multi"

	"perun.network/perun-stellar-backend/wallet/types"
)

const (
	HashLenXdr        = 32
	StellarContractID = "2"
)

var _ multi.Asset = (*StellarAsset)(nil)

type (
	// Asset represents a generic asset.
	Asset struct {
		contractID xdr.Hash
	}
	// StellarAsset represents a Stellar asset.
	StellarAsset struct {
		Asset Asset
		id    CCID
	}
	// CCID is a unique identifier for a channel asset.
	CCID struct {
		backendID uint32
		ledgerID  ContractLID
	}

	// ContractLID is a unique identifier for a contract.
	ContractLID struct{ string }
)

// BackendID returns the backend ID of the asset.
func (c CCID) BackendID() uint32 {
	return c.backendID
}

// LedgerID returns the ledger ID of the asset.
func (c CCID) LedgerID() multi.LedgerID {
	return c.ledgerID
}

// MakeContractID makes a ChainID for the given id.
func MakeContractID(id string) ContractLID {
	return ContractLID{id}
}

// MakeCCID makes a CCID for the given id.
func MakeCCID(ledgerID ContractLID) CCID {
	return CCID{types.StellarBackendID, ledgerID}
}

// UnmarshalBinary unmarshals the contractID from its binary representation.
func (id *ContractLID) UnmarshalBinary(data []byte) error {
	str := hex.EncodeToString(data) // Convert binary data to hex string
	id.string = str
	return nil
}

// MarshalBinary marshals the contractID into its binary representation.
func (id ContractLID) MarshalBinary() ([]byte, error) {
	if id.string == "" {
		return nil, errors.New("nil ContractID")
	}
	return hex.DecodeString(id.string)
}

// MapKey returns the asset's map key representation.
func (id ContractLID) MapKey() multi.LedgerIDMapKey {
	if id.string == "" {
		return ""
	}
	return multi.LedgerIDMapKey(id.string)
}

// ContractID returns the contract ID of the asset.
func (a Asset) ContractID() xdr.Hash {
	return a.contractID
}

// NewStellarAsset creates a new Stellar asset with the given contract ID.
func NewStellarAsset(contractID xdr.Hash) *StellarAsset {
	return &StellarAsset{Asset: Asset{contractID}, id: MakeCCID(MakeContractID(StellarContractID))}
}

// MarshalBinary marshals the Stellar asset into its binary representation.
func (s StellarAsset) MarshalBinary() (data []byte, err error) {
	return s.Asset.MarshalBinary()
}

// UnmarshalBinary unmarshals the Stellar asset from its binary representation.
func (s *StellarAsset) UnmarshalBinary(data []byte) error {
	var addr [HashLenXdr]byte
	copy(addr[:], data)
	err := s.Asset.UnmarshalBinary(data)
	if err != nil {
		return errors.New("could not unmarshal contract id")
	}
	s.id = MakeCCID(MakeContractID(StellarContractID))
	return nil
}

// MarshalBinary marshals the asset into its binary representation.
func (a Asset) MarshalBinary() (data []byte, err error) {
	return a.contractID.MarshalBinary()
}

// UnmarshalBinary unmarshals the asset from its binary representation.
func (a *Asset) UnmarshalBinary(data []byte) error {
	var addr [HashLenXdr]byte
	copy(addr[:], data)
	err := a.contractID.UnmarshalBinary(data)
	if err != nil {
		return errors.New("could not unmarshal contract id")
	}
	return nil
}

// Equal checks if the given asset is equal to the Stellar asset.
func (s StellarAsset) Equal(asset channel.Asset) bool {
	_, ok := asset.(*StellarAsset)
	return ok
}

// Equal checks if the given asset is equal to the asset.
func (a Asset) Equal(asset channel.Asset) bool {
	_, ok := asset.(*Asset)
	return ok
}

// Address returns the address of the Stellar asset.
func (s StellarAsset) Address() []byte {
	return s.Asset.Address()
}

// Address returns the address of the asset.
func (a Asset) Address() []byte {
	return a.contractID[:]
}

// LedgerBackendID returns the asset ID of the Stellar asset.
func (s StellarAsset) LedgerBackendID() multi.LedgerBackendID {
	return s.id
}

// MapKey returns the asset's map key representation.
func (s StellarAsset) MapKey() AssetMapKey {
	d, err := s.MarshalBinary()
	if err != nil {
		log.Fatalf("could not marshal asset: %v", err)
		return ""
	}

	return AssetMapKey(d)
}

// LedgerID returns the ledger ID the asset lives on.
func (s StellarAsset) LedgerID() multi.LedgerID {
	return s.id.LedgerID()
}

// MakeScAddress generates a ScAddress from the Stellar asset.
func (s StellarAsset) MakeScAddress() (xdr.ScAddress, error) {
	hash := s.Asset.contractID
	scvAddr, err := MakeContractAddress(hash)
	if err != nil {
		return xdr.ScAddress{}, errors.New("could not generate contract address")
	}
	return scvAddr, nil
}

// FromScAddress generates a Stellar asset from the given ScAddress.
func (s *StellarAsset) FromScAddress(address xdr.ScAddress) error {
	if addrType := address.Type; addrType != xdr.ScAddressTypeScAddressTypeContract {
		return errors.New("invalid address type")
	}

	s.Asset.contractID = *address.ContractId
	s.id = MakeCCID(MakeContractID(StellarContractID))
	return nil
}

// NewStellarAssetFromScAddress creates a new Stellar asset from the given ScAddress.
func NewStellarAssetFromScAddress(address xdr.ScAddress) (*StellarAsset, error) {
	s := &StellarAsset{}
	err := s.FromScAddress(address)
	if err != nil {
		return nil, err
	}
	s.id = MakeCCID(MakeContractID(StellarContractID))
	return s, nil
}

// MustStellarAsset panics if the given asset is not a Stellar asset.
func MustStellarAsset(asset channel.Asset) *StellarAsset {
	p, ok := asset.(*StellarAsset)
	if !ok {
		panic("Asset has invalid type")
	}
	return p
}

// ToStellarAsset converts the given asset to a Stellar asset.
func ToStellarAsset(asset channel.Asset) (*StellarAsset, error) {
	p, ok := asset.(*StellarAsset)
	if !ok {
		return nil, errors.New("asset has invalid type")
	}
	return p, nil
}

// MakeAccountAddress generates an account address from the given keypair.
func MakeAccountAddress(kp keypair.KP) (xdr.ScAddress, error) {
	accountID, err := xdr.AddressToAccountId(kp.Address())
	if err != nil {
		return xdr.ScAddress{}, err
	}
	return xdr.NewScAddress(xdr.ScAddressTypeScAddressTypeAccount, accountID)
}

// AccountAddressFromAddress generates an account address from the given address.
func AccountAddressFromAddress(addr keypair.FromAddress) (xdr.ScAddress, error) {
	accountID, err := xdr.AddressToAccountId(addr.Address())
	if err != nil {
		return xdr.ScAddress{}, err
	}
	return xdr.NewScAddress(xdr.ScAddressTypeScAddressTypeAccount, accountID)
}

// MakeContractAddress generates a contract address from the given contract ID.
func MakeContractAddress(contractID xdr.Hash) (xdr.ScAddress, error) {
	return xdr.NewScAddress(xdr.ScAddressTypeScAddressTypeContract, contractID)
}

// ToAccountAddress converts the given ScAddress to an account address.
func ToAccountAddress(address xdr.ScAddress) (keypair.FromAddress, error) {
	if address.Type != xdr.ScAddressTypeScAddressTypeAccount {
		return keypair.FromAddress{}, errors.New("invalid address type")
	}
	kp, err := keypair.ParseAddress(address.AccountId.Address())
	if err != nil {
		return keypair.FromAddress{}, err
	}
	return *kp, nil
}
