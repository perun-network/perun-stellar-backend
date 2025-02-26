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

package scval

import "github.com/stellar/go/xdr"

// WrapTrue wraps a true into an xdr.ScVal.
func WrapTrue() (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvBool, true)
}

// MustWrapTrue wraps a true into a xdr.ScVal.
func MustWrapTrue() (xdr.ScVal, error) {
	v, err := WrapTrue()
	if err != nil {
		return xdr.ScVal{}, err
	}
	return v, nil
}

// WrapFalse wraps a false into an xdr.ScVal.
func WrapFalse() (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvBool, false)
}

// MustWrapFalse wraps a false into a xdr.ScVal.
func MustWrapFalse() (xdr.ScVal, error) {
	v, err := WrapFalse()
	if err != nil {
		return xdr.ScVal{}, err
	}
	return v, nil
}

// WrapBool wraps a bool into an xdr.ScVal.
func WrapBool(b bool) (xdr.ScVal, error) {
	if b {
		return WrapTrue()
	}
	return WrapFalse()
}

// MustWrapBool wraps a bool into a xdr.ScVal.
func MustWrapBool(b bool) (xdr.ScVal, error) {
	if b {
		return MustWrapTrue()
	}
	return MustWrapFalse()
}
