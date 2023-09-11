package test

import (
	"math/rand"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/wallet/test"
	swallet "perun.network/perun-stellar-backend/wallet"
)

type Randomizer struct {
	Account *swallet.Account
}

// NewRandomAddress implements test.Randomizer
func (r *Randomizer) NewRandomAddress(rng *rand.Rand) wallet.Address {
	return swallet.NewRandomAddress(rng)
}

// NewWallet implements test.Randomizer
func (*Randomizer) NewWallet() test.Wallet {
	panic("unimplemented")
}

// RandomWallet implements test.Randomizer
func (*Randomizer) RandomWallet() test.Wallet {
	panic("unimplemented")
}

var _ test.Randomizer = (*Randomizer)(nil)
