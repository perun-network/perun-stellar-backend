// Copyright 2023 PolyCrypt GmbH
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
	"fmt"
	"github.com/stellar/go/xdr"
	"log"
	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wire"
	"time"
)

const MaxIterationsUntilAbort = 20
const DefaultPollingInterval = time.Duration(6) * time.Second

type Funder struct {
	cb              *client.ContractBackend
	perunAddr       xdr.ScAddress
	assetAddrs      xdr.ScVec
	maxIters        int
	pollingInterval time.Duration
}

func NewFunder(acc *wallet.Account, contractBackend *client.ContractBackend, perunAddr xdr.ScAddress, assetAddrs xdr.ScVec) *Funder {
	return &Funder{
		cb:              contractBackend,
		perunAddr:       perunAddr,
		assetAddrs:      assetAddrs,
		maxIters:        MaxIterationsUntilAbort,
		pollingInterval: DefaultPollingInterval,
	}
}

func (f *Funder) GetPerunAddr() xdr.ScAddress {
	return f.perunAddr
}

func (f *Funder) GetAssetAddrs() []xdr.ScAddress {
	var addrs []xdr.ScAddress
	for _, addrScVal := range f.assetAddrs {
		addr := addrScVal.MustAddress()
		addrs = append(addrs, addr)
	}
	return addrs
}

func (f *Funder) Fund(ctx context.Context, req pchannel.FundingReq) error {
	log.Println("Fund called")

	if req.Idx != 0 && req.Idx != 1 {
		return errors.New("req.Idx must be 0 or 1")
	}

	if req.Idx == pchannel.Index(0) {
		log.Println("open channel")
		err := f.openChannel(ctx, req)
		if err != nil {
			return err
		}
	}
	log.Println("fund party")
	return f.fundParty(ctx, req)
}
func (f *Funder) fundParty(ctx context.Context, req pchannel.FundingReq) error {

	party := getPartyByIndex(req.Idx)

	log.Printf("%s: Funding channel...\n", party)

	for i := 0; i < f.maxIters; i++ {
		log.Println("funding loop, iteration: ", i)
		select {
		case <-ctx.Done():
			timeoutErr := makeTimeoutErr([]pchannel.Index{req.Idx}, 0)
			errAbort := f.AbortChannel(ctx, req.State)
			log.Printf("%s: Aborting channel due to timeout...\n", party)
			if errAbort != nil {
				return errAbort
			}
			return timeoutErr

		case <-time.After(f.pollingInterval):

			log.Printf("%s: Polling for opened channel...\n", party)
			chanState, err := f.cb.GetChannelInfo(ctx, f.perunAddr, req.State.ID)
			if err != nil {
				log.Printf("%s: Error while polling for opened channel: %v\n", party, err)
				continue
			}

			log.Printf("%s: Found opened channel!\n", party)
			if chanState.Control.FundedA && chanState.Control.FundedB {
				return nil
			}

			if req.Idx == pchannel.Index(0) && !chanState.Control.FundedA {
				log.Println("Funding party A")
				err := f.FundChannel(ctx, req.State, false)
				if err != nil {
					return err
				}
				t0, ok := req.State.Assets[0].(*types.StellarAsset)
				if !ok {
					return fmt.Errorf("expected StellarAsset at index 0, got %T", req.State.Assets[0])
				}
				cAdd0, err := types.MakeContractAddress(t0.ContractID())
				if err != nil {
					return err
				}
				var bal0 string
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
				t1, ok := req.State.Assets[1].(*types.StellarAsset)
				if !ok {
					return fmt.Errorf("expected StellarAsset at index 1, got %T", req.State.Assets[1])
				}
				cAdd1, err := types.MakeContractAddress(t1.ContractID())
				if err != nil {
					return err
				}
				bal1, err := f.cb.GetBalance(cAdd1)
				if err != nil {
					log.Println("Error while getting balance: ", err)
				}
				log.Println("Balance A: ", bal0, bal1, " after funding amount: ", req.State.Balances, req.State.Assets)
				continue
			}
			if req.Idx == pchannel.Index(1) && !chanState.Control.FundedB && chanState.Control.FundedA {
				log.Println("Funding party B")
				err := f.FundChannel(ctx, req.State, true)
				if err != nil {
					return err
				}
				t0, ok := req.State.Assets[0].(*types.StellarAsset)
				if !ok {
					return fmt.Errorf("expected StellarAsset at index 0, got %T", req.State.Assets[0])
				}
				cAdd0, err := types.MakeContractAddress(t0.ContractID())
				if err != nil {
					return err
				}
				bal0, err := f.cb.GetBalance(cAdd0)
				if err != nil {
					log.Println("Error while getting balance: ", err)
				}
				t1, ok := req.State.Assets[1].(*types.StellarAsset)
				if !ok {
					return fmt.Errorf("expected StellarAsset at index 1, got %T", req.State.Assets[1])
				}
				cAdd1, err := types.MakeContractAddress(t1.ContractID())
				if err != nil {
					return err
				}
				bal1, err := f.cb.GetBalance(cAdd1)
				if err != nil {
					log.Println("Error while getting balance: ", err)
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
		log.Println(err)
		return errors.New("error while opening channel in party A")
	}
	_, err = f.cb.GetChannelInfo(ctx, f.perunAddr, req.State.ID)
	return nil
}

func (f *Funder) FundChannel(ctx context.Context, state *pchannel.State, funderIdx bool) error {
	balsStellar, err := wire.MakeBalances(state.Allocation)
	log.Println("Funding channel with balances: ", balsStellar)
	if err != nil {
		return errors.New("error while making balances")
	}

	if !containsAllAssets(balsStellar.Tokens, f.assetAddrs) {
		return errors.New("asset address is not equal to the address stored in the state")
	}
	return f.cb.Fund(ctx, f.perunAddr, state.ID, funderIdx)
}

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

// Function to check if all assets in state.Allocation are present in f.assetAddrs
func containsAllAssets(stateAssets xdr.ScVec, fAssets xdr.ScVec) bool {
	fAssetSet := assetSliceToSet(fAssets)

	for _, asset := range stateAssets {
		if _, found := fAssetSet[asset.String()]; !found {
			return false
		}
	}

	return true
}

// Helper function to convert a slice of xdr.Asset to a set (map for fast lookup)
func assetSliceToSet(assets xdr.ScVec) map[string]struct{} {
	assetSet := make(map[string]struct{})
	for _, asset := range assets {
		assetSet[asset.String()] = struct{}{}
	}
	return assetSet
}
