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
	"log"
	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/event"
	"reflect"
	"time"
)

func (s *AdjEventSub) Next() pchannel.AdjudicatorEvent {
	if s.closer.IsClosed() {
		return nil
	}

	if s.getEvents() == nil {
		return nil
	}

	var challengeElapsed time.Time

	if s.challengeDuration != nil {
		challengeElapsed = time.Now().Add(*s.challengeDuration)
	}

	select {
	case ev := <-s.getEvents():
		if ev == nil {
			return nil
		}

		switch e := ev.(type) {
		case *event.DisputedEvent:
			log.Println("DisputedEvent received")
			dispEvent := pchannel.AdjudicatorEventBase{
				VersionV: e.Version(),
				IDV:      e.ID(),
				TimeoutV: event.NewTimeTimeout(challengeElapsed),
			}
			ddn := &pchannel.RegisteredEvent{AdjudicatorEventBase: dispEvent, State: nil, Sigs: nil}
			s.closer.Close()
			return ddn

		case *event.CloseEvent:

			log.Println("CloseEvent received")
			conclEvent := pchannel.AdjudicatorEventBase{
				VersionV: e.Version(),
				IDV:      e.ID(),
				TimeoutV: event.NewTimeTimeout(challengeElapsed),
			}
			ccn := &pchannel.ConcludedEvent{AdjudicatorEventBase: conclEvent}
			s.closer.Close()
			return ccn

		default:
			log.Printf("Received an unknown event type: %v\n", reflect.TypeOf(e))
			return nil
		}

	case <-s.closer.Closed():
		return nil
	}
}

func (s *AdjEventSub) Close() error {
	s.closer.Close()
	return nil
}

func (s *AdjEventSub) getEvents() <-chan event.PerunEvent {
	return s.events
}

func (s *AdjEventSub) Err() error {
	return s.err
}
