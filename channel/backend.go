// Copyright 2024 PolyCrypt GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package channel

import (
	"crypto/sha256"
	"errors"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/channel/types"
	wtypes "perun.network/perun-stellar-backend/wallet/types"
	"perun.network/perun-stellar-backend/wire"
)

const EthBackendID = 1

type backend struct{}

var Backend = backend{}

func init() {
	channel.SetBackend(Backend, wtypes.StellarBackendID)
}

func (b backend) CalcID(params *channel.Params) (channel.ID, error) {
	wp, _ := wire.MustMakeParams(*params)
	bytes, err := wp.MarshalBinary()
	if err != nil {
		return channel.ID{}, err
	}
	return sha256.Sum256(bytes), nil
}

func (b backend) Sign(account wallet.Account, state *channel.State) (wallet.Sig, error) {

	if err := checkBackends(state.Allocation.Backends); err != nil {
		return nil, errors.New("invalid backends in state allocation: " + err.Error())
	}

	ethState := ToEthState(state)

	bytes, err := EncodeEthState(&ethState)
	if err != nil {
		return nil, err
	}
	sig, err := account.SignData(bytes)
	if err != nil {
		return nil, err
	}
	return sig, err
}

func (b backend) Verify(addr wallet.Address, state *channel.State, sig wallet.Sig) (bool, error) {
	ethState := ToEthState(state)
	bytes, err := EncodeEthState(&ethState)

	if err != nil {
		return false, err
	}
	return wallet.VerifySignature(bytes, sig, addr)
}

func (b backend) NewAsset() channel.Asset {
	return &types.StellarAsset{}
}

func EncodeState(state *channel.State) ([]byte, error) {
	// check if state also has different backends stored in allocation

	if err := checkBackends(state.Allocation.Backends); err != nil {
		return nil, errors.New("invalid backends in state allocation: " + err.Error())
	}

	ws, err := wire.MakeState(*state)
	if err != nil {
		return nil, err
	}
	return ws.MarshalBinary()
}

func (b backend) NewAppID() (channel.AppID, error) {
	addr := &wtypes.Address{}
	return &AppID{addr}, nil
}

func checkBackends(backends []wallet.BackendID) error {
	if len(backends) == 0 {
		return errors.New("backends slice is empty")
	}

	hasStellarBackend := false

	for _, backend := range backends {
		if backend == wtypes.StellarBackendID {
			hasStellarBackend = true
		}
	}

	if !hasStellarBackend {
		return errors.New("StellarBackendID not found in backends")
	}

	return nil
}
