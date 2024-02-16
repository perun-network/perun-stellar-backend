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
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wire"
	"time"
)

const MaxIterationsUntilAbort = 20
const DefaultPollingInterval = time.Duration(6) * time.Second

type Funder struct {
	stellarClient   *env.StellarClient
	perunID         xdr.ScAddress
	assetID         xdr.ScAddress
	maxIters        int
	pollingInterval time.Duration
}

func NewFunder(acc *wallet.Account, stellarClient *env.StellarClient, perunID xdr.ScAddress, assetID xdr.ScAddress) *Funder {
	return &Funder{
		stellarClient:   stellarClient,
		perunID:         perunID,
		assetID:         assetID,
		maxIters:        MaxIterationsUntilAbort,
		pollingInterval: DefaultPollingInterval,
	}
}

func (f *Funder) GetPerunID() xdr.ScAddress {
	return f.perunID
}

func (f *Funder) GetAssetID() xdr.ScAddress {
	return f.assetID
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
			return f.AbortChannel(ctx, req.Params, req.State)
		case <-time.After(f.pollingInterval):

			log.Printf("%s: Polling for opened channel...\n", party)
			chanState, err := f.GetChannelState(ctx, req.Params, req.State)
			if err != nil {
				log.Printf("%s: Error while polling for opened channel: %v\n", party, err)
				continue
			}

			log.Printf("%s: Found opened channel!\n", party)
			if chanState.Control.FundedA && chanState.Control.FundedB {
				return nil
			}

			if req.Idx == pchannel.Index(0) && !chanState.Control.FundedA {
				err := f.FundChannel(ctx, req.Params, req.State, false)
				if err != nil {
					return err
				}
				continue
			}
			if req.Idx == pchannel.Index(1) && !chanState.Control.FundedB {
				err := f.FundChannel(ctx, req.Params, req.State, true)
				if err != nil {
					return err
				}
				continue
			}
		}
	}
	return f.AbortChannel(ctx, req.Params, req.State)
}

func (f *Funder) openChannel(ctx context.Context, req pchannel.FundingReq) error {
	err := f.OpenChannel(ctx, req.Params, req.State)
	if err != nil {
		return errors.New("error while opening channel in party A")
	}
	return nil
}

func (f *Funder) OpenChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State) error {

	perunAddress := f.GetPerunID()
	openTxArgs, err := env.BuildOpenTxArgs(params, state)
	if err != nil {
		return errors.New("error while building open tx")
	}
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction("open", openTxArgs, perunAddress)
	if err != nil {
		return errors.New("error while invoking and processing host function: open")
	}

	_, err = DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (f *Funder) FundChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State, funderIdx bool) error {

	perunAddress := f.GetPerunID()
	tokenAddress := f.GetAssetID()

	chanId := state.ID

	fundTxArgs, err := env.BuildFundTxArgs(chanId, funderIdx)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	balsStellar, err := wire.MakeBalances(state.Allocation)
	if err != nil {
		return errors.New("error while making balances")
	}

	tokenIDAddrFromBals := balsStellar.Token

	sameContractTokenID := tokenIDAddrFromBals.Equals(tokenAddress)
	if !sameContractTokenID {
		return errors.New("tokenIDAddrFromBals not equal to tokenContractAddress")
	}

	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction("fund", fundTxArgs, perunAddress)
	if err != nil {
		return errors.New("error while invoking and processing host function: fund")
	}

	_, err = DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (f *Funder) AbortChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State) error {

	contractAddress := f.GetPerunID()
	chanId := state.ID

	openTxArgs, err := env.BuildGetChannelTxArgs(chanId)
	if err != nil {
		return errors.New("error while building get_channel tx")
	}
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction("abort_funding", openTxArgs, contractAddress)
	if err != nil {
		return errors.New("error while invoking and processing host function: abort_funding")
	}

	_, err = DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (f *Funder) GetChannelState(ctx context.Context, params *pchannel.Params, state *pchannel.State) (wire.Channel, error) {

	contractAddress := f.GetPerunID()
	chanId := state.ID

	getchTxArgs, err := env.BuildGetChannelTxArgs(chanId)
	if err != nil {
		return wire.Channel{}, errors.New("error while building get_channel tx")
	}
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction("get_channel", getchTxArgs, contractAddress)
	if err != nil {
		return wire.Channel{}, errors.New("error while processing and submitting get_channel tx")
	}

	retVal := txMeta.V3.SorobanMeta.ReturnValue
	var getChan wire.Channel

	err = getChan.FromScVal(retVal)
	if err != nil {
		return wire.Channel{}, errors.New("error while decoding return value")
	}
	return getChan, nil
}

func getPartyByIndex(funderIdx pchannel.Index) string {
	if funderIdx == 1 {
		return "Party B"
	}
	return "Party A"
}
