package channel

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"log"
	"os"
	//"perun.network/perun-stellar-backend/channel/env"
)

const PerunContractPath = "../testdata/perun_soroban_contract.wasm"

// func InstallAndCreateContract(reqAlice *YourReqType, kpAlice YourKeyType) error { // Assuming "YourReqType" and "YourKeyType" are the types of `reqAlice` and `kpAlice` respectively.
// 	// Install the contract
// 	installContractOp := AssembleInstallContractCodeOp(kpAlice.Address(), PerunContractPath)
// 	preFlightOp, minFee := env.PreflightHostFunctions(reqAlice, *installContractOp)
// 	if err := itest.MustSubmitOperationsWithFee(reqAlice, kpAlice, 5*minFee, &preFlightOp); err != nil {
// 		return err
// 	}

// 	// Create the contract
// 	createContractOp := client.AssembleCreateContractOp(kpAlice.Address(), client.PerunContractPath, "a1", itest.GetPassPhrase())
// 	preFlightOp, minFee = itest.PreflightHostFunctions(reqAlice, *createContractOp)
// 	if _, err := itest.SubmitOperationsWithFee(reqAlice, kpAlice, minFee, &preFlightOp); err != nil {
// 		return err
// 	}

// 	return nil
// }

func AssembleInstallContractCodeOp(sourceAccount string, wasmFileName string) *txnbuild.InvokeHostFunction {
	// Assemble the InvokeHostFunction UploadContractWasm operation:
	// CAP-0047 - https://github.com/stellar/stellar-protocol/blob/master/core/cap-0047.md#creating-a-contract-using-invokehostfunctionop

	contract, err := os.ReadFile(wasmFileName)
	if err != nil {
		panic(err)
	}

	return &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeUploadContractWasm,
			Wasm: &contract,
		},
		SourceAccount: sourceAccount,
	}
}

func AssembleCreateContractOp(sourceAccount string, wasmFileName string, contractSalt string, passPhrase string) *txnbuild.InvokeHostFunction {
	// Assemble the InvokeHostFunction CreateContract operation:
	// CAP-0047 - https://github.com/stellar/stellar-protocol/blob/master/core/cap-0047.md#creating-a-contract-using-invokehostfunctionop

	contract, err := os.ReadFile(wasmFileName)
	if err != nil {
		panic(err)
	}

	salt := sha256.Sum256([]byte(contractSalt))
	log.Printf("Salt hash: %v", hex.EncodeToString(salt[:]))
	saltParameter := xdr.Uint256(salt)

	accountId := xdr.MustAddress(sourceAccount)
	contractHash := xdr.Hash(sha256.Sum256(contract))

	return &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeCreateContract,
			CreateContract: &xdr.CreateContractArgs{
				ContractIdPreimage: xdr.ContractIdPreimage{
					Type: xdr.ContractIdPreimageTypeContractIdPreimageFromAddress,
					FromAddress: &xdr.ContractIdPreimageFromAddress{
						Address: xdr.ScAddress{
							Type:      xdr.ScAddressTypeScAddressTypeAccount,
							AccountId: &accountId,
						},
						Salt: saltParameter,
					},
				},
				Executable: xdr.ContractExecutable{
					Type:     xdr.ContractExecutableTypeContractExecutableWasm,
					WasmHash: &contractHash,
				},
			},
		},
		SourceAccount: sourceAccount,
	}
}
