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
