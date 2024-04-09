// Copyright 2024 PolyCrypt GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package event

import (
	"errors"
	"fmt"
	"github.com/stellar/go/xdr"
	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/wire"
)

type Version = uint64
type Event = xdr.ContractEvent
type EventType int

const (
	EventTypeOpen          EventType = iota
	EventTypeFundChannel             // participant/s funding channel
	EventTypeFundedChannel           // participants have funded channel
	EventTypeClosed                  // channel closed -> withdrawing enabled
	EventTypeWithdrawing             // participant/s withdrawing
	EventTypeWithdrawn               // participants have withdrawn
	EventTypeForceClose              // participant has force closed the channel
	EventTypeDisputed                // participant has disputed the channel
	EventTypeError                   // inconsistent event
)

const AssertPerunSymbol = "perun"

var (
	STELLAR_PERUN_CHANNEL_CONTRACT_TOPICS = map[xdr.ScSymbol]EventType{
		xdr.ScSymbol("open"):     EventTypeOpen,
		xdr.ScSymbol("fund"):     EventTypeFundChannel,
		xdr.ScSymbol("fund_c"):   EventTypeFundedChannel,
		xdr.ScSymbol("closed"):   EventTypeClosed,
		xdr.ScSymbol("withdraw"): EventTypeWithdrawing,
		xdr.ScSymbol("pay_c"):    EventTypeWithdrawn,
		xdr.ScSymbol("f_closed"): EventTypeForceClose,
		xdr.ScSymbol("dispute"):  EventTypeDisputed,
	}

	ErrNotStellarPerunContract = errors.New("event was not from a Perun payment channel contract")
	ErrEventUnsupported        = errors.New("this type of event is unsupported")
	ErrEventIntegrity          = errors.New("contract ID does not match payment channel + passphrase")
	ErrEventDecode             = errors.New("error while decoding events")
	ErrNoFundEvent             = errors.New("fund event not found")
	ErrNoCloseEvent            = errors.New("close event not found")
	ErrNoWithdrawEvent         = errors.New("withdraw event not found")
	ErrNoDisputeEvent          = errors.New("dispute event not found")
	ErrNoForceCloseEvent       = errors.New("force close event not found")
)

type controlsState map[string]bool

type (
	PerunEvent interface {
		GetID() pchannel.ID
		GetChannel() wire.Channel
		GetVersion() Version
		GetType() (EventType, error)
	}

	OpenEvent struct {
		channel   wire.Channel
		eventType EventType
		idv       pchannel.ID
		versionV  Version
	}
	FundEvent struct {
		channel   wire.Channel
		eventType EventType
		idv       pchannel.ID
		versionV  Version
	}

	CloseEvent struct {
		channel   wire.Channel
		eventType EventType
		idv       pchannel.ID
		versionV  Version
	}

	WithdrawnEvent struct {
		channel   wire.Channel
		eventType EventType
		idv       pchannel.ID
		versionV  Version
		// Timestamp uint64
	}

	DisputedEvent struct {
		channel   wire.Channel
		eventType EventType
		idv       pchannel.ID
		versionV  Version
	}
)

type StellarEvent struct {
	Type         EventType
	ChannelState wire.Channel
}

func (e *StellarEvent) GetType() EventType {
	return e.Type
}

func (e *OpenEvent) GetChannel() wire.Channel {
	return e.channel
}
func (e *OpenEvent) GetType() (EventType, error) {
	return EventTypeOpen, nil
}

func (e *OpenEvent) GetID() pchannel.ID {
	return e.idv
}
func (e *OpenEvent) GetVersion() Version {
	return e.versionV
}

func (e *WithdrawnEvent) GetChannel() wire.Channel {
	return e.channel
}

func (e *WithdrawnEvent) GetType() (EventType, error) {
	withdrawnA := e.channel.Control.WithdrawnA
	withdrawnB := e.channel.Control.WithdrawnB

	if withdrawnA && withdrawnB {
		return EventTypeWithdrawn, nil
	} else if withdrawnA != withdrawnB {
		return EventTypeWithdrawing, nil
	}
	return EventTypeError, errors.New("withdraw event has no consistent type: not withdrawn")
}

