package util

import (
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/xdr"
	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wallet"

	"crypto/rand"
	"encoding/binary"
	mathrand "math/rand"
)

func Deploy(stellarEnv *env.IntegrationTestEnv, kp *keypair.Full, hz horizon.Account, contractPath string) (xdr.ScAddress, xdr.Hash) {
	// Install contract
	installContractOp := channel.AssembleInstallContractCodeOp(kp.Address(), contractPath)
	preFlightOp, minFee := stellarEnv.PreflightHostFunctions(&hz, *installContractOp)
	_ = stellarEnv.MustSubmitOperationsWithFee(&hz, kp, minFee, &preFlightOp)

	// Create the contract
	createContractOp := channel.AssembleCreateContractOp(kp.Address(), contractPath, "a1", stellarEnv.GetPassPhrase())
	preFlightOp, minFee = stellarEnv.PreflightHostFunctions(&hz, *createContractOp)
	_, err := stellarEnv.SubmitOperationsWithFee(&hz, kp, minFee, &preFlightOp)
	if err != nil {
		panic(err)
	}
	contractID := preFlightOp.Ext.SorobanData.Resources.Footprint.ReadWrite[0].MustContractData().Contract.ContractId
	contractHash := preFlightOp.Ext.SorobanData.Resources.Footprint.ReadOnly[0].MustContractCode().Hash
	contractIDAddress := xdr.ScAddress{
		Type:       xdr.ScAddressTypeScAddressTypeContract,
		ContractId: contractID,
	}
	return contractIDAddress, contractHash

}

func MakeRandPerunWallet() (*wallet.EphemeralWallet, *wallet.Account, *keypair.Full) {
	w := wallet.NewEphemeralWallet()

	// Read 8 bytes from crypto/rand
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}

	// Convert 8 bytes to uint64 for seeding math/rand
	seed := binary.LittleEndian.Uint64(b[:])

	// Create a math/rand.Rand with the seed
	r := mathrand.New(mathrand.NewSource(int64(seed)))

	acc, kp, err := w.AddNewAccount(r)
	if err != nil {
		panic(err)
	}
	return w, acc, kp
}

func NewRandAsset() types.StellarAsset {
	var contractID xdr.Hash
	_, err := rand.Read(contractID[:])
	if err != nil {
		panic(err)
	}

	stellarAsset := types.NewStellarAsset(contractID)

	return *stellarAsset

}
