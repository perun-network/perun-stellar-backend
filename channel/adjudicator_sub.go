package channel

import (
	"context"
	"errors"
	"fmt"
	"github.com/stellar/go/xdr"
	pchannel "perun.network/go-perun/channel"
	log "perun.network/go-perun/log"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/wire"
	pkgsync "polycry.pt/poly-go/sync"
	"time"
)

const (
	DefaultBufferSize                  = 3
	DefaultSubscriptionPollingInterval = time.Duration(10) * time.Second
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
	events        chan AdjEvent
	Ev            []AdjEvent
	err           error
	panicErr      chan error
	cancel        context.CancelFunc
	closer        *pkgsync.Closer
	pollInterval  time.Duration
	log           log.Embedding
}

func NewAdjudicatorSub(ctx context.Context, cid pchannel.ID, stellarClient *env.StellarClient) *AdjEventSub {
	getChanArgs, err := env.BuildGetChannelTxArgs(cid)
	if err != nil {
		panic(err)
	}

	sub := &AdjEventSub{
		queryChanArgs: getChanArgs,
		stellarClient: stellarClient,
		chanControl:   wire.Control{},
		cid:           cid,
		events:        make(chan AdjEvent, DefaultBufferSize),
		Ev:            make([]AdjEvent, 0),
		panicErr:      make(chan error, 1),
		pollInterval:  DefaultSubscriptionPollingInterval,
		closer:        new(pkgsync.Closer),
		log:           log.MakeEmbedding(log.Default()),
	}

	ctx, sub.cancel = context.WithCancel(ctx)
	go sub.run(ctx)
	return sub

}

func (s *AdjEventSub) run(ctx context.Context) {
	s.log.Log().Info("Listening for channel state changes")
	chanControl, err := s.getChanControl()
	if err != nil {
		s.panicErr <- err
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
		case <-ctx.Done():
			finish(nil)
			return
		case <-time.After(s.pollInterval):

			newChanControl, err := s.getChanControl()

			if err != nil {

				s.panicErr <- err
			}
			// decode channel state difference to events
			adjEvent, err := DifferencesInControls(s.chanControl, newChanControl)
			if err != nil {
				s.panicErr <- err
			}

			if adjEvent == nil {
				s.chanControl = newChanControl
				s.log.Log().Debug("No events yet, continuing polling...")
				continue polling

			} else {
				s.log.Log().Debug("Event detected, evaluating events...")

				// Store the event

				s.log.Log().Debugf("Found event: %v", adjEvent)
				s.events <- adjEvent
				return
			}
		}
	}
}

func (s *AdjEventSub) getChanControl() (wire.Control, error) {
	// query channel state

	getChanArgs, err := env.BuildGetChannelTxArgs(s.cid)
	if err != nil {
		return wire.Control{}, err
	}

	chanParams, err := s.stellarClient.GetChannelState(getChanArgs)
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
			fmt.Println("controlCurr.Closed need to insert data: ", controlCurr.Closed)

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
