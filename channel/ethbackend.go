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

package channel

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"log"
	"math/big"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/multi"
	"perun.network/perun-stellar-backend/channel/types"
	wtypes "perun.network/perun-stellar-backend/wallet/types"
)

// This part of the package transfers Ethereum backend functionality to encode States the same way they are encoded in the Eth Backend

// ToEthState converts a channel.State to a ChannelState struct.
func ToEthState(s *channel.State) EthChannelState {
	backends := make([]*big.Int, len(s.Allocation.Assets))
	channelIDs := make([]channel.ID, len(s.Allocation.Assets))
	for i := range s.Allocation.Assets { // we assume that for each asset there is an element in backends corresponding to the backendID the asset belongs to.
		backends[i] = big.NewInt(int64(s.Allocation.Backends[i]))
		channelIDs[i] = s.ID[s.Allocation.Backends[i]]
	}
	locked := make([]ChannelSubAlloc, len(s.Locked))
	for i, sub := range s.Locked {
		subIDs := make([]channel.ID, len(backends))
		for j := range subIDs {
			subIDs[j] = sub.ID[s.Allocation.Backends[j]]
		}
		// Create index map.
		indexMap := make([]uint16, s.NumParts())
		if len(sub.IndexMap) == 0 {
			for i := range indexMap {
				indexMap[i] = uint16(i)
			}
		} else {
			for i, x := range sub.IndexMap {
				indexMap[i] = uint16(x)
			}
		}

		locked[i] = ChannelSubAlloc{ID: subIDs, Balances: sub.Bals, IndexMap: indexMap}
	}

	// iterate over s.Allocation.Backends and check if they are of type EthAsset
	// if not, panic

	assets := make([]ChannelAsset, len(s.Allocation.Assets))

	for i, backendID := range s.Allocation.Backends {
		switch backendID {
		case EthBackendID:
			assets[i] = assetToEthAsset(s.Allocation.Assets[i])

		case wtypes.StellarBackendID:
			assets[i] = assetToStellarAsset(s.Allocation.Assets[i])

		default:
			log.Panicf("wrong backend ID: %d", backendID)
		}

	}

	outcome := ChannelAllocation{
		Assets:   assets,
		Backends: backends,
		Balances: s.Balances,
		Locked:   locked,
	}
	// Check allocation dimensions
	if len(outcome.Assets) != len(outcome.Balances) || len(s.Balances) != len(outcome.Balances) {
		log.Panic("invalid allocation dimensions")
	}
	appData, err := s.Data.MarshalBinary()
	if err != nil {
		log.Panicf("error encoding app data: %v", err)
	}
	return EthChannelState{
		ChannelID: channelIDs,
		Version:   s.Version,
		Outcome:   outcome,
		AppData:   appData,
		IsFinal:   s.IsFinal,
	}
}

func assetToEthAsset(asset channel.Asset) ChannelAsset {
	multiAsset, ok := asset.(multi.Asset)
	if !ok {
		log.Panicf("expected asset of type MultiLedgerAsset, but got wrong asset type: %T", asset)
	}
	id := new(big.Int)
	_, ok = id.SetString(string(multiAsset.AssetID().LedgerId().MapKey()), 10) // base 10 for decimal numbers
	if !ok {
		log.Panicf("Error: Failed to parse string into big.Int")
	}
	ethAddress := common.Address{}
	ethAddress.SetBytes(multiAsset.Address())
	return ChannelAsset{
		ChainID:  id,
		EthAsset: ethAddress,
		CCAsset:  make([]byte, 32),
	}
}

func assetToStellarAsset(asset channel.Asset) ChannelAsset {
	var assetBytes []byte
	var err error

	switch v := asset.(type) {
	case *types.StellarAsset:
		assetBytes, err = v.MarshalBinary()
	default:
		log.Panicf("expected asset of type Stellar or MultiAsset, but got: %T", asset)
	}

	if err != nil {
		log.Panicf("error encoding asset: %v", err)
	}

	return ChannelAsset{
		ChainID:  big.NewInt(wtypes.StellarBackendID),
		EthAsset: common.HexToAddress("0x0000000000000000000000000000000000000000"),
		CCAsset:  assetBytes,
	}
}

