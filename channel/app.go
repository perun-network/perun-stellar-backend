// Copyright 2024 - See NOTICE file for copyright holders.
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
	"math/rand"

	"perun.network/go-perun/channel"
	"perun.network/go-perun/wallet"

	stwallet "perun.network/perun-stellar-backend/wallet"
	types "perun.network/perun-stellar-backend/wallet/types"
)

var _ channel.AppID = new(AppID)

type AppID struct {
	wallet.Address
}

type AppIDKey string

func (a AppID) Equal(b channel.AppID) bool {
	bTyped, ok := b.(*AppID)
	if !ok {
		return false
	}
	return a.Address.Equal(bTyped.Address)
}

func (a AppID) Key() channel.AppIDKey {
	b, err := a.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return channel.AppIDKey(b)
}

func (a AppID) MarshalBinary() ([]byte, error) {
	data, err := a.Address.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (a *AppID) UnmarshalBinary(data []byte) error {
	addr := &types.Address{}
	err := addr.UnmarshalBinary(data)
	if err != nil {
		return err
	}
	appaddr := &AppID{addr}
	*a = *appaddr
	return nil
}

func NewRandomAppID(rng *rand.Rand) *AppID {
	addr := stwallet.NewRandomAddress(rng)
	return &AppID{addr}
}
