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
	"math/big"
	"perun.network/go-perun/wallet"

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
		assetID     AssetID
		AssetHolder wallet.Address
	}

	AssetID struct {
		backendID uint32
		LedgerID  ChainID
	}

	// AssetMapKey is the map key representation of an asset.
	AssetMapKey string
)

func MakeEthAsset(id *big.Int, holder wallet.Address) EthAsset {
	return EthAsset{assetID: AssetID{backendID: 1, LedgerID: MakeChainID(id)}, AssetHolder: holder}
}

func (id AssetID) BackendID() uint32 {
	return id.backendID
}

func (id AssetID) LedgerId() multi.LedgerID {
	return &id.LedgerID
}

// MakeAssetID makes a AssetID for the given id.
func MakeAssetID(id *big.Int) multi.AssetID {
	if id.Sign() < 0 {
		panic("must not be smaller than zero")
	}
	return AssetID{backendID: 1, LedgerID: MakeChainID(id)}
}

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
	err := perunio.Encode(&buf, a.assetID.LedgerID, a.assetID.backendID, &a.AssetHolder)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary unmarshals the asset from its binary representation.
func (a *EthAsset) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	return perunio.Decode(buf, &a.assetID.LedgerID, &a.assetID.backendID, &a.AssetHolder)
}

// LedgerID returns the ledger ID the asset lives on.
func (a EthAsset) LedgerID() multi.LedgerID {
	return a.AssetID().LedgerId()
}

// LedgerID returns the ledger ID the asset lives on.
func (a EthAsset) AssetID() multi.AssetID {
	return a.assetID
}

// Equal returns true iff the asset equals the given asset.
func (a EthAsset) Equal(b channel.Asset) bool {
	ethAsset, ok := b.(*EthAsset)
	if !ok {
		return false
	}
	return a.assetID.LedgerID.MapKey() == ethAsset.assetID.LedgerID.MapKey() && a.AssetHolder.Equal(ethAsset.AssetHolder)
}

// Address returns the address of the asset.
func (a EthAsset) Address() string {
	return a.AssetHolder.String()
}
