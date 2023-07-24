package wallet

import (
	"crypto/ed25519"
	"errors"
	"github.com/stellar/go/keypair"
	"math/rand"
	"perun.network/go-perun/wallet"
)

// Account is used for signing channel state.
type Account struct {
	// privateKey is the private key of the account.
	privateKey ed25519.PrivateKey
	// ParticipantAddress references the ParticipantAddress of the Participant this account belongs to.
	ParticipantAddress keypair.FromAddress
}

// NewRandomAccountWithAddress creates a new account with a random private key and the given address as
// Account.ParticipantAddress.
func NewRandomAccountWithAddress(rng *rand.Rand, addr *keypair.FromAddress) (*Account, error) {
	_, s, err := ed25519.GenerateKey(rng)
	if err != nil {
		return nil, err
	}
	return &Account{privateKey: s, ParticipantAddress: *addr}, nil
}

// NewRandomAccount creates a new account with a random private key. It also creates a random key pair, using its
// address as the account'privateKey Account.ParticipantAddress.
func NewRandomAccount(rng *rand.Rand) (*Account, *keypair.Full, error) {
	kp, err := keypair.Random()
	if err != nil {
		return nil, nil, err
	}
	acc, err := NewRandomAccountWithAddress(rng, kp.FromAddress())
	if err != nil {
		return nil, nil, err
	}
	return acc, kp, nil
}

// Address returns the Participant this account belongs to.
func (a Account) Address() wallet.Address {
	return NewParticipant(a.ParticipantAddress, a.privateKey.Public().(ed25519.PublicKey))
}

// SignData signs the given data with the account's private key.
func (a Account) SignData(data []byte) ([]byte, error) {
	if len(a.privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key size")
	}
	return ed25519.Sign(a.privateKey, data), nil
}
