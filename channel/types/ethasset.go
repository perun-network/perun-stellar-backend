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

package types

import (
	"bytes"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	wtypes "perun.network/perun-stellar-backend/wallet/types"

	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/multi"
	"perun.network/go-perun/wire/perunio"
)

var _ channel.Asset = new(EthAsset)

// ChainID identifies a specific Ethereum backend.
type ChainID struct {
	*big.Int
}

// MakeChainID makes a ChainID for the given id.
func MakeChainID(id *big.Int) ChainID {
	if id.Sign() < 0 {
		panic("must not be smaller than zero")
	}
	return ChainID{id}
}

// UnmarshalBinary unmarshals the chainID from its binary representation.
func (id *ChainID) UnmarshalBinary(data []byte) error {
	id.Int = new(big.Int).SetBytes(data)
	return nil
}

// MarshalBinary marshals the chainID into its binary representation.
func (id ChainID) MarshalBinary() (data []byte, err error) {
	if id.Sign() == -1 {
		return nil, errors.New("cannot marshal negative ChainID")
	}
	return id.Bytes(), nil
}

// MapKey returns the asset's map key representation.
func (id ChainID) MapKey() multi.LedgerIDMapKey {
	return multi.LedgerIDMapKey(id.Int.String())
}

type (
	// Asset is an Ethereum asset.
	EthAsset struct {
		ChainID     ChainID
		AssetHolder wtypes.EthAddress
	}

	// AssetMapKey is the map key representation of an asset.
	AssetMapKey string
)

// MapKey returns the asset's map key representation.
func (a EthAsset) MapKey() AssetMapKey {
	d, err := a.MarshalBinary()
	if err != nil {
		panic(err)
	}

	return AssetMapKey(d)
}

// MarshalBinary marshals the asset into its binary representation.
func (a EthAsset) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	err := perunio.Encode(&buf, a.ChainID, &a.AssetHolder)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary unmarshals the asset from its binary representation.
func (a *EthAsset) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	return perunio.Decode(buf, &a.ChainID, &a.AssetHolder)
}

// LedgerID returns the ledger ID the asset lives on.
func (a EthAsset) LedgerID() multi.LedgerID {
	return &a.ChainID
}

// NewAsset creates a new asset from an chainID and the AssetHolder address.
func NewAsset(chainID *big.Int, assetHolder common.Address) *EthAsset {
	id := MakeChainID(chainID)
	return &EthAsset{id, *wtypes.AsWalletAddr(assetHolder)}
}

// EthAddress returns the Ethereum address of the asset.
func (a EthAsset) EthAddress() common.Address {
	return common.Address(a.AssetHolder)
}

// Equal returns true iff the asset equals the given asset.
func (a EthAsset) Equal(b channel.Asset) bool {
	ethAsset, ok := b.(*EthAsset)
	if !ok {
		return false
	}
	return a.ChainID.MapKey() == ethAsset.ChainID.MapKey() && a.EthAddress() == ethAsset.EthAddress()
}
