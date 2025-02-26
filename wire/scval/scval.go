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

func WrapScAddress(address xdr.ScAddress) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvAddress, address)
}

func WrapScAddresses(address []xdr.ScAddress) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvAddress, address)
}

func MustWrapScAddress(address xdr.ScAddress) (xdr.ScVal, error) {
	v, err := WrapScAddress(address)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return v, nil
}

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

func WrapInt128Parts(parts xdr.Int128Parts) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvI128, parts)
}

func WrapVec(scVec xdr.ScVec) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvVec, &scVec)
}

func MustWrapInt128Parts(parts xdr.Int128Parts) xdr.ScVal {
	v, err := WrapInt128Parts(parts)
	if err != nil {
		panic(err)
	}
	return v
}

func WrapScMap(m xdr.ScMap) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvMap, &m)
}

func MustWrapScMap(m xdr.ScMap) xdr.ScVal {
	v, err := WrapScMap(m)
	if err != nil {
		panic(err)
	}
	return v
}

func WrapScSymbol(symbol xdr.ScSymbol) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvSymbol, symbol)
}

func MustWrapScSymbol(symbol xdr.ScSymbol) xdr.ScVal {
	v, err := WrapScSymbol(symbol)
	if err != nil {
		panic(err)
	}
	return v
}

func WrapScString(str xdr.ScString) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvString, str)
}

func WrapScUint64(ui xdr.Uint64) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvU64, ui)
}

func MustWrapScString(str xdr.ScString) (xdr.ScVal, error) {
	v, err := WrapScString(str)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return v, nil
}

func MustWrapScUint64(ui xdr.Uint64) (xdr.ScVal, error) {
	v, err := WrapScUint64(ui)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return v, nil
}

func WrapScBytes(b xdr.ScBytes) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvBytes, b)
}

func WrapScVec(v xdr.ScVec) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvVec, &v)
}

func MustWrapScBytes(b xdr.ScBytes) (xdr.ScVal, error) {
	v, err := WrapScBytes(b)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return v, nil
}

func WrapUint64(i xdr.Uint64) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvU64, i)
}

func MustWrapUint64(i xdr.Uint64) xdr.ScVal {
	v, err := WrapUint64(i)
	if err != nil {
		panic(err)
	}
	return v
}
