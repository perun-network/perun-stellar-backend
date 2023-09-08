package test

import (
	"perun.network/go-perun/wallet/test"
	_ "perun.network/perun-stellar-backend/channel/test"
)

func init() {
	test.SetRandomizer(&Randomizer{})
}
