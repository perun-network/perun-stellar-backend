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

// Address implements the wallet.Address interface for the Stellar backend.
type Address keypair.FromAddress

var _ wallet.Address = (*Address)(nil)

// Equal compares two addresses for equality.
func (a *Address) Equal(addr wallet.Address) bool {
	other, ok := addr.(*Address)
	if !ok {
		return false
	}
	return (*keypair.FromAddress)(a).Equal((*keypair.FromAddress)(other))
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (a *Address) MarshalBinary() ([]byte, error) {
	return (*keypair.FromAddress)(a).MarshalBinary()
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (a *Address) UnmarshalBinary(data []byte) error {
	return (*keypair.FromAddress)(a).UnmarshalBinary(data)
}

// String returns the string representation of the address.
func (a *Address) String() string {
	return (*keypair.FromAddress)(a).Address()
}

// BackendID returns the Stellar backend ID.
func (a *Address) BackendID() wallet.BackendID {
	return StellarBackendID
}
