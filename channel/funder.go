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
	"github.com/stellar/go/xdr"
	"log"
	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wire"
	"time"
)

const MaxIterationsUntilAbort = 20
const DefaultPollingInterval = time.Duration(6) * time.Second

type Funder struct {
	stellarClient   *client.Client
	perunAddr       xdr.ScAddress
	assetAddr       xdr.ScAddress
	maxIters        int
	pollingInterval time.Duration
}

func NewFunder(acc *wallet.Account, stellarClient *client.Client, perunAddr xdr.ScAddress, assetAddr xdr.ScAddress) *Funder {
	return &Funder{
		stellarClient:   stellarClient,
		perunAddr:       perunAddr,
		assetAddr:       assetAddr,
		maxIters:        MaxIterationsUntilAbort,
		pollingInterval: DefaultPollingInterval,
	}
}

func (f *Funder) GetPerunAddr() xdr.ScAddress {
	return f.perunAddr
}

func (f *Funder) GetAssetAddr() xdr.ScAddress {
	return f.assetAddr
}

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
func (f *Funder) fundParty(ctx context.Context, req pchannel.FundingReq) error {

	party := getPartyByIndex(req.Idx)

	log.Printf("%s: Funding channel...\n", party)

	for i := 0; i < f.maxIters; i++ {
		select {
		case <-ctx.Done():
			return f.AbortChannel(ctx, req.State)
		case <-time.After(f.pollingInterval):

			log.Printf("%s: Polling for opened channel...\n", party)
			chanState, err := f.stellarClient.GetChannelInfo(ctx, f.perunAddr, req.State.ID)
			if err != nil {
				log.Printf("%s: Error while polling for opened channel: %v\n", party, err)
				continue
			}

			log.Printf("%s: Found opened channel!\n", party)
			if chanState.Control.FundedA && chanState.Control.FundedB {
				return nil
			}

			if req.Idx == pchannel.Index(0) && !chanState.Control.FundedA {
				err := f.FundChannel(ctx, req.State, false)
				if err != nil {
					return err
				}
				continue
			}
			if req.Idx == pchannel.Index(1) && !chanState.Control.FundedB {
				err := f.FundChannel(ctx, req.State, true)
				if err != nil {
					return err
				}
				continue
			}
		}
	}
	return f.AbortChannel(ctx, req.State)
}

func (f *Funder) openChannel(ctx context.Context, req pchannel.FundingReq) error {
	err := f.stellarClient.Open(ctx, f.perunAddr, req.Params, req.State)
	if err != nil {
		return errors.New("error while opening channel in party A")
	}
	return nil
}

func (f *Funder) FundChannel(ctx context.Context, state *pchannel.State, funderIdx bool) error {

	balsStellar, err := wire.MakeBalances(state.Allocation)
	if err != nil {
		return errors.New("error while making balances")
	}

	if !balsStellar.Token.Equals(f.assetAddr) {
		return errors.New("asset address is not equal to the address stored in the state")
	}

	return f.stellarClient.Fund(ctx, f.perunAddr, f.assetAddr, state.ID, funderIdx)
}

func (f *Funder) AbortChannel(ctx context.Context, state *pchannel.State) error {
	return f.stellarClient.Abort(ctx, f.perunAddr, state)
}

func getPartyByIndex(funderIdx pchannel.Index) string {
	if funderIdx == 1 {
		return "Party B"
	}
	return "Party A"
}
