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
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"log"
	"math/big"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/channel/types"
	wtypes "perun.network/perun-stellar-backend/wallet/types"
	"strings"
)

// This part of the package transfers Ethereum backend functionality to encode States the same way they are encoded in the Eth Backend

// ToEthState converts a channel.State to a ChannelState struct.
func ToEthState(s *channel.State) EthChannelState {
	locked := make([]ChannelSubAlloc, len(s.Locked))
	for i, sub := range s.Locked {
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

		locked[i] = ChannelSubAlloc{ID: sub.ID, Balances: sub.Bals, IndexMap: indexMap}
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
		Backends: s.Allocation.Backends,
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
		ChannelID: s.ID,
		Version:   s.Version,
		Outcome:   outcome,
		AppData:   appData,
		IsFinal:   s.IsFinal,
	}
}

func assetToEthAsset(asset channel.Asset) ChannelAsset {
	ethAsset, ok := asset.(*types.EthAsset)
	if !ok {
		log.Panicf("expected asset of type Ethereum, but got wrong asset type: %T", asset)
	}

	return ChannelAsset{
		ChainID:  ethAsset.ChainID.Int,
		EthAsset: ethAsset.EthAddress(),
		CCAsset:  make([]byte, 0),
	}
}

func assetToStellarAsset(asset channel.Asset) ChannelAsset {
	stellarAsset, ok := asset.(*types.StellarAsset)
	if !ok {
		log.Panicf("expected asset of type Stellar, but got wrong asset type: %T", asset)
	}

	assetBytes, err := stellarAsset.MarshalBinary()
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
	// Define the ABI spec for the state type
	const stateType = `tuple(
        bytes32[] channelID, 
        uint64 version, 
        tuple(
            tuple(uint256 chainID, address ethHolder, bytes ccHolder)[] assets, 
            uint256[] backends, 
            uint256[][] balances, 
            tuple(bytes32[] ID, uint256[] balances, uint16[] indexMap)[] locked
        ) outcome, 
        bytes appData, 
        bool isFinal
    )`

	// Create a new ABI object with just the stateType
	parsedAbi, err := abi.JSON(strings.NewReader(fmt.Sprintf(`[{"type":"%s"}]`, stateType)))
	if err != nil {
		return nil, err
	}

	// Encode the state data into ABI format
	encoded, err := parsedAbi.Pack("", state)
	if err != nil {
		return nil, err
	}

	return encoded, nil
}

// here we have ethereum methods

// ChannelState is an auto generated low-level Go binding around an user-defined struct.
type EthChannelState struct {
	ChannelID map[wallet.BackendID][32]byte
	Version   uint64
	Outcome   ChannelAllocation
	AppData   []byte
	IsFinal   bool
}

type ChannelAllocation struct {
	Assets   []ChannelAsset
	Backends []wallet.BackendID
	Balances [][]*big.Int
	Locked   []ChannelSubAlloc
}

type ChannelAsset struct {
	ChainID  *big.Int
	EthAsset common.Address
	CCAsset  []byte
}

type ChannelSubAlloc struct {
	ID       map[wallet.BackendID][32]byte
	Balances []*big.Int
	IndexMap []uint16
}
