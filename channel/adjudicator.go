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
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/stellar/go/xdr"
	pchannel "perun.network/go-perun/channel"
	"perun.network/go-perun/log"
	pwallet "perun.network/go-perun/wallet"

	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/wallet"
)

var ErrChannelAlreadyClosed = errors.New("channel is already closed")

var DefaultChallengeDuration = time.Duration(20) * time.Second //nolint:gomnd

// Adjudicator implements the Adjudicator interface for Stellar.
type Adjudicator struct {
	challengeDuration *time.Duration
	log               log.Embedding
	CB                *client.ContractBackend
	acc               *wallet.Account
	assetAddrs        []xdr.ScVal // xdr.ScAddress
	perunAddr         xdr.ScAddress
	maxIters          int
	pollingInterval   time.Duration
	oneWithdrawer     bool
}

// NewAdjudicator returns a new Adjudicator.
func NewAdjudicator(acc *wallet.Account, cb *client.ContractBackend, perunID xdr.ScAddress, assetIDs []xdr.ScVal, oneWithdrawer bool) *Adjudicator {
	return &Adjudicator{
		challengeDuration: &DefaultChallengeDuration,
		CB:                cb,
		acc:               acc,
		perunAddr:         perunID,
		assetAddrs:        assetIDs,
		maxIters:          MaxIterationsUntilAbort,
		pollingInterval:   DefaultPollingInterval,
		log:               log.MakeEmbedding(log.Default()),
		oneWithdrawer:     oneWithdrawer,
	}
}

// GetPerunAddr returns the perun address of the adjudicator.
func (a *Adjudicator) GetPerunAddr() xdr.ScAddress {
	return a.perunAddr
}

// GetAssetAddrs returns the asset addresses of the adjudicator.
func (a *Adjudicator) GetAssetAddrs() []xdr.ScVal {
	return a.assetAddrs
}

// Subscribe subscribes to the adjudicator.
func (a *Adjudicator) Subscribe(ctx context.Context, cid pchannel.ID) (pchannel.AdjudicatorSubscription, error) {
	perunAddr := a.GetPerunAddr()
	assetAddrs := a.GetAssetAddrs()
	return NewAdjudicatorSub(ctx, cid, a.CB, perunAddr, assetAddrs, a.challengeDuration)
}

// Withdraw withdraws the channel.
func (a *Adjudicator) Withdraw(ctx context.Context, req pchannel.AdjudicatorReq, smap pchannel.StateMap) error {
	log.Println("Withdraw called by Adjudicator")
	chanControl, errChanState := a.CB.GetChannelInfo(ctx, a.perunAddr, req.Tx.State.ID)
	if errChanState != nil {
		return errChanState
	}
	if chanControl.Control.Closed {
		log.Println("Channel is already closed")
		if ((chanControl.Control.WithdrawnA || a.oneWithdrawer) && req.Idx == 0) || (req.Idx == 1 && chanControl.Control.WithdrawnB) {
			log.Println("Channel is already withdrawn")
			return nil
		}
		return a.handleWithdrawal(ctx, req)
	}
	//nolint:nestif
	if req.Tx.State.IsFinal {
		log.Println("Channel is final, closing now")
		withdrawSelf := needWithdraw([]pchannel.Bal{req.Tx.State.Balances[0][req.Idx], req.Tx.State.Balances[1][req.Idx]}, req.Tx.State.Assets)
		withdrawOther := needWithdraw([]pchannel.Bal{req.Tx.State.Balances[0][1-req.Idx], req.Tx.State.Balances[1][1-req.Idx]}, req.Tx.State.Assets)
		if req.Idx == 0 && a.oneWithdrawer && (!withdrawSelf || !withdrawOther) { // If one participant does not need to withdraw, the swap is cross-chain which means A does not need to close
			log.Println("A only closes when A & B have to withdraw")
			return nil
		}
		err := a.Close(ctx, req.Tx.State, req.Tx.Sigs)
		if err != nil {
			chanControl, errChanState = a.CB.GetChannelInfo(ctx, a.perunAddr, req.Tx.State.ID)
			if errChanState != nil {
				log.Println("Error getting channel info: ", errChanState)
				return errChanState
			}

			if chanControl.Control.Closed {
				if a.oneWithdrawer && req.Idx == 0 {
					log.Println("Channel is already closed, A returns nil")
					return nil
				}
				return a.handleWithdrawal(ctx, req)
			}
			log.Println("Error closing channel: ", err)
			return err
		}
		log.Println("closed channel, ", err)
		return err
	}

	if err := a.ForceClose(ctx, req.Tx.State, req.Tx.Sigs); err != nil {
		log.Println("ForceClose called")
		if errors.Is(err, ErrChannelAlreadyClosed) {
			return a.handleWithdrawal(ctx, req)
		}
		return err
	}

	log.Println("ForceClose called")
	return a.handleWithdrawal(ctx, req)
}

