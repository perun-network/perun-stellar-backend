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
	"log"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/channel/types"
	wtypes "perun.network/perun-stellar-backend/wallet/types"
	"perun.network/perun-stellar-backend/wire"
)

type backend struct{}

var Backend = backend{}

func init() {
	channel.SetBackend(Backend)
}

func (b backend) CalcID(params *channel.Params) channel.ID {
	wp, err := wire.MustMakeParams(*params)
	if err != nil {
		log.Println("CalcID called with invalid params:", err)
		return channel.ID{}
	}
	bytes, err := wp.MarshalBinary()
	if err != nil {
		log.Println("CalcID called with invalid params:", err)
		return channel.ID{}
	}
	id := sha256.Sum256(bytes)
	log.Println("CalcID called:", id)
	return id
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

func (b backend) NewAppID() channel.AppID {
	addr := &wtypes.Address{}
	return &AppID{addr}
}
