package wallet_test

import (
	"github.com/stretchr/testify/require"
	"math/rand"
	gptest "perun.network/go-perun/wallet/test"
	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wallet/types"
	pkgtest "polycry.pt/poly-go/test"
	"testing"
)

func TestEphemeralWallet(t *testing.T) {
	rng := pkgtest.Prng(t)
	w := wallet.NewEphemeralWallet()

	acc, _, err := w.AddNewAccount(rng)
	require.NoError(t, err)

	unlockedAccount, err := w.Unlock(acc.Address())
	require.NoError(t, err)
	require.Equal(t, acc.Address(), unlockedAccount.Address())

	msg := []byte("hello world")
	sig, err := unlockedAccount.SignData(msg)
	require.NoError(t, err)

	valid, err := wallet.Backend.VerifySignature(msg, sig, acc.Address())
	require.NoError(t, err)
	require.True(t, valid)
}

func setup(rng *rand.Rand) *gptest.Setup {
	w := wallet.NewEphemeralWallet()
	acc, _, err := w.AddNewAccount(rng)
	if err != nil {
		panic(err)
	}
	acc2, _, err := w.AddNewAccount(rng)
	if err != nil {
		panic(err)
	}
	binAddr2, err := acc2.Address().MarshalBinary()
	if err != nil {
		panic(err)
	}
	z, err := types.ZeroAddress()
	if err != nil {
		panic(err)
	}
	return &gptest.Setup{
		Backend:           wallet.Backend,
		Wallet:            w,
		AddressInWallet:   acc.Address(),
		ZeroAddress:       z,
		DataToSign:        []byte("pls sign me"),
		AddressMarshalled: binAddr2,
	}
}

func TestAddress(t *testing.T) {
	rng := pkgtest.Prng(t)
	gptest.TestAddress(t, setup(rng))
}

func TestGenericSignatureSizeTest(t *testing.T) {
	rng := pkgtest.Prng(t)
	gptest.GenericSignatureSizeTest(t, setup(rng))
}

func TestAccountWithWalletAndBackend(t *testing.T) {
	rng := pkgtest.Prng(t)
	gptest.TestAccountWithWalletAndBackend(t, setup(rng))
}
