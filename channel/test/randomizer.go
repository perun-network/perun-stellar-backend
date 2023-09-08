package test

import (
	"math/rand"

	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/test"
	"perun.network/perun-stellar-backend/channel/types"
)

type Randomizer struct{}

func (*Randomizer) NewRandomAsset(*rand.Rand) channel.Asset {
	return &types.StellarAsset{}
}

var _ test.Randomizer = (*Randomizer)(nil)