// EncodeState encodes the state as with abi.encode() in the smart contracts.
func EncodeEthState(state *EthChannelState) ([]byte, error) {

	// Define the top-level ABI type for the state struct.
	stateType, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "channelID", Type: "bytes32[]"},
		{Name: "version", Type: "uint64"},
		{Name: "outcome", Type: "tuple", Components: []abi.ArgumentMarshaling{
			{Name: "assets", Type: "tuple[]", Components: []abi.ArgumentMarshaling{
				{Name: "chainID", Type: "uint256"},
				{Name: "ethHolder", Type: "address"},
				{Name: "ccHolder", Type: "bytes"},
			}},
			{Name: "backends", Type: "uint256[]"},
			{Name: "balances", Type: "uint256[][]"},
			{Name: "locked", Type: "tuple[]", Components: []abi.ArgumentMarshaling{
				{Name: "ID", Type: "bytes32[]"},
				{Name: "balances", Type: "uint256[]"},
				{Name: "indexMap", Type: "uint16[]"},
			}},
		}},
		{Name: "appData", Type: "bytes"},
		{Name: "isFinal", Type: "bool"},
	})
	if err != nil {
		return nil, err
	}

	// Define the Arguments.
	args := abi.Arguments{
		{Type: stateType},
	}

	// Pack the data for encoding.
	return args.Pack(
		struct {
			ChannelID [][32]byte
			Version   uint64
			Outcome   struct {
				Assets []struct {
					ChainID   *big.Int
					EthHolder common.Address
					CcHolder  []byte
				}
				Backends []*big.Int
				Balances [][]*big.Int
				Locked   []struct {
					ID       [][32]byte
					Balances []*big.Int
					IndexMap []uint16
				}
			}
			AppData []byte
			IsFinal bool
		}{
			ChannelID: state.ChannelID,
			Version:   state.Version,
			Outcome: struct {
				Assets []struct {
					ChainID   *big.Int
					EthHolder common.Address
					CcHolder  []byte
				}
				Backends []*big.Int
				Balances [][]*big.Int
				Locked   []struct {
					ID       [][32]byte
					Balances []*big.Int
					IndexMap []uint16
				}
			}{
				Assets: func() []struct {
					ChainID   *big.Int
					EthHolder common.Address
					CcHolder  []byte
				} {
					var assets []struct {
						ChainID   *big.Int
						EthHolder common.Address
						CcHolder  []byte
					}
					for _, asset := range state.Outcome.Assets {
						assets = append(assets, struct {
							ChainID   *big.Int
							EthHolder common.Address
							CcHolder  []byte
						}{
							ChainID:   asset.ChainID,
							EthHolder: asset.EthAsset,
							CcHolder:  asset.CCAsset,
						})
					}
					return assets
				}(),
				Backends: state.Outcome.Backends,
				Balances: state.Outcome.Balances,
				Locked: func() []struct {
					ID       [][32]byte
					Balances []*big.Int
					IndexMap []uint16
				} {
					var locked []struct {
						ID       [][32]byte
						Balances []*big.Int
						IndexMap []uint16
					}
					for _, lock := range state.Outcome.Locked {
						locked = append(locked, struct {
							ID       [][32]byte
							Balances []*big.Int
							IndexMap []uint16
						}{
							ID:       lock.ID,
							Balances: lock.Balances,
							IndexMap: lock.IndexMap,
						})
					}
					return locked
				}(),
			},
			AppData: state.AppData,
			IsFinal: state.IsFinal,
		},
	)
}

// here we have ethereum methods

// ChannelState is an auto generated low-level Go binding around an user-defined struct.
type EthChannelState struct {
	ChannelID [][32]byte
	Version   uint64
	Outcome   ChannelAllocation
	AppData   []byte
	IsFinal   bool
}

type ChannelAllocation struct {
	Assets   []ChannelAsset
	Backends []*big.Int
	Balances [][]*big.Int
	Locked   []ChannelSubAlloc
}

type ChannelAsset struct {
	ChainID  *big.Int
	EthAsset common.Address
	CCAsset  []byte
}

type ChannelSubAlloc struct {
	ID       [][32]byte
	Balances []*big.Int
	IndexMap []uint16
}
