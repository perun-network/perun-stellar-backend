package test

import (
	"perun.network/go-perun/wallet/test"
	_ "perun.network/perun-stellar-backend/channel/test"
	wtest "perun.network/perun-stellar-backend/wallet/test"
)

func init() {
	test.SetRandomizer(&wtest.Randomizer{})
}
