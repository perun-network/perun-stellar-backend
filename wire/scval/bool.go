// Copyright 2023 PolyCrypt GmbH
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

package scval

import "github.com/stellar/go/xdr"

func WrapTrue() (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvBool, true)
}

func MustWrapTrue() xdr.ScVal {
	v, err := WrapTrue()
	if err != nil {
		panic(err)
	}
	return v
}

func WrapFalse() (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvBool, false)
}

func MustWrapFalse() xdr.ScVal {
	v, err := WrapFalse()
	if err != nil {
		panic(err)
	}
	return v
}

func WrapBool(b bool) (xdr.ScVal, error) {
	if b {
		return WrapTrue()
	} else {
		return WrapFalse()
	}
}

func MustWrapBool(b bool) xdr.ScVal {
	if b {
		return MustWrapTrue()
	} else {
		return MustWrapFalse()
	}
}
