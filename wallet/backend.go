// Copyright 2023 PolyCrypt GmbH
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

package wallet

import (
	"crypto/ed25519"
	"errors"
	"io"
	"perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/wallet/types"
)

// SignatureLength is the length of a signature in bytes.
const SignatureLength = 64

type backend struct{}

var Backend = backend{}

func init() {
	wallet.SetBackend(Backend)
}

func (b backend) NewAddress() wallet.Address {
	return &types.Participant{}
}

// DecodeSig decodes a signature of length SignatureLength from the reader.
func (b backend) DecodeSig(reader io.Reader) (wallet.Sig, error) {
	sig := make([]byte, SignatureLength)
	if _, err := io.ReadFull(reader, sig); err != nil {
		return nil, err
	}
	return sig, nil
}

func (b backend) VerifySignature(msg []byte, sig wallet.Sig, a wallet.Address) (bool, error) {
	p, ok := a.(*types.Participant)
	if !ok {
		return false, errors.New("participant has invalid type")
	}
	if len(sig) != ed25519.SignatureSize {
		return false, errors.New("invalid signature size")
	}
	return ed25519.Verify(p.PublicKey, msg, sig), nil
}
