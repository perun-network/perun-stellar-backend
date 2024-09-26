// Copyright 2024 PolyCrypt GmbH
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
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
	"log"
	"math/big"
	assettypes "perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wallet/types"

	"perun.network/perun-stellar-backend/wire/scval"
)

const (
	StellarPubKeyLength              = 32
	CCPubKeyLength                   = 65
	CCAddrLength                     = 20
	SymbolStellarAddr   xdr.ScSymbol = "stellar_addr"
	SymbolStellarPubKey xdr.ScSymbol = "stellar_pubkey"
	SymbolCCAddress     xdr.ScSymbol = "cc_addr"
	ChanTypeCrossSymbol xdr.ScSymbol = "Cross"
)

type Participant struct {
	StellarAddr   xdr.ScAddress
	StellarPubKey xdr.ScBytes
	CCAddr        xdr.ScBytes // Ethereum Address cannot be encoded as xdr.ScAddress, has to be encoded as xdr.ScBytes
}

func (p Participant) ToScVal() (xdr.ScVal, error) {
	stellarAddr, err := scval.WrapScAddress(p.StellarAddr)
	if err != nil {
		return xdr.ScVal{}, err
	}

	if len(p.StellarPubKey) != StellarPubKeyLength && len(p.StellarPubKey) != CCPubKeyLength {
		log.Println(len(p.StellarPubKey))
		return xdr.ScVal{}, errors.New("invalid Layer 2 public key length")
	}

	if len(p.CCAddr) != CCAddrLength {
		return xdr.ScVal{}, errors.New("invalid cross-chain address length")
	}

	xdrSym := scval.MustWrapScSymbol(ChanTypeCrossSymbol)
	xdrStellarPubkeyBytes := scval.MustWrapScBytes(p.StellarPubKey) //p.StellarPubKey)

	stellarPubKey := xdr.ScVec{xdrSym, xdrStellarPubkeyBytes}
	stellarPubKeyVal, err := scval.WrapVec(stellarPubKey)
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

func (p *Participant) FromScVal(v xdr.ScVal) error {
	m, ok := v.GetMap()
	if !ok {
		return errors.New("expected map decoding Participant")
	}
	if len(*m) != 3 {
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

	stellarPubKeyVals, ok := stellarPubKeyVal.GetVec()
	if !ok {
		return errors.New("expected vec decoding stellarPubKeyVal")
	}

	if len(*stellarPubKeyVals) != 2 {
		return errors.New("expected vec of length 2")
	}

	stellarPubKey, ok := (*stellarPubKeyVals)[1].GetBytes()
	if !ok {
		return errors.New("expected bytes decoding stellarPubKeyVal")
	}
	_, ok = (*stellarPubKeyVals)[0].GetSym()
	if !ok {
		return errors.New("expected symbol decoding stellarPubKeyVal")
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

func (p Participant) EncodeTo(e *xdr3.Encoder) error {
	v, err := p.ToScVal()
	if err != nil {
		return err
	}
	return v.EncodeTo(e)
}

func (p *Participant) DecodeFrom(d *xdr3.Decoder) (int, error) {
	var v xdr.ScVal
	i, err := d.Decode(&v)
	if err != nil {
		return i, err
	}
	return i, p.FromScVal(v)
}

func (p Participant) MarshalBinary() ([]byte, error) {
	buf := bytes.Buffer{}
	e := xdr3.NewEncoder(&buf)
	err := p.EncodeTo(e)
	return buf.Bytes(), err
}

func (p *Participant) UnmarshalBinary(data []byte) error {
	d := xdr3.NewDecoder(bytes.NewReader(data))
	_, err := p.DecodeFrom(d)
	return err
}

func ParticipantFromScVal(v xdr.ScVal) (Participant, error) {
	var p Participant
	err := (&p).FromScVal(v)
	return p, err
}

// Convert ECDSA public key to bytes
func PublicKeyToBytes(pubKey *ecdsa.PublicKey) []byte {
	// Get the X and Y coordinates
	xBytes := pubKey.X.Bytes()
	yBytes := pubKey.Y.Bytes()

	// Calculate the byte lengths for fixed-width representation
	curveBits := pubKey.Curve.Params().BitSize
	curveByteSize := (curveBits + 7) / 8 // Calculate byte size

	// Create fixed-size byte slices for X and Y
	xPadded := make([]byte, curveByteSize)
	yPadded := make([]byte, curveByteSize)
	copy(xPadded[curveByteSize-len(xBytes):], xBytes)
	copy(yPadded[curveByteSize-len(yBytes):], yBytes)

	// Concatenate the X and Y coordinates
	pubKeyBytes := make([]byte, 0, 65)
	pubKeyBytes = append(pubKeyBytes, 0x04) // Uncompressed prefix
	pubKeyBytes = append(pubKeyBytes, xPadded...)
	pubKeyBytes = append(pubKeyBytes, yPadded...)
	return pubKeyBytes
}

// Convert bytes back to ECDSA public key
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

func MakeParticipant(participant types.Participant) (Participant, error) {
	stellarAddr, err := assettypes.MakeAccountAddress(&participant.StellarAddress)
	if err != nil {
		return Participant{}, err
	}
	if &participant.StellarPubKey == nil {
		return Participant{}, errors.New("invalid Stellar public key length")
	}

	if !participant.StellarPubKey.Curve.IsOnCurve(participant.StellarPubKey.X, participant.StellarPubKey.Y) {
		return Participant{}, errors.New("Stellar public key is not on the curve")
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

func ToParticipant(participant Participant) (types.Participant, error) {
	kp, err := assettypes.ToAccountAddress(participant.StellarAddr)
	if err != nil {
		return types.Participant{}, err
	}

	var ccAddr [20]byte
	copy(ccAddr[:], participant.CCAddr[:])

	if len(participant.CCAddr) != 20 {
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
