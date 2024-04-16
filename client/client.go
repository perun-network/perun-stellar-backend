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

package client

import (
	"context"
	"errors"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"log"
	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"
	"sync"

	"perun.network/perun-stellar-backend/event"
	"perun.network/perun-stellar-backend/wire"
)

var _ StellarClient = (*Client)(nil)

var ErrCouldNotDecodeTxMeta = errors.New("could not decode tx output")

type Client struct {
	hzClient  *horizonclient.Client
	keyHolder keyHolder
	mtx       sync.Mutex
}
type keyHolder struct {
	kp *keypair.Full
}

func New(kp *keypair.Full) *Client {
	return &Client{
		hzClient:  NewHorizonClient(),
		keyHolder: newKeyHolder(kp),
		mtx:       sync.Mutex{},
	}
}

func (c *Client) Open(ctx context.Context, perunAddr xdr.ScAddress, params *pchannel.Params, state *pchannel.State) error {

	openTxArgs, err := buildOpenTxArgs(*params, *state)
	if err != nil {
		return errors.New("error while building open tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("open", openTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function: open")
	}

	evs, err := event.DecodeEventsPerun(txMeta)
	if err != nil {
		return err
	}

	err = event.AssertOpenEvent(evs)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Abort(ctx context.Context, perunAddr xdr.ScAddress, state *pchannel.State) error {

	chanId := state.ID
	abortTxArgs, err := buildChanIdTxArgs(chanId)
	if err != nil {
		return errors.New("error while building abort_funding tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("abort_funding", abortTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function: abort_funding")
	}

	_, err = event.DecodeEventsPerun(txMeta)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Fund(ctx context.Context, perunAddr xdr.ScAddress, assetAddr xdr.ScAddress, chanID pchannel.ID, fudnerIdx bool) error {

	fundTxArgs, err := buildChanIdxTxArgs(chanID, fudnerIdx)
	if err != nil {
		return errors.New("error while building fund tx")
	}

	txMeta, err := c.InvokeAndProcessHostFunction("fund", fundTxArgs, perunAddr)
	if err != nil {
		return err
	}

	evs, err := event.DecodeEventsPerun(txMeta)
	if err != nil {
		return err
	}

	err = event.AssertFundedEvent(evs)

	if err == event.ErrNoFundEvent {
		chanFunded, err := c.GetChannelInfo(ctx, perunAddr, chanID)
		if err != nil {
			return err
		}
		if chanFunded.Control.FundedA || chanFunded.Control.FundedB {
			return nil
		} else if chanFunded.Control.FundedA != chanFunded.Control.FundedB {
			return nil
		} else {
			return errors.New("no funding happened after calling fund")
		}

	} else if err != nil {
		return event.ErrNoFundEvent
	}

	return nil
}

func (c *Client) Close(ctx context.Context, perunAddr xdr.ScAddress, state *pchannel.State, sigs []pwallet.Sig) error {

	log.Println("Close called")
	closeTxArgs, err := buildSignedStateTxArgs(*state, sigs)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("close", closeTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function: close")
	}

	evs, err := event.DecodeEventsPerun(txMeta)
	if err != nil {
		return err
	}

	err = event.AssertCloseEvent(evs)
	if err == event.ErrNoCloseEvent {
		chanInfo, err := c.GetChannelInfo(ctx, perunAddr, state.ID)
		if err != nil {
			return errors.New("could not get channel info")
		}
		if chanInfo.Control.Closed {
			return nil
		}
	}

	return event.ErrNoCloseEvent
}

func (c *Client) ForceClose(ctx context.Context, perunAddr xdr.ScAddress, chanId pchannel.ID) error {
	log.Println("ForceClose called")

	forceCloseTxArgs, err := buildChanIdTxArgs(chanId)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("force_close", forceCloseTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function")
	}
	evs, err := event.DecodeEventsPerun(txMeta)
	if err != nil {
		return err
	}

	err = event.AssertForceCloseEvent(evs)
	if err == event.ErrNoForceCloseEvent {
		chanInfo, err := c.GetChannelInfo(ctx, perunAddr, chanId)
		if err != nil {
			return errors.New("could not retrieve channel info")
		}
		if !chanInfo.Control.Disputed {
			return errors.New("force close of a state that is not disputed")
		}

		if chanInfo.Control.Closed {
			return errors.New("force close of a channel that is closed already")
		}

	}

	return nil
}

func (c *Client) Dispute(ctx context.Context, perunAddr xdr.ScAddress, state *pchannel.State, sigs []pwallet.Sig) error {
	closeTxArgs, err := buildSignedStateTxArgs(*state, sigs)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("dispute", closeTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function: dispute")
	}
	evs, err := event.DecodeEventsPerun(txMeta)

	if err != nil {
		return err
	}

	err = event.AssertDisputeEvent(evs)
	if err == event.ErrNoDisputeEvent {
		chanInfo, err := c.GetChannelInfo(ctx, perunAddr, state.ID)
		if err != nil {
			return errors.New("could not retrieve channel info")
		}
		if chanInfo.Control.Disputed || chanInfo.Control.Closed {
			return nil
		}
	} else {
		return err
	}

	return nil
}
func (c *Client) Withdraw(ctx context.Context, perunAddr xdr.ScAddress, req pchannel.AdjudicatorReq) error {
	chanID, partyIdx := req.Tx.State.ID, req.Idx
	withdrawerIdx := partyIdx == 1
	if partyIdx > 1 {
		return errors.New("invalid party index for withdrawal")
	}

	withdrawTxArgs, err := buildChanIdxTxArgs(chanID, withdrawerIdx)
	if err != nil {
		return errors.New("error building fund tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("withdraw", withdrawTxArgs, perunAddr)
	if err != nil {
		return errors.New("error in host function: withdraw")
	}

	evs, err := event.DecodeEventsPerun(txMeta)
	if err != nil {
		return err
	}

	err = event.AssertWithdrawEvent(evs)
	if err != event.ErrNoWithdrawEvent {
		return err
	}

	chanInfo, err := c.GetChannelInfo(ctx, perunAddr, chanID)
	if err != nil {
		return err
	}
	if (withdrawerIdx && chanInfo.Control.WithdrawnB) || (!withdrawerIdx && chanInfo.Control.WithdrawnA) {
		return nil
	}

	return event.ErrNoWithdrawEvent
}

func (c *Client) GetChannelInfo(ctx context.Context, perunAddr xdr.ScAddress, chanId pchannel.ID) (wire.Channel, error) {

	getchTxArgs, err := buildChanIdTxArgs(chanId)
	if err != nil {
		return wire.Channel{}, errors.New("error while building get_channel tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("get_channel", getchTxArgs, perunAddr)
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
