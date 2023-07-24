package wallet

import (
	"crypto/ed25519"
	"errors"
	"io"
	"perun.network/go-perun/wallet"
)

// SignatureLength is the length of a signature in bytes.
const SignatureLength = 64

type backend struct{}

var Backend = backend{}

func init() {
	wallet.SetBackend(Backend)
}

func (b backend) NewAddress() wallet.Address {
	return &Participant{}
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
	p, ok := a.(*Participant)
	if !ok {
		return false, errors.New("Participant has invalid type")
	}
	if len(sig) != ed25519.SignatureSize {
		return false, errors.New("invalid signature size")
	}
	return ed25519.Verify(p.PublicKey, msg, sig), nil
}