func (e *WithdrawnEvent) GetID() pchannel.ID {
	return e.idv
}
func (e *WithdrawnEvent) GetVersion() Version {
	return e.versionV
}

func (e *CloseEvent) GetChannel() wire.Channel {
	return e.channel
}

func (e *CloseEvent) GetType() (EventType, error) {
	return EventTypeClosed, nil
}

func (e *CloseEvent) GetID() pchannel.ID {
	return e.idv
}
func (e *CloseEvent) GetVersion() Version {
	return e.versionV
}
func (e *FundEvent) GetChannel() wire.Channel {
	return e.channel
}

func (e *FundEvent) GetType() (EventType, error) {
	fundedA := e.channel.Control.FundedA
	fundedB := e.channel.Control.FundedB

	if fundedA && fundedB {
		return EventTypeFundedChannel, nil
	} else if fundedA != fundedB {
		return EventTypeFundChannel, nil
	}
	return EventTypeError, errors.New("funding event has no consistent type: not funded")
}

func (e *FundEvent) GetID() pchannel.ID {
	return e.idv
}
func (e *FundEvent) GetVersion() Version {
	return e.versionV
}

func (e *DisputedEvent) GetID() pchannel.ID {
	return e.idv
}

func (e *DisputedEvent) GetChannel() wire.Channel {
	return e.channel
}

func (e *DisputedEvent) GetVersion() Version {
	return e.versionV
}

func (e *DisputedEvent) GetType() (EventType, error) {
	return EventTypeDisputed, nil
}

func DecodeEventsPerun(txMeta xdr.TransactionMeta) ([]PerunEvent, error) {
	evs := make([]PerunEvent, 0)

	txEvents := txMeta.V3.SorobanMeta.Events

	for _, ev := range txEvents {
		sev := StellarEvent{}
		topics := ev.Body.V0.Topics

		if len(topics) < 2 {
			return nil, ErrNotStellarPerunContract
		}
		perunString, ok := topics[0].GetSym()

		if perunString == "transfer" {
			continue
		}

		if perunString != AssertPerunSymbol {
			return nil, ErrNotStellarPerunContract
		}
		if !ok {
			return nil, ErrNotStellarPerunContract
		}

		fn, ok := topics[1].GetSym()
		if !ok {
			return nil, ErrNotStellarPerunContract
		}

		if eventType, found := STELLAR_PERUN_CHANNEL_CONTRACT_TOPICS[fn]; !found {
			return nil, ErrNotStellarPerunContract
		} else {
			sev.Type = eventType
		}

		switch sev.GetType() {
		case EventTypeOpen:

			openEventchanStellar, err := GetChannelFromEvents(ev.Body.V0.Data)
			if err != nil {
				return nil, err
			}

			controlsOpen := initControlState(openEventchanStellar.Control)

			err = checkOpen(controlsOpen)
			if err != nil {
				fmt.Println(err)
			}

			openEvent := OpenEvent{
				channel: openEventchanStellar,
			}

			evs = append(evs, &openEvent)

		case EventTypeFundChannel:
			fundEventchanStellar, _, err := GetChannelBoolFromEvents(ev.Body.V0.Data)
			if err != nil {
				return nil, err
			}

			fundEvent := FundEvent{
				channel: fundEventchanStellar,
			}
			evs = append(evs, &fundEvent)
		case EventTypeClosed:
			closedEventchanStellar, err := GetChannelFromEvents(ev.Body.V0.Data)
			if err != nil {
				return nil, err
			}

			closeEvent := CloseEvent{
				channel: closedEventchanStellar,
			}
			evs = append(evs, &closeEvent)
		case EventTypeWithdrawn:
			withdrawnEventchanStellar, err := GetChannelFromEvents(ev.Body.V0.Data)
			if err != nil {
				return nil, err
			}
			withdrawnEvent := WithdrawnEvent{
				channel: withdrawnEventchanStellar,
			}
			evs = append(evs, &withdrawnEvent)

		case EventTypeDisputed:
			disputedEventchanStellar, err := GetChannelFromEvents(ev.Body.V0.Data)
			if err != nil {
				return nil, err
			}
			disputedEvent := DisputedEvent{
				channel: disputedEventchanStellar,
			}
			evs = append(evs, &disputedEvent)

		}

	}
	return evs, nil
}

