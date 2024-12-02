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
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/event"
	wtypes "perun.network/perun-stellar-backend/wallet/types"
	"perun.network/perun-stellar-backend/wire"
)

var ErrCouldNotDecodeTxMeta = errors.New("could not decode tx output")

type Client struct {
	hzClient  *horizonclient.Client
	keyHolder keyHolder
}

type keyHolder struct {
	kp *keypair.Full
}

func (cb *ContractBackend) Open(ctx context.Context, perunAddr xdr.ScAddress, params *pchannel.Params, state *pchannel.State) error {
	log.Println("Open called")
	openTxArgs, err := buildOpenTxArgs(*params, *state)
	if err != nil {
		return errors.New("error while building open tx")
	}
	txMeta, err := cb.InvokeSignedTx("open", openTxArgs, perunAddr)
	if err != nil {
		return errors.Join(errors.New("error while invoking and processing host function: open"), err)
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

func (cb *ContractBackend) Abort(ctx context.Context, perunAddr xdr.ScAddress, state *pchannel.State) error {

	chanId := state.ID[wtypes.StellarBackendID]
	abortTxArgs, err := buildChanIdTxArgs(chanId)
	if err != nil {
		return errors.New("error while building abort_funding tx")
	}
	txMeta, err := cb.InvokeSignedTx("abort_funding", abortTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function: abort_funding")
	}

	_, err = event.DecodeEventsPerun(txMeta)
	if err != nil {
		return err
	}

	return nil
}

func (cb *ContractBackend) Fund(ctx context.Context, perunAddr xdr.ScAddress, chanID pchannel.ID, funderIdx bool) error {
	log.Println("Fund called by ContractBackend")
	fundTxArgs, err := buildChanIdxTxArgs(chanID, funderIdx)
	if err != nil {
		return errors.New("error while building fund tx")
	}

	txMeta, err := cb.InvokeSignedTx("fund", fundTxArgs, perunAddr)
	if err != nil {
		return err
	}

	evs, err := event.DecodeEventsPerun(txMeta)
	if err != nil {
		return err
	}

	err = event.AssertFundedEvent(evs)

	if err == event.ErrNoFundEvent {
		chanFunded, err := cb.GetChannelInfo(ctx, perunAddr, chanID)
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

func (cb *ContractBackend) Close(ctx context.Context, perunAddr xdr.ScAddress, state *pchannel.State, sigs []pwallet.Sig) error {

	log.Println("Close called by ContractBackend")
	closeTxArgs, err := buildSignedStateTxArgs(*state, sigs)
	log.Println("Close: ", closeTxArgs)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := cb.InvokeSignedTx("close", closeTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function: close")
	}

	evs, err := event.DecodeEventsPerun(txMeta)
	if err != nil {
		return err
	}

	err = event.AssertCloseEvent(evs)
	if err == event.ErrNoCloseEvent {
		chanInfo, err := cb.GetChannelInfo(ctx, perunAddr, state.ID[wtypes.StellarBackendID])
		if err != nil {
			return errors.New("could not get channel info")
		}
		if chanInfo.Control.Closed {
			return nil
		}
	}

	return event.ErrNoCloseEvent
}

func (cb *ContractBackend) ForceClose(ctx context.Context, perunAddr xdr.ScAddress, chanId pchannel.ID) error {
	log.Println("ForceClose called")

	forceCloseTxArgs, err := buildChanIdTxArgs(chanId)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := cb.InvokeSignedTx("force_close", forceCloseTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function")
	}
	evs, err := event.DecodeEventsPerun(txMeta)
	if err != nil {
		return err
	}

	err = event.AssertForceCloseEvent(evs)
	if err == event.ErrNoForceCloseEvent {
		chanInfo, err := cb.GetChannelInfo(ctx, perunAddr, chanId)
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

func (cb *ContractBackend) Dispute(ctx context.Context, perunAddr xdr.ScAddress, state *pchannel.State, sigs []pwallet.Sig) error {
	disputeTxArgs, err := buildSignedStateTxArgs(*state, sigs)
	if err != nil {
		return errors.Join(errors.New("error while building dispute tx"), err)
	}
	txMeta, err := cb.InvokeSignedTx("dispute", disputeTxArgs, perunAddr)
	if err != nil {
		return errors.Join(errors.New("error while invoking and processing host function: dispute"), err)
	}
	evs, err := event.DecodeEventsPerun(txMeta)

	if err != nil {
		return err
	}

	err = event.AssertDisputeEvent(evs)
	if err == event.ErrNoDisputeEvent {
		chanInfo, err := cb.GetChannelInfo(ctx, perunAddr, state.ID[wtypes.StellarBackendID])
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
func (cb *ContractBackend) Withdraw(ctx context.Context, perunAddr xdr.ScAddress, req pchannel.AdjudicatorReq, withdrawerIdx bool, oneWithdrawer bool) error {
	log.Println("Withdraw called by ContractBackend")

	chanID := req.Tx.State.ID

	withdrawTxArgs, err := buildWithdrawTxArgs(chanID[wtypes.StellarBackendID], withdrawerIdx, oneWithdrawer)
	if err != nil {
		return errors.New("error building fund tx")
	}
	txMeta, err := cb.InvokeSignedTx("withdraw", withdrawTxArgs, perunAddr)
	if err != nil {
		return errors.New("error in host function: withdraw")
	}
	tr := cb.GetTransactor()
	clientAddress, err := tr.GetAddress()
	if err != nil {
		log.Println("Error while getting client address: ", err)
	}
	tokenAddr0, ok := req.Tx.State.Assets[0].(*types.StellarAsset)
	bal0 := "bal0"
	bal1 := "bal1"
	if ok {
		cAdd0, err := types.MakeContractAddress(tokenAddr0.Asset.ContractID())
		if err != nil {
			return err
		}
		bal0, err = cb.GetBalance(cAdd0)
		if err != nil {
			log.Println("Error while getting balance: ", err)
		}
	}
	tokenAddr1, ok := req.Tx.State.Assets[1].(*types.StellarAsset)
	if ok {
		cAdd1, err := types.MakeContractAddress(tokenAddr1.Asset.ContractID())
		if err != nil {
			return err
		}
		bal1, err = cb.GetBalance(cAdd1)
		if err != nil {
			log.Println("Error while getting balance: ", err)
		}
	}
	log.Println("Balance: ", bal0, bal1, " after withdrawing: ", clientAddress, req.Tx.State.Assets)
	evs, err := event.DecodeEventsPerun(txMeta)
	if err != nil {
		return err
	}

	finished, err := event.AssertWithdrawEvent(evs)
	if err != event.ErrNoWithdrawEvent {
		return err
	}

	if !finished {
		chanInfo, err := cb.GetChannelInfo(ctx, perunAddr, chanID[wtypes.StellarBackendID])
		if err != nil {
			return err
		}
		if (withdrawerIdx && chanInfo.Control.WithdrawnB) || (!withdrawerIdx && chanInfo.Control.WithdrawnA) {
			return nil
		}
	} else {
		return nil
	}
	return event.ErrNoWithdrawEvent
}

func (cb *ContractBackend) GetChannelInfo(ctx context.Context, perunAddr xdr.ScAddress, chanId pchannel.ID) (wire.Channel, error) {
	getchTxArgs, err := buildChanIdTxArgs(chanId)
	if err != nil {
		return wire.Channel{}, errors.New("error while building get_channel tx")
	}
	chanInfo, _, err := cb.InvokeUnsignedTx("get_channel", getchTxArgs, perunAddr)
	if err != nil {
		return wire.Channel{}, errors.Join(errors.New("error while processing and submitting get_channel tx"), err)
	}

	return chanInfo, nil
}

func (cb *ContractBackend) GetBalanceUser(cID xdr.ScAddress) (string, error) {
	tr := cb.GetTransactor()
	addr, err := tr.GetAddress()
	if err != nil {
		return "", err
	}
	accountId, err := xdr.AddressToAccountId(addr)
	if err != nil {
		return "", err
	}
	scAddr, err := xdr.NewScAddress(xdr.ScAddressTypeScAddressTypeAccount, accountId)
	if err != nil {
		return "", err
	}
	TokenNameArgs, err := BuildGetTokenBalanceArgs(scAddr)
	if err != nil {
		return "", err
	}
	tx, err := cb.InvokeSignedTx("balance", TokenNameArgs, cID)
	if err != nil {
		return "", err
	}
	return tx.V3.SorobanMeta.ReturnValue.String(), nil
}
