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

package test

import (
	"crypto/rand"
	"github.com/stellar/go/xdr"
	mrand "math/rand"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/test"
	"perun.network/perun-stellar-backend/channel/types"
)

type Randomizer struct{}

var _ test.Randomizer = (*Randomizer)(nil)

func (*Randomizer) NewRandomAsset(*mrand.Rand) channel.Asset {
	return NewRandomStellarAsset()
}

func NewRandomStellarAsset() *types.StellarAsset {
	var contractID xdr.Hash
	if _, err := rand.Read(contractID[:]); err != nil {
		return nil
	}
	return types.NewStellarAsset(contractID)
}
