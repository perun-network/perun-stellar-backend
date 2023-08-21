package wire

import (
	"errors"
	"github.com/stellar/go/xdr"
	"perun.network/perun-stellar-backend/wire/scval"
	"sort"
	"strings"
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
