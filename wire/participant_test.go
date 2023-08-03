package wire_test

import (
	"github.com/stretchr/testify/require"
	"perun.network/perun-stellar-backend/wire"
	"testing"
)

func TestParticipant(t *testing.T) {
	// Participant XDR generated by soroban contract
	x := []byte{0, 0, 0, 17, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 15, 0, 0, 0, 4, 97, 100, 100, 114, 0, 0, 0, 18, 0, 0, 0, 1, 111, 69, 81, 199, 239, 4, 247, 14, 49, 177, 211, 74, 68, 253, 8, 232, 85, 220, 39, 55, 21, 152, 51, 165, 106, 167, 168, 9, 185, 195, 55, 133, 0, 0, 0, 15, 0, 0, 0, 6, 112, 117, 98, 107, 101, 121, 0, 0, 0, 0, 0, 13, 0, 0, 0, 32, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	p := &wire.Participant{}
	err := p.UnmarshalBinary(x)
	require.NoError(t, err)
	res, err := p.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, x, res)
}
