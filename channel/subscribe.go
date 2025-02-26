// Copyright 2025 PolyCrypt GmbH
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
	"reflect"

	pchannel "perun.network/go-perun/channel"

	"perun.network/perun-stellar-backend/event"
)

func (s *AdjEventSub) Next() pchannel.AdjudicatorEvent {
	if s.closer.IsClosed() {
		return nil
	}

	if s.getEvents() == nil {
		return nil
	}
	select {
	case ev := <-s.getEvents():
		if ev == nil {
			return nil
		}

		switch e := ev.(type) {
		case *event.DisputedEvent:
			log.Println("DisputedEvent received - build RegisteredEvent")
			dispEvent := pchannel.AdjudicatorEventBase{
				VersionV: e.Version(),
				IDV:      e.ID(),
				TimeoutV: event.MakeTimeout(*s.challengeDuration),
			}
			adjDispEvent := &pchannel.RegisteredEvent{AdjudicatorEventBase: dispEvent, State: nil, Sigs: nil}
			return adjDispEvent

		case *event.CloseEvent:

			log.Println("CloseEvent received - build ConcludedEvent, ", e.ID())

			conclEvent := pchannel.AdjudicatorEventBase{
				VersionV: e.Version(),
				IDV:      e.ID(),
				TimeoutV: event.MakeTimeout(*s.challengeDuration),
			}
			adjConclEvent := &pchannel.ConcludedEvent{AdjudicatorEventBase: conclEvent}
			return adjConclEvent

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
