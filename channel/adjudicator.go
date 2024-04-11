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
	"perun.network/perun-stellar-backend/event"
	"perun.network/perun-stellar-backend/wallet"

	"time"
)

const DefaultEventBufferSize = 10

var ErrChannelAlreadyClosed = errors.New("channel is already closed")

var DefaultChallengeDuration = time.Duration(20) * time.Second

type Adjudicator struct {
	challengeDuration *time.Duration
	log               log.Embedding
	StellarClient     *client.Client
	acc               *wallet.Account
	assetAddr         xdr.ScAddress
	perunAddr         xdr.ScAddress
	maxIters          int
	pollingInterval   time.Duration
	eventSink         *StellarEventSink
}

// every stellarclient invocation with event output takes stellarevents and puts it into this sink. the setup happens during the definition of the
// adjudicator
// then, when subscribe happens, then this sink is read out by adjudicatoreventsubscription. this sink always takes the events,
// which are processed in the adjeventsub, and then output in the run() function, then appear in adjeventsub.Next(), which reads out the events and
// makes adjevent from the previous stellarevents.
type StellarEventSink struct {
	eventChan chan event.PerunEvent
	errChan   chan error
}

func NewStellarEventSink() *StellarEventSink {
	return &StellarEventSink{
		eventChan: make(chan event.PerunEvent, DefaultEventBufferSize),
		errChan:   make(chan error, 1),
	}
}

// func (s *StellarEventSub) run(context.Context) {
// 	for {
// 		select {
// 		case ev := <-s.eventChan:

// 		}
// 	}
// }

// NewAdjudicator returns a new Adjudicator.
func NewAdjudicator(acc *wallet.Account, stellarClient *client.Client, perunID xdr.ScAddress, assetID xdr.ScAddress) *Adjudicator {

	return &Adjudicator{
		challengeDuration: &DefaultChallengeDuration,
		StellarClient:     stellarClient,
		acc:               acc,
		perunAddr:         perunID,
		assetAddr:         assetID,
		maxIters:          MaxIterationsUntilAbort,
		pollingInterval:   DefaultPollingInterval,
		log:               log.MakeEmbedding(log.Default()),
		eventSink:         NewStellarEventSink(),
	}
}

func (a *Adjudicator) GetPerunAddr() xdr.ScAddress {
	return a.perunAddr
}

func (a *Adjudicator) GetAssetAddr() xdr.ScAddress {
	return a.assetAddr
}

func (a *Adjudicator) Subscribe(ctx context.Context, cid pchannel.ID) (pchannel.AdjudicatorSubscription, error) {
	c := a.StellarClient
	perunAddr := a.GetPerunAddr()
	assetAddr := a.GetAssetAddr()
	return NewAdjudicatorSub(ctx, cid, c, perunAddr, assetAddr, a.challengeDuration, a.eventSink)
}

func (a *Adjudicator) Withdraw(ctx context.Context, req pchannel.AdjudicatorReq, smap pchannel.StateMap) error {

	if req.Tx.State.IsFinal {
		log.Println("Withdraw called")

		if err := a.Close(ctx, req.Tx.State, req.Tx.Sigs); err != nil {
			chanControl, errChanState := a.StellarClient.GetChannelInfo(ctx, a.perunAddr, req.Tx.State.ID)
			if errChanState != nil {
				return errChanState
			}

			if chanControl.Control.Closed {
				return a.withdraw(ctx, req)
			}
			return err
		}

		return a.withdraw(ctx, req)

	} else {
		err := a.ForceClose(ctx, req.Tx.State, req.Tx.Sigs)
		log.Println("ForceClose called")
		if errors.Is(err, ErrChannelAlreadyClosed) {
			return a.withdraw(ctx, req)
		}
		if err != nil {
			return err
		}

		err = a.withdraw(ctx, req)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Adjudicator) withdraw(ctx context.Context, req pchannel.AdjudicatorReq) error {
	perunAddress := a.GetPerunAddr()
	return a.StellarClient.Withdraw(ctx, perunAddress, req)
}

func (a *Adjudicator) Close(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error {

	log.Println("Close called")
	perunAddr := a.GetPerunAddr()
	eventsAfterClose, err := a.StellarClient.Close(ctx, perunAddr, state, sigs)
	if err != nil {
		return err
	}

	if len(eventsAfterClose) > 0 {
		for _, ev := range eventsAfterClose {
			fmt.Println("event after close: ", ev)
			a.eventSink.eventChan <- ev
		}
	}

	return nil
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
	return a.StellarClient.Dispute(ctx, contractAddress, state, sigs)
}

func (a *Adjudicator) ForceClose(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error {
	return a.StellarClient.ForceClose(ctx, a.perunAddr, state.ID)
}

func (a Adjudicator) Progress(ctx context.Context, req pchannel.ProgressReq) error {
	// only relevant for AppChannels
	return nil
}
