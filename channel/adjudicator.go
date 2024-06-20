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
	"context"
	"errors"
	"fmt"
	"github.com/stellar/go/xdr"
	pchannel "perun.network/go-perun/channel"
	"perun.network/go-perun/log"
	pwallet "perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/wallet"
	"time"
)

var ErrChannelAlreadyClosed = errors.New("channel is already closed")

var DefaultChallengeDuration = time.Duration(20) * time.Second

type Adjudicator struct {
	challengeDuration *time.Duration
	log               log.Embedding
	CB                *client.ContractBackend
	acc               *wallet.Account
	assetAddrs        xdr.ScVec //xdr.ScAddress
	perunAddr         xdr.ScAddress
	maxIters          int
	pollingInterval   time.Duration
}

// NewAdjudicator returns a new Adjudicator.
func NewAdjudicator(acc *wallet.Account, cb *client.ContractBackend, perunID xdr.ScAddress, assetIDs xdr.ScVec) *Adjudicator {
	return &Adjudicator{
		challengeDuration: &DefaultChallengeDuration,
		CB:                cb,
		acc:               acc,
		perunAddr:         perunID,
		assetAddrs:        assetIDs,
		maxIters:          MaxIterationsUntilAbort,
		pollingInterval:   DefaultPollingInterval,
		log:               log.MakeEmbedding(log.Default()),
	}
}

func (a *Adjudicator) GetPerunAddr() xdr.ScAddress {
	return a.perunAddr
}

func (a *Adjudicator) GetAssetAddrs() []xdr.ScAddress {
	var addrs []xdr.ScAddress
	for _, addrScVal := range a.assetAddrs {
		addr := addrScVal.MustAddress()
		addrs = append(addrs, addr)
	}

	return addrs
}

func (a *Adjudicator) Subscribe(ctx context.Context, cid pchannel.ID) (pchannel.AdjudicatorSubscription, error) {
	perunAddr := a.GetPerunAddr()
	assetAddrs := a.GetAssetAddrs()
	return NewAdjudicatorSub(ctx, cid, a.CB, perunAddr, assetAddrs, a.challengeDuration)
}

func (a *Adjudicator) Withdraw(ctx context.Context, req pchannel.AdjudicatorReq, smap pchannel.StateMap) error {

	if req.Tx.State.IsFinal {
		log.Println("Channel is finalized: ", req.Idx)
		log.Println("Withdraw called: ", req.Idx)

		if err := a.Close(ctx, req.Tx.State, req.Tx.Sigs); err != nil {
			log.Println("Close failed", err, req.Idx)
			chanControl, errChanState := a.CB.GetChannelInfo(ctx, a.perunAddr, req.Tx.State.ID)
			if errChanState != nil {
				log.Println("Error while getting channel info: ", req.Idx)
				return errChanState
			}

			if chanControl.Control.Closed {
				log.Println("Channel is already closed: ", req.Idx)
				return a.withdraw(ctx, req)
			}
			return err
		}

		return a.withdraw(ctx, req)

	} else {
		err := a.ForceClose(ctx, req.Tx.State, req.Tx.Sigs)
		log.Println("ForceClose called")
		if errors.Is(err, ErrChannelAlreadyClosed) {
			log.Println("Channel is already closed, force close")
			return a.withdraw(ctx, req)
		}
		if err != nil {
			return err
		}

		log.Println("Channel is not finalized, force close")
		err = a.withdraw(ctx, req)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Adjudicator) withdraw(ctx context.Context, req pchannel.AdjudicatorReq) error {
	perunAddress := a.GetPerunAddr()
	return a.CB.Withdraw(ctx, perunAddress, req)
}

func (a *Adjudicator) Close(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error {

	log.Println("Close called")
	perunAddr := a.GetPerunAddr()

	return a.CB.Close(ctx, perunAddr, state, sigs)
}

// Register registers and disputes a channel.
func (a *Adjudicator) Register(ctx context.Context, req pchannel.AdjudicatorReq, states []pchannel.SignedState) error {
	log.Println("Register called")
	err := a.Dispute(ctx, req.Tx.State, req.Tx.Sigs)
	if err != nil {
		return fmt.Errorf("error while disputing channel: %w", err)
	}
	return nil
}

func (a *Adjudicator) Dispute(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error {
	contractAddress := a.GetPerunAddr()
	return a.CB.Dispute(ctx, contractAddress, state, sigs)
}

func (a *Adjudicator) ForceClose(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error {
	return a.CB.ForceClose(ctx, a.perunAddr, state.ID)
}

func (a Adjudicator) Progress(ctx context.Context, req pchannel.ProgressReq) error {
	// only relevant for AppChannels
	return nil
}
