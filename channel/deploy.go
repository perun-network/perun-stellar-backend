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

package channel

import (
	"crypto/sha256"
	"os"

	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

const PerunContractPath = "../testdata/perun_soroban_multi_contract.wasm"

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
	saltParameter := xdr.Uint256(salt)

	accountID := xdr.MustAddress(sourceAccount)
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
							AccountId: &accountID,
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
