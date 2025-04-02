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
	"context"
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/stellar/go/xdr"
	pchannel "perun.network/go-perun/channel"

	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wire"
)

const (
	MaxIterationsUntilAbort = 30
	DefaultPollingInterval  = time.Duration(4) * time.Second
)

// Funder is a struct that implements the Funder interface for Stellar.
type Funder struct {
	cb              *client.ContractBackend
	perunAddr       xdr.ScAddress
	assetAddrs      []xdr.ScVal
	maxIters        int
	pollingInterval time.Duration
}

// NewFunder returns a new Funder.
func NewFunder(acc *wallet.Account, contractBackend *client.ContractBackend, perunAddr xdr.ScAddress, assetAddrs []xdr.ScVal) *Funder {
	return &Funder{
		cb:              contractBackend,
		perunAddr:       perunAddr,
		assetAddrs:      assetAddrs,
		maxIters:        MaxIterationsUntilAbort,
		pollingInterval: DefaultPollingInterval,
	}
}

// GetPerunAddr returns the perun address of the funder.
func (f *Funder) GetPerunAddr() xdr.ScAddress {
	return f.perunAddr
}

// GetAssetAddrs returns the asset addresses of the funder.
func (f *Funder) GetAssetAddrs() []xdr.ScVal {
	return f.assetAddrs
}

// Fund first calls open if the channel is not opened and then funds the channel.
func (f *Funder) Fund(ctx context.Context, req pchannel.FundingReq) error {
	log.Println("Fund called")

	if req.Idx != 0 && req.Idx != 1 {
		return errors.New("req.Idx must be 0 or 1")
	}

	if req.Idx == pchannel.Index(0) {
		err := f.openChannel(ctx, req)
		if err != nil {
			return err
		}
	}

	return f.fundParty(ctx, req)
}

//nolint:funlen,gocyclo
func (f *Funder) fundParty(ctx context.Context, req pchannel.FundingReq) error {
	party := getPartyByIndex(req.Idx)

	log.Printf("%s: Funding channel...", party)

	for i := 0; i < f.maxIters; i++ {
		select {
		case <-ctx.Done():
			timeoutErr := makeTimeoutErr([]pchannel.Index{req.Idx}, 0)
			errAbort := f.AbortChannel(ctx, req.State)
			log.Printf("%s: Aborting channel due to timeout...", party)
			if errAbort != nil {
				return errAbort
			}
			return timeoutErr

		case <-time.After(f.pollingInterval):

			log.Printf("%s: Polling for opened channel...", party)
			chanState, err := f.cb.GetChannelInfo(ctx, f.perunAddr, req.State.ID)
			if err != nil {
				log.Printf("%s: Error while polling for opened channel: %v", party, err)
				continue
			}

			log.Printf("%s: Found opened channel!", party)
			if chanState.Control.FundedA && chanState.Control.FundedB {
				return nil
			}

			if req.Idx == pchannel.Index(0) && !chanState.Control.FundedA { //nolint:nestif
				shouldFund := needFunding(req.State.Balances[0], req.State.Assets)
				if !shouldFund {
					log.Println("Party A does not need to fund")
					return nil
				}
				err := f.FundChannel(ctx, req.State, false)
				if err != nil {
					return err
				}
				bal0 := "bal0"
				bal1 := "bal1"
				t0, ok := req.State.Assets[0].(*types.StellarAsset)
				if ok {
					cAdd0, err := types.MakeContractAddress(t0.Asset.ContractID())
					if err != nil {
						return err
					}
					for {
						bal0, err = f.cb.GetBalance(cAdd0)
						if err != nil {
							log.Println("Error while getting balance: ", err)
						}
						if bal0 != "" {
							break
						}
						time.Sleep(1 * time.Second) // Wait for a second before retrying
					}
				}
				t1, ok := req.State.Assets[1].(*types.StellarAsset)
				if ok {
					cAdd1, err := types.MakeContractAddress(t1.Asset.ContractID())
					if err != nil {
						return err
					}
					bal1, err = f.cb.GetBalance(cAdd1)
					if err != nil {
						log.Println("Error while getting balance: ", err)
					}
				}
				log.Println("Balance A: ", bal0, bal1, " after funding amount: ", req.State.Balances, req.State.Assets)
				continue
			}
			//nolint:nestif
			if req.Idx == pchannel.Index(1) && !chanState.Control.FundedB && (chanState.Control.FundedA || !needFunding(req.State.Balances[0], req.State.Assets)) { // If party A has funded or does not need to fund, party B funds
				log.Println("Funding party B")
				shouldFund := needFunding(req.State.Balances[1], req.State.Assets)
				if !shouldFund {
					log.Println("Party B does not need to fund", req.State.Balances[1], req.State.Assets)
					return nil
				}
				err := f.FundChannel(ctx, req.State, true)
				if err != nil {
					return err
				}
				bal0 := "bal0"
				bal1 := "bal1"
				t0, ok := req.State.Assets[0].(*types.StellarAsset)
				if ok {
					cAdd0, err := types.MakeContractAddress(t0.Asset.ContractID())
					if err != nil {
						return err
					}
					bal0, err = f.cb.GetBalance(cAdd0)
					if err != nil {
						log.Println("Error while getting balance: ", err)
					}
				}
				t1, ok := req.State.Assets[1].(*types.StellarAsset)
				if ok {
					cAdd1, err := types.MakeContractAddress(t1.Asset.ContractID())
					if err != nil {
						return err
					}
					bal1, err = f.cb.GetBalance(cAdd1)
					if err != nil {
						log.Println("Error while getting balance: ", err)
					}
				}
				log.Println("Balance B: ", bal0, bal1, " after funding amount: ", req.State.Balances, req.State.Assets)
				continue
			}
		}
	}
	return f.AbortChannel(ctx, req.State)
}

