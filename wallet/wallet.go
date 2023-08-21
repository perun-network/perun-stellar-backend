package wallet

import (
	"errors"
	"github.com/stellar/go/keypair"
	"math/rand"
	"perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/wallet/types"
	"polycry.pt/poly-go/sync"
)

type EphemeralWallet struct {
	lock     sync.Mutex
	accounts map[string]*Account
}

func (e *EphemeralWallet) Unlock(a wallet.Address) (wallet.Account, error) {
	addr, ok := a.(*types.Participant)
	if !ok {
		return nil, errors.New("incorrect Participant type")
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	account, ok := e.accounts[addr.String()]
	if !ok {
		return nil, errors.New("account not found")
	}
	return account, nil
}

func (e *EphemeralWallet) LockAll() {}

func (e *EphemeralWallet) IncrementUsage(address wallet.Address) {}

func (e *EphemeralWallet) DecrementUsage(address wallet.Address) {}

func (e *EphemeralWallet) AddNewAccount(rng *rand.Rand) (*Account, *keypair.Full, error) {
	acc, kp, err := NewRandomAccount(rng)
	if err != nil {
		return nil, nil, err
	}
	return acc, kp, e.AddAccount(acc)
}

func (e *EphemeralWallet) AddAccount(acc *Account) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	k := types.AsParticipant(acc.Address()).String()
	_, ok := e.accounts[k]
	if ok {
		return errors.New("account already exists")
	}
	e.accounts[k] = acc
	return nil
}

func NewEphemeralWallet() *EphemeralWallet {
	return &EphemeralWallet{
		accounts: make(map[string]*Account),
	}
}
