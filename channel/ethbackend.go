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
	"github.com/pkg/errors"
	"log"

	"math/big"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/channel/types"
	wtypes "perun.network/perun-stellar-backend/wallet/types"
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
		BackendID: EthBackendID,
		ChainID:   ethAsset.ChainID.Int,
		EthAsset:  ethAsset.EthAddress(),
		CCAsset:   []byte{},
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
		BackendID: wtypes.StellarBackendID,
		ChainID:   big.NewInt(wtypes.StellarBackendID),
		EthAsset:  common.Address{},
		CCAsset:   assetBytes,
	}
}

// EncodeState encodes the state as with abi.encode() in the smart contracts.
func EncodeEthState(state *EthChannelState) ([]byte, error) {
	args := abi.Arguments{{Type: abiState}}
	enc, err := args.Pack(*state)
	return enc, errors.WithStack(err)
}

var (
	// compile time check that we implement the channel backend interface.
	// _ channel.Backend = new(Backend)
	// Definition of ABI datatypes.
	abiUint256, _ = abi.NewType("uint256", "", nil)
	abiAddress, _ = abi.NewType("address", "", nil)
	abiBytes32, _ = abi.NewType("bytes32", "", nil)
	abiParams     abi.Type
	abiState      abi.Type
	abiProgress   abi.Method
	abiRegister   abi.Method
	// MaxBalance is the maximum amount of funds per asset that a user can possess.
	// It is set to 2 ^ 256 - 1.
	MaxBalance = abi.MaxUint256
)

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
	Balances [][]*big.Int
	Locked   []ChannelSubAlloc
}

type ChannelAsset struct {
	BackendID int
	ChainID   *big.Int
	EthAsset  common.Address
	CCAsset   []byte
}

type ChannelSubAlloc struct {
	ID       map[wallet.BackendID][32]byte
	Balances []*big.Int
	IndexMap []uint16
}
