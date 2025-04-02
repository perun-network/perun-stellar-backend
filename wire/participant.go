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
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/wire"

	assettypes "perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wallet/types"
	"perun.network/perun-stellar-backend/wire/scval"
)

const (
	CCPubKeyLength                   = 65
	CCAddrLength                     = 20
	SymbolStellarAddr   xdr.ScSymbol = "stellar_addr"
	SymbolStellarPubKey xdr.ScSymbol = "stellar_pubkey"
	SymbolCCAddress     xdr.ScSymbol = "cc_addr"
)

// WirePart represents a participant on the wire.
//
//nolint:golint
type WirePart struct {
	*types.Participant
}

// Equal compares two WirePart addresses.
func (w WirePart) Equal(address wire.Address) bool {
	return w.Participant.Equal(address.(*WirePart).Participant)
}

// Cmp compares two WirePart addresses.
func (w WirePart) Cmp(address wire.Address) int {
	if w.Participant.Equal(address.(*WirePart).Participant) {
		return 0
	}
	return 1
}

// Verify verifies a message signature.
func (w WirePart) Verify(_ []byte, sig []byte) error {
	if !bytes.Equal(sig, []byte("Authenticate")) {
		return errors.New("invalid signature")
	}
	return nil
}

// MarshalBinary encodes a WirePart to an xdr.Encoder.
func (w WirePart) MarshalBinary() ([]byte, error) {
	return w.Participant.MarshalBinary()
}

// UnmarshalBinary decodes a WirePart from binary data.
func (w *WirePart) UnmarshalBinary(data []byte) error {
	return w.Participant.UnmarshalBinary(data)
}

// NewWirePart creates a new WirePart from a Participant.
func NewWirePart(participant *types.Participant) *WirePart {
	return &WirePart{participant}
}

// Verify verifies a message signature.
// It returns an error if the signature is invalid.
func (p Participant) Verify(_ []byte, sig []byte) error {
	if !bytes.Equal(sig, []byte("Authenticate")) {
		return errors.New("invalid signature")
	}
	return nil
}

// Participant represents a participant on the soroban-contract.
type Participant struct {
	StellarAddr   xdr.ScAddress
	StellarPubKey xdr.ScBytes
	CCAddr        xdr.ScBytes // Ethereum Address cannot be encoded as xdr.ScAddress, has to be encoded as xdr.ScBytes
}

// ToScVal encodes a Participant to an xdr.ScVal.
func (p Participant) ToScVal() (xdr.ScVal, error) {
	stellarAddr, err := scval.WrapScAddress(p.StellarAddr)
	if err != nil {
		return xdr.ScVal{}, err
	}

	if len(p.StellarPubKey) != CCPubKeyLength {
		return xdr.ScVal{}, errors.New("invalid Layer 2 public key length")
	}

	if len(p.CCAddr) != CCAddrLength {
		return xdr.ScVal{}, errors.New("invalid cross-chain address length")
	}

	stellarPubKeyVal, err := scval.MustWrapScBytes(p.StellarPubKey)
	if err != nil {
		return xdr.ScVal{}, err
	}

	ccAddr, err := scval.WrapScBytes(p.CCAddr)
	if err != nil {
		return xdr.ScVal{}, err
	}
	m, err := MakeSymbolScMap(
		[]xdr.ScSymbol{
			SymbolStellarAddr,
			SymbolStellarPubKey,
			SymbolCCAddress,
		},
		[]xdr.ScVal{stellarAddr, stellarPubKeyVal, ccAddr},
	)
	if err != nil {
		return xdr.ScVal{}, err
	}
	return scval.WrapScMap(m)
}

// FromScVal decodes a Participant from an xdr.ScVal.
func (p *Participant) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map decoding Participant")
	}
	if len(*m) != 3 { //nolint:gomnd
		return errors.New("expected map of length 3")
	}
	stellarAddrVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolStellarAddr), *m)
	if err != nil {
		return err
	}
	stellarAddr, ok := stellarAddrVal.GetAddress()
	if !ok {
		return errors.New("expected Stellar address")
	}
	// For Cross-chain, this comes from an enum in the contract
	stellarPubKeyVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolStellarPubKey), *m)
	if err != nil {
		return err
	}

	stellarPubKey, ok := stellarPubKeyVal.GetBytes()
	if !ok {
		return errors.New("expected bytes decoding stellarPubKeyVal")
	}

	if len(stellarPubKey) != CCPubKeyLength {
		return errors.New("invalid stellar public key length")
	}

	ccAddrVal, err := GetMapValue(scval.MustWrapScSymbol(SymbolCCAddress), *m)
	if err != nil {
		return err
	}
	ccAddr, ok := ccAddrVal.GetBytes()
	if !ok {
		return errors.New("expected bytes decoding ccAddrVal")
	}
	if len(ccAddr) != CCAddrLength {
		return errors.New("invalid public key length")
	}

	p.StellarAddr = stellarAddr
	p.StellarPubKey = stellarPubKey
	p.CCAddr = ccAddr
	return nil
}

