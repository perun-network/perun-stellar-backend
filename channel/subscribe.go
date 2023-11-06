package channel

import (
	"fmt"
	"reflect"

	pchannel "perun.network/go-perun/channel"
)

// Next implements the AdjudicatorSub.Next function.
func (s *AdjEventSub) Next() pchannel.AdjudicatorEvent {
	if s.closer.IsClosed() {
		return nil
	}

	if s.Events() == nil {
		return nil
	}

	select {
	case event := <-s.Events():
		fmt.Println("Event received: ", event)
		if event == nil {
			fmt.Println("Event nil received: ", event)
			return nil
		}

		timestamp := event.Tstamp()

		switch e := event.(type) {
		case *DisputedEvent:
			fmt.Println("DisputedEvent received")
			dispEvent := pchannel.AdjudicatorEventBase{
				VersionV: e.Version(),
				IDV:      e.ID(),
				TimeoutV: MakeTimeout(timestamp),
			}
			ddn := &pchannel.RegisteredEvent{AdjudicatorEventBase: dispEvent, State: nil, Sigs: nil}
			s.closer.Close()
			return ddn

		case *CloseEvent:

			fmt.Println("CloseEvent received")
			conclEvent := pchannel.AdjudicatorEventBase{
				VersionV: e.Version(),
				IDV:      e.ID(),
				TimeoutV: MakeTimeout(timestamp),
			}
			ccn := &pchannel.ConcludedEvent{AdjudicatorEventBase: conclEvent}
			s.closer.Close()
			return ccn

		default:
			fmt.Printf("Received an unknown event type: %v\n", reflect.TypeOf(e))
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

func (s *AdjEventSub) Events() <-chan AdjEvent {
	return s.events
}

func (s *AdjEventSub) Err() error {
	return s.err
}

func (s *AdjEventSub) PanicErr() <-chan error {
	return s.panicErr
}
