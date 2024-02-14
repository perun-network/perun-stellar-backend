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
	"github.com/stellar/go/keypair"
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
	acc             *wallet.Account
	kpFull          *keypair.Full
	perunID         xdr.ScAddress
	assetID         xdr.ScAddress
	maxIters        int
	pollingInterval time.Duration
}

func NewFunder(acc *wallet.Account, kp *keypair.Full, stellarClient *env.StellarClient, perunID xdr.ScAddress, assetID xdr.ScAddress) *Funder {
	return &Funder{
		stellarClient:   stellarClient,
		acc:             acc,
		kpFull:          kp,
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
	switch req.Idx {
	case 0:
		return f.fundPartyA(ctx, req)
	case 1:
		return f.fundPartyB(ctx, req)
	default:
		return errors.New("invalid index")
	}
}

func (f *Funder) fundPartyA(ctx context.Context, req pchannel.FundingReq) error {

	err := f.OpenChannel(ctx, req.Params, req.State)
	if err != nil {

		return errors.New("error while opening channel in party A")
	}
	err = f.FundChannel(ctx, req.Params, req.State, false)
	if err != nil {
		return err
	}

polling:
	for i := 0; i < f.maxIters; i++ {
		select {
		case <-ctx.Done():
			return f.AbortChannel(ctx, req.Params, req.State)
		case <-time.After(f.pollingInterval):
			chanState, err := f.GetChannelState(ctx, req.Params, req.State)
			if err != nil {
				continue polling
			}
			if chanState.Control.FundedA && chanState.Control.FundedB {
				return nil
			}

		}
	}
	return f.AbortChannel(ctx, req.Params, req.State)
}

func (f *Funder) fundPartyB(ctx context.Context, req pchannel.FundingReq) error {

polling:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(f.pollingInterval):
			log.Println("Party B: Polling for opened channel...")
			chanState, err := f.GetChannelState(ctx, req.Params, req.State)
			if err != nil {
				log.Println("Party B: Error while polling for opened channel:", err)
				continue polling
			}
			log.Println("Party B: Found opened channel!")
			if chanState.Control.FundedA && chanState.Control.FundedB {
				return nil
			}
			return f.FundChannel(ctx, req.Params, req.State, true)
		}
	}
}

func (f *Funder) OpenChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State) error {

	perunAddress := f.GetPerunID()
	kp := f.kpFull

	openTxArgs, err := env.BuildOpenTxArgs(params, state)
	if err != nil {
		return errors.New("error while building open tx")
	}
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction("open", openTxArgs, perunAddress, kp)
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

	kp := f.kpFull
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

	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction("fund", fundTxArgs, perunAddress, kp)
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
	kp := f.kpFull
	chanId := state.ID

	openTxArgs, err := env.BuildGetChannelTxArgs(chanId)
	if err != nil {
		return errors.New("error while building get_channel tx")
	}
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction("abort_funding", openTxArgs, contractAddress, kp)
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
	kp := f.kpFull
	chanId := state.ID

	getchTxArgs, err := env.BuildGetChannelTxArgs(chanId)
	if err != nil {
		return wire.Channel{}, errors.New("error while building get_channel tx")
	}
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction("get_channel", getchTxArgs, contractAddress, kp)
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
