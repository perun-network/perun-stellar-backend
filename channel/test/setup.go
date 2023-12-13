package test

import (
	"math/big"
	"testing"

	//"perun.network/go-perun/backend/sim/wallet"
	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"
	//pwtest "perun.network/go-perun/wallet/test"

	ptest "perun.network/go-perun/channel/test"
	pkgtest "polycry.pt/poly-go/test"
)

func NewParamsState(t *testing.T) (*pchannel.Params, *pchannel.State) {

	rng := pkgtest.Prng(t)

	numParts := 2

	return ptest.NewRandomParamsAndState(rng, ptest.WithNumLocked(0).Append(
		ptest.WithVersion(0),
		ptest.WithNumParts(numParts),
		ptest.WithIsFinal(false),
		ptest.WithLedgerChannel(true),
		ptest.WithVirtualChannel(false),
		ptest.WithNumAssets(1),
		ptest.WithoutApp(),
		ptest.WithBalancesInRange(big.NewInt(0).Mul(big.NewInt(1), big.NewInt(100_000)), big.NewInt(0).Mul(big.NewInt(1), big.NewInt(100_000))),
	))
}

func NewParamsWithAddressState(t *testing.T, partsAddr []pwallet.Address) (*pchannel.Params, *pchannel.State) {

	rng := pkgtest.Prng(t)

	numParts := 2

	return ptest.NewRandomParamsAndState(rng, ptest.WithNumLocked(0).Append(
		ptest.WithVersion(0),
		ptest.WithNumParts(numParts),
		ptest.WithParts(partsAddr...),
		ptest.WithIsFinal(false),
		ptest.WithLedgerChannel(true),
		ptest.WithVirtualChannel(false),
		ptest.WithNumAssets(1),
		ptest.WithoutApp(),
		ptest.WithBalancesInRange(big.NewInt(0).Mul(big.NewInt(1), big.NewInt(100_000)), big.NewInt(0).Mul(big.NewInt(1), big.NewInt(100_000))),
	))
}
