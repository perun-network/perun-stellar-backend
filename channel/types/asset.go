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

package types

import (
	"errors"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/channel"
)

const HashLenXdr = 32

type StellarAsset struct {
	contractID xdr.Hash
}

func (s StellarAsset) ContractID() xdr.Hash {
	return s.contractID
}

func NewStellarAsset(contractID xdr.Hash) *StellarAsset {
	return &StellarAsset{contractID: contractID}
}

func (s StellarAsset) MarshalBinary() (data []byte, err error) {
	return s.contractID.MarshalBinary()
}

func (s *StellarAsset) UnmarshalBinary(data []byte) error {
	var addr [HashLenXdr]byte
	copy(addr[:], data)
	err := s.contractID.UnmarshalBinary(data)
	if err != nil {
		return errors.New("could not unmarshal contract id")
	}
	return nil
}

func (s StellarAsset) Equal(asset channel.Asset) bool {
	_, ok := asset.(*StellarAsset)
	return ok
}

func (s StellarAsset) MakeScAddress() (xdr.ScAddress, error) {
	hash := s.contractID
	scvAddr, err := MakeContractAddress(hash)
	if err != nil {
		return xdr.ScAddress{}, errors.New("could not generate contract address")
	}
	return scvAddr, nil
}

func (s *StellarAsset) FromScAddress(address xdr.ScAddress) error {
	addrType := address.Type
	if addrType != xdr.ScAddressTypeScAddressTypeContract {
		return errors.New("invalid address type")
	}

	s.contractID = *address.ContractId
	return nil
}

func NewStellarAssetFromScAddress(address xdr.ScAddress) (*StellarAsset, error) {
	s := &StellarAsset{}
	err := s.FromScAddress(address)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func MustStellarAsset(asset channel.Asset) *StellarAsset {
	p, ok := asset.(*StellarAsset)
	if !ok {
		panic("Asset has invalid type")
	}
	return p
}

func ToStellarAsset(asset channel.Asset) (*StellarAsset, error) {
	p, ok := asset.(*StellarAsset)
	if !ok {
		return nil, errors.New("asset has invalid type")
	}
	return p, nil
}

func MakeAccountAddress(kp keypair.KP) (xdr.ScAddress, error) {
	accountId, err := xdr.AddressToAccountId(kp.Address())
	if err != nil {
		return xdr.ScAddress{}, err
	}
	return xdr.NewScAddress(xdr.ScAddressTypeScAddressTypeAccount, accountId)
}

func MakeContractAddress(contractID xdr.Hash) (xdr.ScAddress, error) {
	return xdr.NewScAddress(xdr.ScAddressTypeScAddressTypeContract, contractID)
}

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
