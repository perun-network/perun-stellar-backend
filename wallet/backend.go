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

package wallet

import (
	"io"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/wire/perunio"

	"perun.network/perun-stellar-backend/wallet/types"
)

// SignatureLength is the length of a signature in bytes.
const SignatureLength = 64

type backend struct{}

var Backend = backend{}

func init() {
	wallet.SetBackend(Backend, types.StellarBackendID)
}

func (b backend) NewAddress() wallet.Address {
	return &types.Participant{}
}

// DecodeSig decodes a signature of length SignatureLength from the reader.
func (b backend) DecodeSig(reader io.Reader) (wallet.Sig, error) {
	buf := make(wallet.Sig, 65) //nolint:gomnd
	return buf, perunio.Decode(reader, &buf)
}

func (b backend) VerifySignature(msg []byte, sig wallet.Sig, a wallet.Address) (bool, error) {
	p, ok := a.(*types.Participant)
	if !ok {
		return false, errors.New("participant has invalid type")
	}
	hash := crypto.Keccak256(msg)
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	hash = crypto.Keccak256(prefix, hash)
	sigCopy := make([]byte, 65) //nolint:gomnd
	copy(sigCopy, sig)
	if len(sigCopy) == 65 && (sigCopy[65-1] >= 27) { //nolint:gomnd
		sigCopy[65-1] -= 27
	}
	pk, err := crypto.SigToPub(hash, sigCopy)
	if err != nil {
		return false, errors.WithStack(err)
	}
	return pk.X.Cmp(p.StellarPubKey.X) == 0 && pk.Y.Cmp(p.StellarPubKey.Y) == 0 && pk.Curve == p.StellarPubKey.Curve, nil
}
