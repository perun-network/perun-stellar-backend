package scval

import "github.com/stellar/go/xdr"

func WrapScAddress(address xdr.ScAddress) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvAddress, address)
}

func MustWrapScAddress(address xdr.ScAddress) xdr.ScVal {
	v, err := WrapScAddress(address)
	if err != nil {
		panic(err)
	}
	return v
}

func WrapInt128Parts(parts xdr.Int128Parts) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvI128, parts)
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

func MustWrapScString(str xdr.ScString) xdr.ScVal {
	v, err := WrapScString(str)
	if err != nil {
		panic(err)
	}
	return v
}

func WrapScBytes(b xdr.ScBytes) (xdr.ScVal, error) {
	return xdr.NewScVal(xdr.ScValTypeScvBytes, b)
}

func MustWrapScBytes(b xdr.ScBytes) xdr.ScVal {
	v, err := WrapScBytes(b)
	if err != nil {
		panic(err)
	}
	return v
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