func initControlState(control wire.Control) controlsState {
	return controlsState{
		"ControlFundedA":    control.FundedA,
		"ControlFundedB":    control.FundedB,
		"ControlClosed":     control.Closed,
		"ControlWithdrawnA": control.WithdrawnA,
		"ControlWithdrawnB": control.WithdrawnB,
		"ControlDisputed":   control.Disputed,
	}
}

func GetChannelFromEvents(evData xdr.ScVal) (wire.Channel, error) {
	var chanStellar wire.Channel

	err := chanStellar.FromScVal(evData)
	if err != nil {
		return wire.Channel{}, err
	}

	return chanStellar, nil
}

func GetChannelBoolFromEvents(evData xdr.ScVal) (wire.Channel, bool, error) {
	var chanStellar wire.Channel

	mvec, ok := evData.GetVec()
	if !ok {
		return wire.Channel{}, false, errors.New("expected vec")
	}

	vecVals := *mvec
	eventBool := vecVals[1]
	eventControl := vecVals[0]
	err := chanStellar.FromScVal(eventControl)
	if err != nil {
		return wire.Channel{}, false, err
	}

	boolIdx, ok := eventBool.GetB()
	if !ok {
		return wire.Channel{}, false, errors.New("expected bool")
	}

	return chanStellar, boolIdx, nil
}

func checkOpen(cState controlsState) error {
	for key, val := range cState {
		if val {
			return errors.New(key + " is not false")
		}
	}
	return nil
}

func AssertOpenEvent(perunEvents []PerunEvent) error {

	if len(perunEvents) == 0 {
		return errors.New("no open event found after opening a channel")
	}

	for _, ev := range perunEvents {
		eventType, err := ev.GetType()
		if err != nil {
			return errors.New("could not retrieve type from event")
		}
		switch eventType {
		case EventTypeOpen:
			return nil
		case EventTypeDisputed:
			return errors.New("disputed already before channel open")
		case EventTypeFundChannel, EventTypeFundedChannel:
			if ev.GetChannel().Control.FundedA || ev.GetChannel().Control.FundedB {
				return nil
			} else {
				return errors.New("funded channel not open yet")
			}
		}
	}
	return errors.New("no event found after opening a channel")
}

func AssertFundedEvent(perunEvents []PerunEvent) error {
	for _, ev := range perunEvents {
		eventType, err := ev.GetType()
		if err != nil {
			return err
		}
		switch eventType {
		case EventTypeFundChannel, EventTypeFundedChannel:
			return nil
		default:
			return ErrNoFundEvent
		}
	}

	return nil
}

func AssertCloseEvent(perunEvents []PerunEvent) error {
	for _, ev := range perunEvents {
		eventType, err := ev.GetType()
		if err != nil {
			return err
		}
		switch eventType {
		case EventTypeClosed:
			return nil
		default:
			return ErrNoCloseEvent
		}
	}

	return nil
}

func AssertWithdrawEvent(perunEvents []PerunEvent) error {
	for _, ev := range perunEvents {
		eventType, err := ev.GetType()
		if err != nil {
			return err
		}
		switch eventType {
		case EventTypeWithdrawing, EventTypeWithdrawn:
			return nil
		default:
			return ErrNoWithdrawEvent
		}
	}

	return nil
}
func AssertForceCloseEvent(perunEvents []PerunEvent) error {
	for _, ev := range perunEvents {
		eventType, err := ev.GetType()
		if err != nil {
			return err
		}
		switch eventType {
		case EventTypeForceClose:
			return nil
		default:
			return ErrNoForceCloseEvent
		}
	}

	return nil
}

func AssertDisputeEvent(perunEvents []PerunEvent) error {
	for _, ev := range perunEvents {
		eventType, err := ev.GetType()
		if err != nil {
			return err
		}
		switch eventType {
		case EventTypeDisputed:
			return nil
		default:
			return ErrNoDisputeEvent
		}
	}

	return nil
}
