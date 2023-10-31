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
	DefaultBufferSize                  = 3
	DefaultSubscriptionPollingInterval = time.Duration(4) * time.Second
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
	env           *env.IntegrationTestEnv
	queryChanArgs xdr.ScVec
	// current state of the channel
	//chanState    wire.Channel
	chanControl  wire.Control
	cid          pchannel.ID
	events       chan AdjEvent
	Ev           []AdjEvent
	err          error
	panicErr     chan error
	cancel       context.CancelFunc
	closer       *pkgsync.Closer
	pollInterval time.Duration
	log          log.Embedding
}

func (e *AdjEventSub) GetState() <-chan AdjEvent {
	return e.events
}

func NewAdjudicatorSub(ctx context.Context, cid pchannel.ID, stellarClient *env.StellarClient) *AdjEventSub {
	getChanArgs, err := env.BuildGetChannelTxArgs(cid)
	if err != nil {
		panic(err)
	}

	sub := &AdjEventSub{
		queryChanArgs: getChanArgs,
		//agent:        conn.PerunAgent,
		chanControl:  wire.Control{},
		events:       make(chan AdjEvent, DefaultBufferSize),
		Ev:           make([]AdjEvent, 0),
		panicErr:     make(chan error, 1),
		pollInterval: DefaultSubscriptionPollingInterval,
		closer:       new(pkgsync.Closer),
		log:          log.MakeEmbedding(log.Default()),
	}

	ctx, sub.cancel = context.WithCancel(ctx)
	go sub.run(ctx)
	return sub

}

func (s *AdjEventSub) run(ctx context.Context) {
	s.log.Log().Info("Listening for channel state changes")
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
		case <-s.events:
			finish(nil)
			return
		case <-time.After(s.pollInterval):

			newChanControl, err := s.getChanControl()

			if err != nil {
				// if query was not successful, simply repeat
				continue polling
			}
			// decode channel state difference to events
			adjEvent, err := DifferencesInControls(newChanControl, s.chanControl)
			if err != nil {
				s.panicErr <- err
			}

			if adjEvent == nil {
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
	//kpStellar := s.env.GetStellarClient().GetAccount()

	chanMeta, err := s.env.GetChannelState(getChanArgs)
	if err != nil {
		return wire.Control{}, err
	}

	retVal := chanMeta.V3.SorobanMeta.ReturnValue
	var chanState wire.Channel

	err = chanState.FromScVal(retVal)
	if err != nil {
		return wire.Control{}, err
	}

	chanControl := chanState.Control

	return chanControl, nil
}

func (s *AdjEventSub) chanStateToEvent(newState wire.Control) (AdjEvent, error) {
	// query channel state

	currControl := s.chanControl

	newCControl, err := s.getChanControl()
	if err != nil {
		return nil, err
	}

	sameChannel := IdenticalControls(currControl, newCControl)

	if sameChannel {
		return CloseEvent{}, nil
	}

	// state has changed: we evaluate the differences
	adjEvents, err := DifferencesInControls(currControl, newCControl)
	if err != nil {
		return nil, err
	}
	return adjEvents, nil

}

func DifferencesInControls(controlCurr, controlNext wire.Control) (AdjEvent, error) {

	if controlCurr.FundedA != controlNext.FundedA {
		if controlCurr.FundedA {
			return nil, errors.New("channel cannot be unfunded before withdrawal")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return FundEvent{}, nil
		}
	}

	if controlCurr.FundedB != controlNext.FundedB {
		if controlCurr.FundedB {
			return nil, errors.New("channel cannot be unfunded before withdrawal")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return FundEvent{}, nil
		}
	}

	if controlCurr.Closed != controlNext.Closed {
		if controlCurr.Closed {
			return nil, errors.New("channel cannot be reopened after closing")
		}
		return CloseEvent{}, nil
	}

	if controlCurr.WithdrawnA != controlNext.WithdrawnA {
		if controlCurr.WithdrawnA {
			return nil, errors.New("channel cannot be unwithdrawn")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return WithdrawnEvent{}, nil
		}
	}

	if controlCurr.WithdrawnB != controlNext.WithdrawnB {
		if controlCurr.WithdrawnB {
			return nil, errors.New("channel cannot be unwithdrawn")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return WithdrawnEvent{}, nil
		}
	}

	if controlCurr.Disputed != controlNext.Disputed {
		if controlCurr.Disputed {
			return nil, errors.New("channel cannot be undisputed")
		}
		return DisputedEvent{}, nil
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