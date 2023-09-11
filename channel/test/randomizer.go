package test

import (
	"github.com/stellar/go/xdr"
	"math/rand"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/test"
	"perun.network/perun-stellar-backend/channel/types"
)

type Randomizer struct{}

var _ test.Randomizer = (*Randomizer)(nil)

func (*Randomizer) NewRandomAsset(*rand.Rand) channel.Asset {
	return NewRandomStellarAsset()
}

func NewRandomStellarAsset() *types.StellarAsset {
	var contractID xdr.Hash
	rand.Read(contractID[:])
	return types.NewStellarAsset(contractID)
}
