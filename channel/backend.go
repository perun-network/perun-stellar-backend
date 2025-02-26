// Copyright 2025 PolyCrypt GmbH
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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
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

// CalcID calculates the channel ID from the channel parameters.
func (b backend) CalcID(params *channel.Params) (channel.ID, error) {
	p, err := ToEthParams(params)
	if err != nil {
		return channel.ID{}, errors.WithMessage(err, "stellar could not convert params")
	}
	bytes, err := EncodeChannelParams(&p)
	if err != nil {
		return channel.ID{}, errors.WithMessage(err, "stellar could not encode params")
	}
	// Hash encoded params.
	return crypto.Keccak256Hash(bytes), nil
}

// Sign signs the channel state with the account.
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

// Verify verifies the signature of the channel state.
func (b backend) Verify(addr wallet.Address, state *channel.State, sig wallet.Sig) (bool, error) {
	ethState := ToEthState(state)
	bytes, err := EncodeEthState(&ethState)
	if err != nil {
		return false, err
	}
	return wallet.VerifySignature(bytes, sig, addr)
}

// NewAsset creates a new Stellar asset.
func (b backend) NewAsset() channel.Asset {
	return &types.StellarAsset{}
}

// EncodeState encodes the channel state.
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

// NewAppID creates a new Stellar app ID.
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
