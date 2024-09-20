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
	"crypto/ed25519"
	"errors"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/xdr"
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

func MakeParticipant(participant types.Participant) (Participant, error) {
	stellarAddr, err := assettypes.MakeAccountAddress(&participant.StellarAddress)
	if err != nil {
		return Participant{}, err
	}
	if len(participant.StellarPubKey) != StellarPubKeyLength {
		return Participant{}, errors.New("invalid Stellar public key length")
	}
	if len(participant.CCAddr) != CCAddrLength {
		return Participant{}, errors.New("invalid cross-chain address length")
	}
	stellarPubKey := xdr.ScBytes(participant.StellarPubKey)
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

	if len(participant.StellarPubKey) != ed25519.PublicKeySize {
		return types.Participant{}, errors.New("invalid public key length")
	}
	if len(participant.CCAddr) != 20 {
		return types.Participant{}, errors.New("invalid cross-chain secp256k1 address length")
	}
	return *types.NewParticipant(kp, ed25519.PublicKey(participant.StellarPubKey), ccAddr), nil
}
