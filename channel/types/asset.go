package types

import (
	"errors"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/channel"
)

const AddressLen = 32

type StellarAsset struct {
	address keypair.FromAddress
}

func (s StellarAsset) MarshalBinary() (data []byte, err error) {
	return s.address.MarshalBinary()
}

func (s StellarAsset) UnmarshalBinary(data []byte) error {
	var addr [AddressLen]byte
	copy(addr[:], data)
	err := s.address.UnmarshalBinary(data)
	if err != nil {
		panic(err)
	}
	return nil
}

func (s StellarAsset) Equal(asset channel.Asset) bool {
	_, ok := asset.(*StellarAsset)
	return ok
}

func (s StellarAsset) MakeScAddress() (xdr.ScAddress, error) {
	scvAddr, err := MakeAccountAddress(&s.address)
	if err != nil {
		panic(err)
	}
	return scvAddr, nil
}

func (s *StellarAsset) FromScAddress(address xdr.ScAddress) error {
	addr, err := ToAccountAddress(address)
	if err != nil {
		panic(err)
	}
	s.address = addr
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
