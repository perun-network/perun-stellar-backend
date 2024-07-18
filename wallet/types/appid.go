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

package types

import (
	"github.com/stellar/go/keypair"
	"perun.network/go-perun/wallet"
)

type Address keypair.FromAddress

var _ wallet.Address = (*Address)(nil)

func (a *Address) Equal(addr wallet.Address) bool {
	other, ok := addr.(*Address)
	if !ok {
		return false
	}
	return (*keypair.FromAddress)(a).Equal((*keypair.FromAddress)(other))
}

func (a *Address) MarshalBinary() ([]byte, error) {
	return (*keypair.FromAddress)(a).MarshalBinary()
}

func (a *Address) UnmarshalBinary(data []byte) error {
	return (*keypair.FromAddress)(a).UnmarshalBinary(data)
}

func (a *Address) String() string {
	return (*keypair.FromAddress)(a).Address()
}
