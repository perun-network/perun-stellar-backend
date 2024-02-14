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
	pchannel "perun.network/go-perun/channel"
	log "perun.network/go-perun/log"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/wire"
	pkgsync "polycry.pt/poly-go/sync"
	"time"
)

const (
	DefaultBufferSize                  = 1024
	DefaultSubscriptionPollingInterval = time.Duration(5) * time.Second
)

type AdjEvent interface {
	// Sets the necessary event data from the channel information
	EventDataFromChannel(cchanState wire.Channel, timestamp uint64) error
	ID() pchannel.ID
	Timeout() pchannel.Timeout
	Version() Version
	Tstamp() uint64
}

// AdjudicatorSub implements the AdjudicatorSubscription interface.
type AdjEventSub struct {
	queryChanArgs xdr.ScVec
	stellarClient *env.StellarClient
	chanControl   wire.Control
	cid           pchannel.ID
	perunID       xdr.ScAddress
	assetID       xdr.ScAddress
	events        chan AdjEvent
	subErrors     chan error
	err           error
	cancel        context.CancelFunc
	closer        *pkgsync.Closer
	pollInterval  time.Duration
	log           log.Embedding
}

func NewAdjudicatorSub(ctx context.Context, cid pchannel.ID, stellarClient *env.StellarClient, perunID xdr.ScAddress, assetID xdr.ScAddress) (*AdjEventSub, error) {
	getChanArgs, err := env.BuildGetChannelTxArgs(cid)
	if err != nil {
		return nil, err
	}

	sub := &AdjEventSub{
		queryChanArgs: getChanArgs,
		stellarClient: stellarClient,
		chanControl:   wire.Control{},
		cid:           cid,
		perunID:       perunID,
		assetID:       assetID,
		events:        make(chan AdjEvent, DefaultBufferSize),
		subErrors:     make(chan error, 1),
		pollInterval:  DefaultSubscriptionPollingInterval,
		closer:        new(pkgsync.Closer),
		log:           log.MakeEmbedding(log.Default()),
	}

	ctx, sub.cancel = context.WithCancel(ctx)
	go sub.run(ctx)
	return sub, nil

}

func (s *AdjEventSub) run(ctx context.Context) {
	s.log.Log().Info("Listening for channel state changes")
	chanControl, err := s.getChanControl()
	if err != nil {
		s.subErrors <- err
	}
	s.chanControl = chanControl
	finish := func(err error) {
		s.err = err
		close(s.events)
	}
polling:
	for {
		s.log.Log().Debug("AdjudicatorSub is listening for Adjudicator Events")
		select {
		case err := <-s.subErrors:
			finish(err)
			return
		case <-ctx.Done():
			finish(nil)
			return
		case <-time.After(s.pollInterval):

			newChanControl, err := s.getChanControl()

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
				s.log.Log().Debug("Event detected, evaluating events...")
				s.log.Log().Debugf("Found event: %v", adjEvent)
				s.events <- adjEvent
				return
			}
		}
	}
}

func (s *AdjEventSub) GetChannelState(chanArgs xdr.ScVec) (wire.Channel, error) {
	contractAddress := s.perunID
	kp := s.stellarClient.GetKeyPair()
	txMeta, err := s.stellarClient.InvokeAndProcessHostFunction("get_channel", chanArgs, contractAddress, kp)
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

func (s *AdjEventSub) getChanControl() (wire.Control, error) {
	// query channel state

	getChanArgs, err := env.BuildGetChannelTxArgs(s.cid)
	if err != nil {
		return wire.Control{}, err
	}

	chanParams, err := s.GetChannelState(getChanArgs)
	if err != nil {
		return wire.Control{}, err
	}
	chanControl := chanParams.Control

	return chanControl, nil
}

func DifferencesInControls(controlCurr, controlNext wire.Control) (AdjEvent, error) {

	if controlCurr.FundedA != controlNext.FundedA {
		if controlCurr.FundedA {
			return nil, errors.New("channel cannot be unfunded A before withdrawal")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return &FundEvent{}, nil
		}
	}

	if controlCurr.FundedB != controlNext.FundedB {
		if controlCurr.FundedB {
			return nil, errors.New("channel cannot be unfunded B before withdrawal")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return &FundEvent{}, nil
		}
	}

	if controlCurr.Closed != controlNext.Closed {
		if controlCurr.Closed {
			return nil, errors.New("channel cannot be reopened after closing")
		}
		if !controlCurr.Closed && controlNext.Closed {
			return &CloseEvent{}, nil
		}
		return &CloseEvent{}, nil
	}

	if controlCurr.WithdrawnA != controlNext.WithdrawnA {
		if controlCurr.WithdrawnA {
			return nil, errors.New("channel cannot be unwithdrawn")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return &WithdrawnEvent{}, nil
		}
	}

	if controlCurr.WithdrawnB != controlNext.WithdrawnB {
		if controlCurr.WithdrawnB {
			return nil, errors.New("channel cannot be unwithdrawn")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return &WithdrawnEvent{}, nil
		}
	}

	if controlCurr.Disputed != controlNext.Disputed {
		if controlCurr.Disputed {
			return nil, errors.New("channel cannot be undisputed")
		}
		return &DisputedEvent{}, nil
	}

	return nil, nil
}

func IdenticalControls(controlCurr, controlNext wire.Control) bool {
	return controlCurr.FundedA == controlNext.FundedA &&
		controlCurr.FundedB == controlNext.FundedB &&
		controlCurr.Closed == controlNext.Closed &&
		controlCurr.WithdrawnA == controlNext.WithdrawnA &&
		controlCurr.WithdrawnB == controlNext.WithdrawnB &&
		controlCurr.Disputed == controlNext.Disputed
}
