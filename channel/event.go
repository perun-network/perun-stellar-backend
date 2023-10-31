package channel

import (
	"errors"
	"fmt"
	"github.com/stellar/go/xdr"
	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/wire"
	"time"
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
	EventTypeForceClosed             // participant has force closed the channel
	EventTypeDisputed                // participant has disputed the channel
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
		xdr.ScSymbol("f_closed"): EventTypeForceClosed,
		xdr.ScSymbol("dispute"):  EventTypeDisputed,
	}

	ErrNotStellarPerunContract = errors.New("event was not from a Perun payment channel contract")
	ErrEventUnsupported        = errors.New("this type of event is unsupported")
	ErrEventIntegrity          = errors.New("contract ID does not match payment channel + passphrase")
)

type controlsState map[string]bool

type (

	// PerunEvent is a Perun event.
	PerunEvent interface {
		ID() pchannel.ID
		Timeout() pchannel.Timeout
		Version() Version
	}

	OpenEvent struct {
		Channel   wire.Channel
		IDV       pchannel.ID
		VersionV  Version
		Timestamp uint64
	}
	FundEvent struct {
		Channel   wire.Channel
		IDV       pchannel.ID
		VersionV  Version
		Timestamp uint64
	}

	CloseEvent struct {
		Channel   wire.Channel
		IDV       pchannel.ID
		VersionV  Version
		Timestamp uint64
	}

	WithdrawnEvent struct {
		Channel   wire.Channel
		IDV       pchannel.ID
		VersionV  Version
		Timestamp uint64
	}

	DisputedEvent struct {
		Channel   wire.Channel
		IDV       pchannel.ID
		VersionV  Version
		Timestamp uint64
	}

	// ConcludedEvent struct {
	// 	Finalized bool
	// 	Alloc     [2]uint64
	// 	Tout      uint64
	// 	Timestamp uint64
	// 	concluded bool
	// 	VersionV  Version
	// 	IDV       pchannel.ID
	// }
)

type StellarEvent struct {
	Type         EventType
	ChannelState wire.Channel
}

func (e StellarEvent) GetType() EventType {
	return e.Type
}

func (e OpenEvent) GetChannel() wire.Channel {
	return e.Channel
}

func (e OpenEvent) ID() pchannel.ID {
	return e.IDV
}
func (e OpenEvent) Version() Version {
	return e.VersionV
}
func (e OpenEvent) Tstamp() uint64 {
	return e.Timestamp
}

func (e OpenEvent) Timeout() pchannel.Timeout {
	when := time.Now().Add(10 * time.Second)
	pollInterval := 1 * time.Second
	return NewTimeout(when, pollInterval)
}

func (e WithdrawnEvent) GetChannel() wire.Channel {
	return e.Channel
}

func (e WithdrawnEvent) ID() pchannel.ID {
	return e.IDV
}
func (e WithdrawnEvent) Version() Version {
	return e.VersionV
}
func (e WithdrawnEvent) Tstamp() uint64 {
	return e.Timestamp
}

func (e WithdrawnEvent) Timeout() pchannel.Timeout {
	when := time.Now().Add(10 * time.Second)
	pollInterval := 1 * time.Second
	return NewTimeout(when, pollInterval)
}

func (e CloseEvent) GetChannel() wire.Channel {
	return e.Channel
}

func (e CloseEvent) ID() pchannel.ID {
	return e.IDV
}
func (e CloseEvent) Version() Version {
	return e.VersionV
}
func (e CloseEvent) Tstamp() uint64 {
	return e.Timestamp
}

func (e CloseEvent) Timeout() pchannel.Timeout {
	when := time.Now().Add(10 * time.Second)
	pollInterval := 1 * time.Second
	return NewTimeout(when, pollInterval)
}

func (e FundEvent) Timeout() pchannel.Timeout {
	when := time.Now().Add(10 * time.Second)
	pollInterval := 1 * time.Second
	return NewTimeout(when, pollInterval)
}

func (e FundEvent) ID() pchannel.ID {
	return e.IDV
}
func (e FundEvent) Version() Version {
	return e.VersionV
}
func (e FundEvent) Tstamp() uint64 {
	return e.Timestamp
}

