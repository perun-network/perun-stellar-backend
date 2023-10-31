package util

import (
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/xdr"
	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wallet"

	//pkgtest "polycry.pt/poly-go/test"
	"crypto/rand"
	"encoding/binary"
	mathrand "math/rand"
)

const PerunContractPath = "./testdata/perun_soroban_contract.wasm"

func Deploy(stellarEnv *env.IntegrationTestEnv, kpAlice *keypair.Full, hzDeployer horizon.Account) xdr.ScAddress {
	// Install contract
	installContractOp := channel.AssembleInstallContractCodeOp(kpAlice.Address(), PerunContractPath)
	preFlightOp, minFee := stellarEnv.PreflightHostFunctions(&hzDeployer, *installContractOp)
	_ = stellarEnv.MustSubmitOperationsWithFee(&hzDeployer, kpAlice, minFee, &preFlightOp)

	// Create the contract
	createContractOp := channel.AssembleCreateContractOp(kpAlice.Address(), PerunContractPath, "a1", stellarEnv.GetPassPhrase())
	preFlightOp, minFee = stellarEnv.PreflightHostFunctions(&hzDeployer, *createContractOp)
	_, err := stellarEnv.SubmitOperationsWithFee(&hzDeployer, kpAlice, minFee, &preFlightOp)
	if err != nil {
		panic(err)
	}
	contractID := preFlightOp.Ext.SorobanData.Resources.Footprint.ReadWrite[0].MustContractData().Contract.ContractId
	contractIDAddress := xdr.ScAddress{
		Type:       xdr.ScAddressTypeScAddressTypeContract,
		ContractId: contractID,
	}
	return contractIDAddress

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