func (f *Funder) openChannel(ctx context.Context, req pchannel.FundingReq) error {
	err := f.cb.Open(ctx, f.perunAddr, req.Params, req.State)
	if err != nil {
		return errors.Join(errors.New("error while opening channel in party A"), err)
	}
	_, err = f.cb.GetChannelInfo(ctx, f.perunAddr, req.State.ID)
	if err != nil {
		log.Println("Error while getting channel info: ", err)
		return err
	}
	return nil
}

// FundChannel funds the channel with the given state.
func (f *Funder) FundChannel(ctx context.Context, state *pchannel.State, funderIdx bool) error {
	balsStellar, err := wire.MakeBalances(state.Allocation)
	if err != nil {
		return errors.New("error while making balances")
	}

	if !containsAllAssets(balsStellar.Tokens, f.assetAddrs) {
		return errors.New("asset address is not equal to the address stored in the state")
	}

	return f.cb.Fund(ctx, f.perunAddr, state.ID, funderIdx)
}

// AbortChannel aborts the channel with the given state.
func (f *Funder) AbortChannel(ctx context.Context, state *pchannel.State) error {
	return f.cb.Abort(ctx, f.perunAddr, state)
}

func getPartyByIndex(funderIdx pchannel.Index) string {
	if funderIdx == 1 {
		return "Party B"
	}
	return "Party A"
}

// makeTimeoutErr returns a FundingTimeoutError for a specific Asset for a specific Funder.
func makeTimeoutErr(remains []pchannel.Index, assetIdx int) error {
	indices := make([]pchannel.Index, 0, len(remains))

	indices = append(indices, remains...)

	return pchannel.NewFundingTimeoutError(
		[]*pchannel.AssetFundingError{{
			Asset:         pchannel.Index(assetIdx),
			TimedOutPeers: indices,
		}},
	)
}

// Function to check if all assets in state.Allocation are present in f.assetAddrs.
func containsAllAssets(stateAssets []wire.Asset, fAssets []xdr.ScVal) bool {
	fAssetSet := assetSliceToSet(fAssets)

	for _, asset := range stateAssets {
		assetVal, err := asset.ToScVal()
		if err != nil {
			return false
		}
		if _, found := fAssetSet[assetVal.String()]; found { // if just one Asset was found, we continue
			return true
		}
	}

	return false
}

// Helper function to convert a slice of xdr.Asset to a set (map for fast lookup).
func assetSliceToSet(assets []xdr.ScVal) map[string]struct{} {
	assetSet := make(map[string]struct{})
	for _, asset := range assets {
		assetSet[asset.String()] = struct{}{}
	}
	return assetSet
}

// needFunding checks if a participant needs to fund the channel.
func needFunding(balances []pchannel.Bal, assets []pchannel.Asset) bool {
	for i, bal := range balances {
		_, ok := assets[i].(*types.StellarAsset)
		if bal.Cmp(big.NewInt(0)) != 0 && ok { // if balance is non 0 and asset is a stellar asset, participant needs to fund
			return true
		}
	}
	return false
}