func (e DisputedEvent) Timeout() pchannel.Timeout {
	when := time.Now().Add(10 * time.Second)
	pollInterval := 1 * time.Second
	return NewTimeout(when, pollInterval)
}

func (e DisputedEvent) ID() pchannel.ID {
	return e.IDV
}
func (e DisputedEvent) Version() Version {
	return e.VersionV
}
func (e DisputedEvent) Tstamp() uint64 {
	return e.Timestamp
}

func DecodeEvents(txMeta xdr.TransactionMeta) ([]PerunEvent, error) {
	evs := make([]PerunEvent, 0)

	txEvents := txMeta.V3.SorobanMeta.Events

	fmt.Println("txEvents: ", txEvents)

	for _, ev := range txEvents {
		sev := StellarEvent{}
		topics := ev.Body.V0.Topics

		if len(topics) < 2 {
			panic(ErrNotStellarPerunContract)
		}
		perunString, ok := topics[0].GetSym()
		if perunString != AssertPerunSymbol {
			panic(ErrNotStellarPerunContract)
		}
		if !ok {
			return []PerunEvent{}, ErrNotStellarPerunContract
		}

		fn, ok := topics[1].GetSym()
		if !ok {
			return []PerunEvent{}, ErrNotStellarPerunContract
		}

		if eventType, found := STELLAR_PERUN_CHANNEL_CONTRACT_TOPICS[fn]; !found {
			return []PerunEvent{}, ErrNotStellarPerunContract
		} else {
			sev.Type = eventType
		}

		switch sev.GetType() {
		case EventTypeOpen:

			openEventchanStellar, err := GetChannelFromEvents(ev.Body.V0.Data)
			if err != nil {
				return []PerunEvent{}, err
			}

			controlsOpen := initControlState(openEventchanStellar.Control)

			err = checkOpen(controlsOpen)
			if err != nil {
				fmt.Println(err)
			}

			openEvent := OpenEvent{
				Channel: openEventchanStellar,
			}

			evs = append(evs, openEvent)

		case EventTypeFundChannel:
			fundEventchanStellar, _, err := GetChannelBoolFromEvents(ev.Body.V0.Data)
			if err != nil {
				return []PerunEvent{}, err
			}

			fundEvent := FundEvent{
				Channel: fundEventchanStellar,
			}
			evs = append(evs, fundEvent)

		case EventTypeClosed:
			closedEventchanStellar, err := GetChannelFromEvents(ev.Body.V0.Data)
			if err != nil {
				return []PerunEvent{}, err
			}

			closeEvent := CloseEvent{
				Channel: closedEventchanStellar,
			}
			evs = append(evs, closeEvent)
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

func (e CloseEvent) EventDataFromChannel(chanState wire.Channel, timestamp uint64) error {

	chanID := chanState.State.ChannelID
	var cid [32]byte
	copy(cid[:], chanID[:])

	e.IDV = cid
	//e.VersionV = version
	e.Timestamp = timestamp
	e.Channel = chanState
	return nil
}

func (e FundEvent) EventDataFromChannel(chanState wire.Channel, timestamp uint64) error {

	chanID := chanState.State.ChannelID
	var cid [32]byte
	copy(cid[:], chanID[:])

	e.IDV = cid
	//e.VersionV = version
	e.Timestamp = timestamp
	e.Channel = chanState
	return nil
}

func (e WithdrawnEvent) EventDataFromChannel(chanState wire.Channel, timestamp uint64) error {

	chanID := chanState.State.ChannelID
	var cid [32]byte
	copy(cid[:], chanID[:])

	e.IDV = cid
	//e.VersionV = version
	e.Timestamp = timestamp
	e.Channel = chanState
	return nil
}

func (e DisputedEvent) EventDataFromChannel(chanState wire.Channel, timestamp uint64) error {

	chanID := chanState.State.ChannelID
	var cid [32]byte
	copy(cid[:], chanID[:])

	e.IDV = cid
	//e.VersionV = version
	e.Timestamp = timestamp
	e.Channel = chanState
	return nil
}
