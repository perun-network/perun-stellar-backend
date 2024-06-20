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
	"github.com/stellar/go/xdr"
	pchannel "perun.network/go-perun/channel"
	log "perun.network/go-perun/log"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/event"
	"perun.network/perun-stellar-backend/wire"
	pkgsync "polycry.pt/poly-go/sync"
	"time"
)

const (
	DefaultBufferSize                  = 1024
	DefaultSubscriptionPollingInterval = time.Duration(15) * time.Second
)

type AdjEventSub struct {
	challengeDuration *time.Duration
	cb                *client.ContractBackend
	chanControl       wire.Control
	cid               pchannel.ID
	perunAddr         xdr.ScAddress
	assetAddrs        []xdr.ScAddress
	events            chan event.PerunEvent
	subErrors         chan error
	err               error
	cancel            context.CancelFunc
	closer            *pkgsync.Closer
	pollInterval      time.Duration
	log               log.Embedding
}

func NewAdjudicatorSub(ctx context.Context, cid pchannel.ID, cb *client.ContractBackend, perunAddr xdr.ScAddress, assetAddrs []xdr.ScAddress, challengeDuration *time.Duration) (pchannel.AdjudicatorSubscription, error) {

	sub := &AdjEventSub{
		challengeDuration: challengeDuration,
		cb:                cb,
		chanControl:       wire.Control{},
		cid:               cid,
		perunAddr:         perunAddr,
		assetAddrs:        assetAddrs,
		events:            make(chan event.PerunEvent, DefaultBufferSize),
		subErrors:         make(chan error, 1),
		pollInterval:      DefaultSubscriptionPollingInterval,
		closer:            new(pkgsync.Closer),
		log:               log.MakeEmbedding(log.Default()),
	}

	ctx, sub.cancel = context.WithCancel(ctx)
	go sub.run(ctx)
	return sub, nil

}

func (s *AdjEventSub) run(ctx context.Context) {
	s.log.Log().Info("Listening for channel state changes")
	chanControl, err := s.cb.GetChannelInfo(ctx, s.perunAddr, s.cid)
	if err != nil {
		s.subErrors <- err
	}

	s.chanControl = chanControl.Control
	finish := func(err error) {
		s.err = err
		close(s.events)
	}
	var newChanControl wire.Control
polling:
	for {
		s.log.Log().Debug("AdjudicatorSub is listening for Adjudicator Events")
		select {
		case err := <-s.subErrors:
			finish(err)
			return
		case <-ctx.Done():
			s.log.Log().Debug("Timeout during Adjudicator Subscription")
			finish(nil)
			return
		case <-time.After(s.pollInterval):
			log.Println("Polling for contract events...", s.cid)
			newChanInfo, err := s.cb.GetChannelInfo(ctx, s.perunAddr, s.cid)
			newChanControl = newChanInfo.Control

			if err != nil {
				s.subErrors <- err
			}
			adjEvent, err := DifferencesInControls(s.chanControl, newChanControl)
			if err != nil {
				s.subErrors <- err
			}

			if adjEvent == nil {
				s.chanControl = newChanControl
				s.log.Log().Debug("No events yet, continuing polling...")
				continue polling

			} else {
				s.log.Log().Debug("Contract event detected, evaluating...")
				s.log.Log().Debugf("Found contract event: %v", adjEvent)
				adjEvent.SetID(s.cid)
				s.events <- adjEvent
				etype, _ := adjEvent.GetType()
				if etype == event.EventTypeWithdrawn {
					return
				}
			}
		}
	}
}

func DifferencesInControls(controlCurr, controlNext wire.Control) (event.PerunEvent, error) {

	if controlCurr.FundedA != controlNext.FundedA {
		if controlCurr.FundedA {
			return nil, errors.New("channel cannot be unfunded A before withdrawal")
		}
	}

	if controlCurr.FundedB != controlNext.FundedB {
		if controlCurr.FundedB {
			return nil, errors.New("channel cannot be unfunded B before withdrawal")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return &event.FundEvent{}, nil
		}
	}

	if controlCurr.Closed != controlNext.Closed {
		if controlCurr.Closed {
			return nil, errors.New("channel cannot be reopened after closing")
		}
		if !controlCurr.Closed && controlNext.Closed {
			return &event.CloseEvent{}, nil
		}
		return &event.CloseEvent{}, nil
	}

	if controlCurr.Closed && controlNext.Closed {
		return &event.CloseEvent{}, nil
	}

	if controlCurr.WithdrawnA != controlNext.WithdrawnA {
		if controlCurr.WithdrawnA {
			return nil, errors.New("channel cannot be unwithdrawn")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return &event.WithdrawnEvent{}, nil
		}
	}

	if controlCurr.WithdrawnB != controlNext.WithdrawnB {
		if controlCurr.WithdrawnB {
			return nil, errors.New("channel cannot be unwithdrawn")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return &event.WithdrawnEvent{}, nil
		}
	}

	if controlCurr.Disputed != controlNext.Disputed {
		if controlCurr.Disputed {
			return nil, errors.New("channel cannot be undisputed")
		}
		return &event.DisputedEvent{}, nil
	}

	return nil, nil
}
