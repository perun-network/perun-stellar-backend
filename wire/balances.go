package wire

import (
	"bytes"
	"encoding/binary"
	"errors"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	"math/big"
	"perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wire/scval"
)

var MaxBalance = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1))

type Balances struct {
	BalA  xdr.Int128Parts
	BalB  xdr.Int128Parts
	Token xdr.ScAddress
}

const (
	SymbolBalancesBalA  xdr.ScSymbol = "bal_a"
	SymbolBalancesBalB  xdr.ScSymbol = "bal_b"
	SymbolBalancesToken xdr.ScSymbol = "token"
)

func (b Balances) ToScVal() (xdr.ScVal, error) {
	var err error
	balA, err := scval.WrapInt128Parts(b.BalA)
	if err != nil {
		return xdr.ScVal{}, err
	}
	balB, err := scval.WrapInt128Parts(b.BalB)
	if err != nil {
		return xdr.ScVal{}, err
	}
	token, err := scval.WrapScAddress(b.Token)
	if err != nil {
		return xdr.ScVal{}, err
	}
	m, err := MakeSymbolScMap(
		[]xdr.ScSymbol{
			SymbolBalancesBalA,
			SymbolBalancesBalB,
			SymbolBalancesToken,
		},
		[]xdr.ScVal{balA, balB, token},
	)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return scval.WrapScMap(m)
}

func (b *Balances) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map")
	}
	if len(*m) != 3 {
		return errors.New("expected map of length 3")
	}
	balAVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolBalancesBalA), *m)
	if err != nil {
		return err
	}
	balA, ok := balAVal.GetI128()
	if !ok {
		return errors.New("expected i128")
	}
	balBVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolBalancesBalB), *m)
	if err != nil {
		return err
	}
	balB, ok := balBVal.GetI128()
	if !ok {
		return errors.New("expected i128")
	}
	tokenVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolBalancesToken), *m)
	if err != nil {
		return err
	}
	token, ok := tokenVal.GetAddress()
	if !ok {
		return errors.New("expected address")
	}
	b.BalA = balA
	b.BalB = balB
	b.Token = token
	return nil
}

func BalancesFromScVal(v xdr.ScVal) (Balances, error) {
	var b Balances
	err := (&b).FromScVal(v)
	return b, err
}

func (b Balances) EncodeTo(e *xdr3.Encoder) error {
	v, err := b.ToScVal()
	if err != nil {
		return err
	}
	_, err = e.Encode(v)
	return err
}

func (b *Balances) DecodeFrom(d *xdr3.Decoder) (int, error) {
	var v xdr.ScVal
	n, err := d.Decode(&v)
	if err != nil {
		return n, err
	}
	return n, b.FromScVal(v)
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (b Balances) MarshalBinary() ([]byte, error) {
	buf := bytes.Buffer{}
	e := xdr3.NewEncoder(&buf)
	err := b.EncodeTo(e)
	return buf.Bytes(), err
}

func (b *Balances) UnmarshalBinary(data []byte) error {
	d := xdr3.NewDecoder(bytes.NewReader(data))
	_, err := b.DecodeFrom(d)
	return err
}

func MakeBalances(alloc channel.Allocation) (Balances, error) {
	// TODO: Move all these checks into a compatibility layer
	if err := alloc.Valid(); err != nil {
		return Balances{}, err
	}
	// single asset
	if len(alloc.Assets) != 1 {
		return Balances{}, errors.New("expected exactly one asset")
	}

	// No sub-channels
	if len(alloc.Locked) != 0 {
		return Balances{}, errors.New("expected no locked funds")
	}
	asset := alloc.Assets[0]
	sa, err := types.ToStellarAsset(asset)
	if err != nil {
		return Balances{}, err
	}
	token, err := sa.MakeScAddress()
	if err != nil {
		return Balances{}, err
	}

	if alloc.NumParts() != 2 {
		return Balances{}, errors.New("expected exactly two parts")
	}

	balA := alloc.Balance(0, asset)
	xdrBalA, err := MakeInt128Parts(balA)
	if err != nil {
		return Balances{}, err
	}

	balB := alloc.Balance(1, asset)
	xdrBalB, err := MakeInt128Parts(balB)
	if err != nil {
		return Balances{}, err
	}

	return Balances{
		BalA:  xdrBalA,
		BalB:  xdrBalB,
		Token: token,
	}, nil
}

func ToAllocation(b Balances) (*channel.Allocation, error) {
	asset, err := types.NewStellarAssetFromScAddress(b.Token)
	if err != nil {
		return nil, err
	}
	alloc := channel.NewAllocation(2, asset)

	balA, err := ToBigInt(b.BalA)
	if err != nil {
		return nil, err
	}
	alloc.SetBalance(0, asset, balA)

	balB, err := ToBigInt(b.BalB)
	if err != nil {
		return nil, err
	}
	alloc.SetBalance(1, asset, balB)
	if err = alloc.Valid(); err != nil {
		return nil, err
	}
	return alloc, nil
}

// MakeInt128Parts converts a big.Int to xdr.Int128Parts.
// It returns an error if the big.Int is negative or too large.
func MakeInt128Parts(i *big.Int) (xdr.Int128Parts, error) {
	if i.Sign() < 0 {
		return xdr.Int128Parts{}, errors.New("expected non-negative balance")
	}
	if i.Cmp(MaxBalance) > 0 {
		return xdr.Int128Parts{}, errors.New("balance too large")
	}
	b := make([]byte, 16)
	b = i.FillBytes(b)
	hi := binary.BigEndian.Uint64(b[:8])
	lo := binary.BigEndian.Uint64(b[8:])
	return xdr.Int128Parts{
		Hi: xdr.Int64(hi),
		Lo: xdr.Uint64(lo),
	}, nil
}

func ToBigInt(i xdr.Int128Parts) (*big.Int, error) {
	b := make([]byte, 16)
	binary.BigEndian.PutUint64(b[:8], uint64(i.Hi))
	binary.BigEndian.PutUint64(b[8:], uint64(i.Lo))
	return new(big.Int).SetBytes(b), nil
}
