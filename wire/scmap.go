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

package wire

import (
	"errors"
	"sort"
	"strings"

	"github.com/stellar/go/xdr"

	"perun.network/perun-stellar-backend/wire/scval"
)

// MakeSymbolScMap creates a xdr.ScMap from a slice of symbols and a slice of values.
// The entries are sorted lexicographically by symbol. We expect that keys does not contain duplicates.
func MakeSymbolScMap(keys []xdr.ScSymbol, values []xdr.ScVal) (xdr.ScMap, error) {
	if len(keys) != len(values) {
		return xdr.ScMap{}, errors.New("keys and values must have the same length")
	}
	m := make(xdr.ScMap, len(keys))
	for i, k := range keys {
		m[i] = xdr.ScMapEntry{
			Key: scval.MustWrapScSymbol(k),
			Val: values[i],
		}
	}
	sort.Slice(m, func(i, j int) bool {
		return strings.Compare(string(m[i].Key.MustSym()), string(m[j].Key.MustSym())) < 0
	})
	return m, nil
}

func GetScMapEntry(key xdr.ScVal, m xdr.ScMap) (xdr.ScMapEntry, error) {
	for _, v := range m {
		if v.Key.Equals(key) {
			return v, nil
		}
	}

	return xdr.ScMapEntry{}, errors.New("key not found")
}

func GetMapValue(key xdr.ScVal, m xdr.ScMap) (xdr.ScVal, error) {
	entry, err := GetScMapEntry(key, m)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return entry.Val, nil
}

func GetScMapValueFromSymbol(key xdr.ScSymbol, m xdr.ScMap) (xdr.ScVal, error) {
	keyVal, err := scval.WrapScSymbol(key)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return GetMapValue(keyVal, m)
}
