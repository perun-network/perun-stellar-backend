package channel

import (
	"crypto/sha256"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wire"
)

type backend struct{}

var Backend = backend{}

func init() {
	channel.SetBackend(Backend)
}

func (b backend) CalcID(params *channel.Params) channel.ID {
	wp := wire.MustMakeParams(*params)
	bytes, err := wp.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return sha256.Sum256(bytes)
}

func (b backend) Sign(account wallet.Account, state *channel.State) (wallet.Sig, error) {
	bytes, err := EncodeState(state)
	if err != nil {
		return nil, err
	}
	return account.SignData(bytes)
}

func (b backend) Verify(addr wallet.Address, state *channel.State, sig wallet.Sig) (bool, error) {
	bytes, err := EncodeState(state)
	if err != nil {
		return false, err
	}
	return wallet.VerifySignature(bytes, sig, addr)
}

func (b backend) NewAsset() channel.Asset {
	return &types.StellarAsset{}
}

func EncodeState(state *channel.State) ([]byte, error) {
	ws, err := wire.MakeState(*state)
	if err != nil {
		return nil, err
	}
	return ws.MarshalBinary()
}
