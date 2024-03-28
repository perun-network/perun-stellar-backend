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
	"fmt"
	"github.com/stellar/go/xdr"
	pchannel "perun.network/go-perun/channel"
	"perun.network/go-perun/log"
	pwallet "perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/event"
	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wire"
	"perun.network/perun-stellar-backend/wire/scval"
	"time"
)

var ErrChannelAlreadyClosed = errors.New("channel is already closed")

type Adjudicator struct {
	log             log.Embedding
	StellarClient   *client.Client
	acc             *wallet.Account
	assetAddr       xdr.ScAddress
	perunAddr       xdr.ScAddress
	maxIters        int
	pollingInterval time.Duration
}

// NewAdjudicator returns a new Adjudicator.

func NewAdjudicator(acc *wallet.Account, stellarClient *client.Client, perunID xdr.ScAddress, assetID xdr.ScAddress) *Adjudicator {
	return &Adjudicator{
		StellarClient:   stellarClient,
		acc:             acc,
		perunAddr:       perunID,
		assetAddr:       assetID,
		maxIters:        MaxIterationsUntilAbort,
		pollingInterval: DefaultPollingInterval,
		log:             log.MakeEmbedding(log.Default()),
	}
}

func (a *Adjudicator) GetPerunAddr() xdr.ScAddress {
	return a.perunAddr
}

func (a *Adjudicator) GetAssetAddr() xdr.ScAddress {
	return a.assetAddr
}

func (a *Adjudicator) Subscribe(ctx context.Context, cid pchannel.ID) (pchannel.AdjudicatorSubscription, error) {
	c := a.StellarClient
	perunAddr := a.GetPerunAddr()
	assetAddr := a.GetAssetAddr()
	return NewAdjudicatorSub(ctx, cid, c, perunAddr, assetAddr)
}

func (a *Adjudicator) Withdraw(ctx context.Context, req pchannel.AdjudicatorReq, smap pchannel.StateMap) error {

	if req.Tx.State.IsFinal {
		log.Println("Withdraw called")

		if err := a.Close(ctx, req.Tx.State, req.Tx.Sigs); err != nil {
			chanControl, errChanState := a.StellarClient.GetChannelInfo(ctx, a.perunAddr, req.Tx.State.ID)
			if errChanState != nil {
				return errChanState
			}

			if chanControl.Control.Closed {
				return a.withdraw(ctx, req)
			}
			return err
		}

		return a.withdraw(ctx, req)

	} else {
		err := a.ForceClose(ctx, req.Tx.State, req.Tx.Sigs)
		log.Println("ForceClose called")
		if errors.Is(err, ErrChannelAlreadyClosed) {
			return a.withdraw(ctx, req)
		}
		if err != nil {
			return err
		}

		err = a.withdraw(ctx, req)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Adjudicator) BuildWithdrawTxArgs(req pchannel.AdjudicatorReq) (xdr.ScVec, error) {

	chanIDStellar := req.Tx.ID[:]
	partyIdx := req.Idx
	var withdrawIdx xdr.ScVal
	if partyIdx == 0 {
		withdrawIdx = scval.MustWrapBool(false)
	} else if partyIdx == 1 {
		withdrawIdx = scval.MustWrapBool(true)
	} else {
		return xdr.ScVec{}, errors.New("invalid party index")
	}
	var chanid xdr.ScBytes
	copy(chanid, chanIDStellar)
	channelID, err := scval.WrapScBytes(chanIDStellar)
	if err != nil {
		return xdr.ScVec{}, err
	}

	withdrawArgs := xdr.ScVec{
		channelID,
		withdrawIdx,
	}
	return withdrawArgs, nil

}

func (a *Adjudicator) withdraw(ctx context.Context, req pchannel.AdjudicatorReq) error {

	perunAddress := a.GetPerunAddr()
	withdrawTxArgs, err := a.BuildWithdrawTxArgs(req)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := a.StellarClient.InvokeAndProcessHostFunction("withdraw", withdrawTxArgs, perunAddress)
	if err != nil {
		return errors.New("error while invoking and processing host function: withdraw")
	}

	_, err = event.DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (a *Adjudicator) Close(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error {

	log.Println("Close called")
	contractAddress := a.GetPerunAddr()
	closeTxArgs, err := BuildCloseTxArgs(*state, sigs)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := a.StellarClient.InvokeAndProcessHostFunction("close", closeTxArgs, contractAddress)
	if err != nil {
		return errors.New("error while invoking and processing host function: close")
	}

	_, err = event.DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

// Register registers and disputes a channel.
func (a *Adjudicator) Register(ctx context.Context, req pchannel.AdjudicatorReq, states []pchannel.SignedState) error {
	log.Println("Register called")
	sigs := req.Tx.Sigs
	state := req.Tx.State
	err := a.Dispute(ctx, state, sigs)
	if err != nil {
		return fmt.Errorf("error while disputing channel: %w", err)
	}
	return nil
}

func (a *Adjudicator) Dispute(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error {
	contractAddress := a.GetPerunAddr()
	closeTxArgs, err := BuildDisputeTxArgs(*state, sigs)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := a.StellarClient.InvokeAndProcessHostFunction("dispute", closeTxArgs, contractAddress)
	if err != nil {
		return errors.New("error while invoking and processing host function: dispute")
	}
	_, err = event.DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}
	return nil
}

func (a *Adjudicator) ForceClose(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error {
	return a.StellarClient.ForceClose(ctx, a.perunAddr, state.ID)
}

func BuildCloseTxArgs(state pchannel.State, sigs []pwallet.Sig) (xdr.ScVec, error) {

	wireState, err := wire.MakeState(state)
	if err != nil {
		return xdr.ScVec{}, err
	}

	sigAXdr, err := scval.WrapScBytes(sigs[0])
	if err != nil {
		return xdr.ScVec{}, err
	}
	sigBXdr, err := scval.WrapScBytes(sigs[1])
	if err != nil {
		return xdr.ScVec{}, err
	}
	xdrState, err := wireState.ToScVal()
	if err != nil {
		return xdr.ScVec{}, err
	}

	fundArgs := xdr.ScVec{
		xdrState,
		sigAXdr,
		sigBXdr,
	}
	return fundArgs, nil
}

func BuildDisputeTxArgs(state pchannel.State, sigs []pwallet.Sig) (xdr.ScVec, error) {

	wireState, err := wire.MakeState(state)
	if err != nil {
		return xdr.ScVec{}, err
	}

	sigAXdr, err := scval.WrapScBytes(sigs[0])
	if err != nil {
		return xdr.ScVec{}, err
	}
	sigBXdr, err := scval.WrapScBytes(sigs[1])
	if err != nil {
		return xdr.ScVec{}, err
	}
	xdrState, err := wireState.ToScVal()
	if err != nil {
		return xdr.ScVec{}, err
	}

	fundArgs := xdr.ScVec{
		xdrState,
		sigAXdr,
		sigBXdr,
	}
	return fundArgs, nil
}

func (a Adjudicator) Progress(ctx context.Context, req pchannel.ProgressReq) error {
	// only relevant for AppChannels
	return nil
}