func (a *Adjudicator) handleWithdrawal(ctx context.Context, req pchannel.AdjudicatorReq) error {
	withdrawOther := needWithdraw([]pchannel.Bal{req.Tx.State.Balances[0][1-req.Idx], req.Tx.State.Balances[1][1-req.Idx]}, req.Tx.State.Assets)
	if a.oneWithdrawer && withdrawOther {
		log.Println("Withdrawing other", req.Idx)
		if err := a.withdrawOther(ctx, req); err != nil {
			log.Println("Error withdrawing other: ", err)
			return a.withdraw(ctx, req)
		}
	}
	withdrawSelf := needWithdraw([]pchannel.Bal{req.Tx.State.Balances[0][req.Idx], req.Tx.State.Balances[1][req.Idx]}, req.Tx.State.Assets)
	if withdrawSelf {
		log.Println("Withdrawing self", req.Idx)
		return a.withdraw(ctx, req)
	}
	return nil
}

func (a *Adjudicator) withdraw(ctx context.Context, req pchannel.AdjudicatorReq) error {
	perunAddress := a.GetPerunAddr()

	withdrawerIdx := req.Idx == 1

	return a.CB.Withdraw(ctx, perunAddress, req, withdrawerIdx, a.oneWithdrawer)
}

func (a *Adjudicator) withdrawOther(ctx context.Context, req pchannel.AdjudicatorReq) error {
	perunAddress := a.GetPerunAddr()

	return a.CB.Withdraw(ctx, perunAddress, req, false, true)
}

// Close closes the channel.
func (a *Adjudicator) Close(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error {
	log.Println("Close called by Adjudicator")
	perunAddr := a.GetPerunAddr()

	return a.CB.Close(ctx, perunAddr, state, sigs)
}

// Register registers and disputes a channel.
func (a *Adjudicator) Register(ctx context.Context, req pchannel.AdjudicatorReq, states []pchannel.SignedState) error {
	log.Println("Register called")
	err := a.Dispute(ctx, req.Tx.State, req.Tx.Sigs)
	if err != nil {
		return fmt.Errorf("error while disputing channel: %w", err)
	}
	log.Println("No withdrawal needed")
	return nil
}

// Dispute disputes a channel.
func (a *Adjudicator) Dispute(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error {
	contractAddress := a.GetPerunAddr()
	return a.CB.Dispute(ctx, contractAddress, state, sigs)
}

// ForceClose forces a channel to close.
func (a *Adjudicator) ForceClose(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error {
	return a.CB.ForceClose(ctx, a.perunAddr, state.ID)
}

// Progress is not relevant for Stellar channels.
func (a Adjudicator) Progress(ctx context.Context, req pchannel.ProgressReq) error {
	// only relevant for AppChannels
	return nil
}

func needWithdraw(balances []pchannel.Bal, assets []pchannel.Asset) bool {
	for i, bal := range balances {
		_, ok := assets[i].(*types.StellarAsset)
		if bal.Cmp(big.NewInt(0)) != 0 && ok { // if balance is 0 or asset is not stellar asset, participant does not need to withdraw
			return true
		}
	}
	return false
}