// EncodeTo encodes a Participant to an xdr.Encoder.
func (p Participant) EncodeTo(e *xdr3.Encoder) error {
	v, err := p.ToScVal()
	if err != nil {
		return err
	}
	return v.EncodeTo(e)
}

// DecodeFrom decodes a Participant from an xdr.Decoder.
func (p *Participant) DecodeFrom(d *xdr3.Decoder) (int, error) {
	var v xdr.ScVal
	i, err := d.Decode(&v)
	if err != nil {
		return i, err
	}
	return i, p.FromScVal(v)
}

// MarshalBinary encodes a Participant to binary data.
func (p Participant) MarshalBinary() ([]byte, error) {
	buf := bytes.Buffer{}
	e := xdr3.NewEncoder(&buf)
	err := p.EncodeTo(e)
	return buf.Bytes(), err
}

// UnmarshalBinary decodes a Participant from binary data.
func (p *Participant) UnmarshalBinary(data []byte) error {
	d := xdr3.NewDecoder(bytes.NewReader(data))
	_, err := p.DecodeFrom(d)
	return err
}

// ParticipantFromScVal creates a Participant from an xdr.ScVal.
func ParticipantFromScVal(v xdr.ScVal) (Participant, error) {
	var p Participant
	err := (&p).FromScVal(v)
	return p, err
}

// PublicKeyToBytes convert ECDSA public key to bytes.
func PublicKeyToBytes(pubKey *ecdsa.PublicKey) []byte {
	// Get the X and Y coordinates
	xBytes := pubKey.X.Bytes()
	yBytes := pubKey.Y.Bytes()

	// Calculate the byte lengths for fixed-width representation.
	curveBits := pubKey.Curve.Params().BitSize
	curveByteSize := (curveBits + 7) / 8 //nolint:gomnd

	// Create fixed-size byte slices for X and Y.
	xPadded := make([]byte, curveByteSize)
	yPadded := make([]byte, curveByteSize)
	copy(xPadded[curveByteSize-len(xBytes):], xBytes)
	copy(yPadded[curveByteSize-len(yBytes):], yBytes)

	// Concatenate the X and Y coordinates
	pubKeyBytes := make([]byte, 0, 65)      //nolint:gomnd
	pubKeyBytes = append(pubKeyBytes, 0x04) //nolint:gomnd
	pubKeyBytes = append(pubKeyBytes, xPadded...)
	pubKeyBytes = append(pubKeyBytes, yPadded...)
	return pubKeyBytes
}

// BytesToPublicKey convert bytes back to ECDSA public key.
func BytesToPublicKey(data []byte) (*big.Int, *big.Int, error) {
	if len(data) != 65 || data[0] != 0x04 {
		return nil, nil, errors.New("invalid public key")
	}
	// Split data into X and Y
	x := new(big.Int).SetBytes(data[1:33])
	y := new(big.Int).SetBytes(data[33:])

	// Return the public key
	return x, y, nil
}

// MakeParticipant creates a Participant from a types.Participant.
func MakeParticipant(participant types.Participant) (Participant, error) {
	stellarAddr, err := assettypes.AccountAddressFromAddress(participant.StellarAddress)
	if err != nil {
		return Participant{}, err
	}
	if participant.StellarPubKey == nil {
		return Participant{}, errors.New("invalid Stellar public key length")
	}

	if !participant.StellarPubKey.Curve.IsOnCurve(participant.StellarPubKey.X, participant.StellarPubKey.Y) {
		return Participant{}, errors.New("stellar public key is not on the curve")
	}
	pk := PublicKeyToBytes(participant.StellarPubKey)

	stellarPubKey := xdr.ScBytes(pk)
	ccAddr := xdr.ScBytes(participant.CCAddr[:])
	return Participant{
		StellarAddr:   stellarAddr,
		StellarPubKey: stellarPubKey,
		CCAddr:        ccAddr,
	}, nil
}

// ToParticipant converts a Participant to a types.Participant.
func ToParticipant(participant Participant) (types.Participant, error) {
	kp, err := assettypes.ToAccountAddress(participant.StellarAddr)
	if err != nil {
		return types.Participant{}, err
	}

	var ccAddr [20]byte
	copy(ccAddr[:], participant.CCAddr[:])

	if len(participant.CCAddr) != 20 { //nolint:gomnd
		return types.Participant{}, errors.New("invalid cross-chain secp256k1 address length")
	}
	// Choose the curve (assuming P256 for this example)
	curve := secp256k1.S256()

	// Unmarshal the bytes back into X and Y coordinates
	x, y, err := BytesToPublicKey(participant.StellarPubKey)
	if x == nil || y == nil || err != nil {
		return types.Participant{}, fmt.Errorf("invalid public key data %v", err)
	}

	// Create an ECDSA public key with the curve and the extracted X, Y coordinates
	pubKey := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}
	return *types.NewParticipant(kp, pubKey, ccAddr), nil
}
