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

// WrapScAddress wraps a ScAddress into a xdr.ScVal.
func WrapScAddress(address xdr.ScAddress) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvAddress, address)
}

// WrapScAddresses wraps a slice of ScAddress into a xdr.ScVal.
func WrapScAddresses(address []xdr.ScAddress) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvAddress, address)
}

// MustWrapScAddress wraps a ScAddress into a xdr.ScVal.
func MustWrapScAddress(address xdr.ScAddress) (xdr.ScVal, error) {
	v, err := WrapScAddress(address)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return v, nil
}

// MakeScVecFromScAddresses wraps scAddresses into a xdr.ScVec.
func MakeScVecFromScAddresses(addresses []xdr.ScAddress) xdr.ScVec {
	var xdrAddresses xdr.ScVec

	for _, val := range addresses {
		xdrAddrVal, err := WrapScAddress(val)
		if err != nil {
			panic("could not wrap address")
		}
		xdrAddresses = append(xdrAddresses, xdrAddrVal)
	}

	return xdrAddresses
}

// WrapInt128Parts wraps a Int128Parts into a xdr.ScVal.
func WrapInt128Parts(parts xdr.Int128Parts) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvI128, parts)
}

// WrapVec wraps a ScVec into a xdr.ScVal.
func WrapVec(scVec xdr.ScVec) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvVec, &scVec)
}

// MustWrapInt128Parts wraps a Int128Parts into a xdr.ScVal.
func MustWrapInt128Parts(parts xdr.Int128Parts) xdr.ScVal {
	v, err := WrapInt128Parts(parts)
	if err != nil {
		panic(err)
	}
	return v
}

// WrapScMap wraps a ScMap into a xdr.ScVal.
func WrapScMap(m xdr.ScMap) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvMap, &m)
}

// MustWrapScMap wraps a ScMap into a xdr.ScVal.
func MustWrapScMap(m xdr.ScMap) xdr.ScVal {
	v, err := WrapScMap(m)
	if err != nil {
		panic(err)
	}
	return v
}

// WrapScSymbol wraps a symbol into a xdr.ScVal.
func WrapScSymbol(symbol xdr.ScSymbol) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvSymbol, symbol)
}

// MustWrapScSymbol wraps a symbol into a xdr.ScVal.
func MustWrapScSymbol(symbol xdr.ScSymbol) xdr.ScVal {
	v, err := WrapScSymbol(symbol)
	if err != nil {
		panic(err)
	}
	return v
}

// WrapScString wraps a string into a xdr.ScVal.
func WrapScString(str xdr.ScString) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvString, str)
}

// WrapScUint64 wraps a Uint64 into a xdr.ScVal.
func WrapScUint64(ui xdr.Uint64) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvU64, ui)
}

// MustWrapScString wraps a string into a xdr.ScVal.
func MustWrapScString(str xdr.ScString) (xdr.ScVal, error) {
	v, err := WrapScString(str)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return v, nil
}

// MustWrapScUint64 wraps a Uint64 into a xdr.ScVal.
func MustWrapScUint64(ui xdr.Uint64) (xdr.ScVal, error) {
	v, err := WrapScUint64(ui)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return v, nil
}

// WrapScBytes wraps a ScBytes into a xdr.ScVal.
func WrapScBytes(b xdr.ScBytes) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvBytes, b)
}

// WrapScVec wraps a ScVec into a xdr.ScVal.
func WrapScVec(v xdr.ScVec) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvVec, &v)
}

// MustWrapScBytes wraps a ScBytes into a xdr.ScVal.
func MustWrapScBytes(b xdr.ScBytes) (xdr.ScVal, error) {
	v, err := WrapScBytes(b)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return v, nil
}

// WrapUint64 wraps a Uint64 into a xdr.ScVal.
func WrapUint64(i xdr.Uint64) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvU64, i)
}

// MustWrapUint64 wraps a Uint64 into a xdr.ScVal.
func MustWrapUint64(i xdr.Uint64) xdr.ScVal {
	v, err := WrapUint64(i)
	if err != nil {
		panic(err)
	}
	return v
}
