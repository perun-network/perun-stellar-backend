package types

import (
	"errors"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/channel"
)

type StellarAsset struct {
}

func (s StellarAsset) MarshalBinary() (data []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (s StellarAsset) UnmarshalBinary(data []byte) error {
	//TODO implement me
	panic("implement me")
}

func (s StellarAsset) Equal(asset channel.Asset) bool {
	//TODO implement me
	panic("implement me")
}

func (s StellarAsset) MakeScAddress() (xdr.ScAddress, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StellarAsset) FromScAddress(xdr.ScAddress) error {
	//TODO implement me
	panic("implement me")
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
