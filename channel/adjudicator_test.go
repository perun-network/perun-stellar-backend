//go:build integration
// +build integration

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
package channel_test

import (
	"github.com/stretchr/testify/require"
	"log"
	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/channel"
	chtest "perun.network/perun-stellar-backend/channel/test"
	"perun.network/perun-stellar-backend/channel/types"
	wtypes "perun.network/perun-stellar-backend/wallet/types"
	"testing"
)

func TestHappyChannel(t *testing.T) {
	setup := chtest.NewTestSetup(t)
	stellarAsset := setup.GetTokenAsset()
	accs := setup.GetAccounts()
	addrAlice := accs[0].Address()
	addrBob := accs[1].Address()
	addrList := []pwallet.Address{addrAlice, addrBob}
	perunParams, perunState := chtest.NewParamsWithAddressStateWithAsset(t, addrList, stellarAsset)

	freqAlice := pchannel.NewFundingReq(perunParams, perunState, 0, perunState.Balances)
	freqBob := pchannel.NewFundingReq(perunParams, perunState, 1, perunState.Balances)

	freqs := []*pchannel.FundingReq{freqAlice, freqBob}

	funders := setup.GetFunders()
	ctx := setup.NewCtx(chtest.DefaultTestTimeout)
	err := chtest.FundAll(ctx, funders, freqs)
	require.NoError(t, err)

	// funding complete

	// Withdrawal
	{
		adjAlice := setup.GetAdjudicators()[0]
		adjBob := setup.GetAdjudicators()[1]

		ctxAliceWithdraw := setup.NewCtx(chtest.DefaultTestTimeout)

		adjState := perunState
		next := adjState.Clone()
		next.Version++
		next.IsFinal = true
		ethState := channel.ToEthState(next)
		bytes, err := channel.EncodeEthState(&ethState)
		require.NoError(t, err)

		signAlice, err := accs[0].SignData(bytes)
		require.NoError(t, err)
		signBob, err := accs[1].SignData(bytes)
		require.NoError(t, err)
		sigs := []pwallet.Sig{signAlice, signBob}
		tx := pchannel.Transaction{State: next, Sigs: sigs}

		reqAlice := pchannel.AdjudicatorReq{
			Params:    perunParams,
			Tx:        tx,
			Acc:       map[pwallet.BackendID]pwallet.Account{2: accs[0]},
			Idx:       pchannel.Index(0),
			Secondary: false}

		reqBob := pchannel.AdjudicatorReq{
			Params:    perunParams,
			Tx:        tx,
			Acc:       map[pwallet.BackendID]pwallet.Account{2: accs[1]},
			Idx:       pchannel.Index(1),
			Secondary: false}

		_, err = adjAlice.Subscribe(ctx, next.ID)
		require.NoError(t, err)

		_, err = adjBob.Subscribe(ctx, next.ID)

		require.NoError(t, err)
		require.NoError(t, adjAlice.Withdraw(ctxAliceWithdraw, reqAlice, nil))

		perunAddrAlice := adjAlice.GetPerunAddr()
		stellarChanAlice, err := adjAlice.CB.GetChannelInfo(ctx, perunAddrAlice, next.ID[2])
		require.True(t, stellarChanAlice.Control.WithdrawnA)
		require.NoError(t, err)
		require.NoError(t, adjBob.Withdraw(ctx, reqBob, nil))

	}

}

func TestHappyChannelOneWithdrawer(t *testing.T) {
	setup := chtest.NewTestSetup(t, true)
	stellarAsset := setup.GetTokenAsset()
	accs := setup.GetAccounts()
	addrAlice := accs[0].Address()
	addrBob := accs[1].Address()
	addrList := []pwallet.Address{addrAlice, addrBob}
	perunParams, perunState := chtest.NewParamsWithAddressStateWithAsset(t, addrList, stellarAsset)

	freqAlice := pchannel.NewFundingReq(perunParams, perunState, 0, perunState.Balances)
	freqBob := pchannel.NewFundingReq(perunParams, perunState, 1, perunState.Balances)

	freqs := []*pchannel.FundingReq{freqAlice, freqBob}

	funders := setup.GetFunders()
	ctx := setup.NewCtx(chtest.DefaultTestTimeout)
	err := chtest.FundAll(ctx, funders, freqs)
	require.NoError(t, err)

	// funding complete

	// Withdrawal
	{
		adjAlice := setup.GetAdjudicators()[0]
		adjBob := setup.GetAdjudicators()[1]

		adjState := perunState
		next := adjState.Clone()
		next.Version++
		next.IsFinal = true
		ethState := channel.ToEthState(next)
		encodedState, err := channel.EncodeEthState(&ethState)
		require.NoError(t, err)
		signAlice, err := accs[0].SignData(encodedState)
		require.NoError(t, err)
		signBob, err := accs[1].SignData(encodedState)
		require.NoError(t, err)
		sigs := []pwallet.Sig{signAlice, signBob}
		tx := pchannel.Transaction{State: next, Sigs: sigs}

		reqBob := pchannel.AdjudicatorReq{
			Params:    perunParams,
			Tx:        tx,
			Acc:       map[pwallet.BackendID]pwallet.Account{wtypes.StellarBackendID: accs[1]},
			Idx:       pchannel.Index(1),
			Secondary: false}

		// Bob withdraws for both, himself and Alice

		require.NoError(t, err)
		require.NoError(t, adjBob.Withdraw(ctx, reqBob, nil))
		cb := adjAlice.CB
		tr := cb.GetTransactor()
		clientAddress, err := tr.GetAddress()
		if err != nil {
			log.Println("Error while getting client address: ", err)
		}
		tokenAddr0, ok := reqBob.Tx.State.Assets[0].(*types.StellarAsset)
		bal0 := "bal0"
		bal1 := "bal1"
		if ok {
			cAdd0, err := types.MakeContractAddress(tokenAddr0.Asset.ContractID())
			require.NoError(t, err)
			bal0, err = cb.GetBalance(cAdd0)
			require.NoError(t, err)
		}
		tokenAddr1, ok := reqBob.Tx.State.Assets[1].(*types.StellarAsset)
		if ok {
			cAdd1, err := types.MakeContractAddress(tokenAddr1.Asset.ContractID())
			require.NoError(t, err)
			bal1, err = cb.GetBalance(cAdd1)
			require.NoError(t, err)
		}
		log.Println("Balance: ", bal0, bal1, " after withdrawing: ", clientAddress, reqBob.Tx.State.Assets)

	}

}

func TestChannel_RegisterFinal(t *testing.T) {
	setup := chtest.NewTestSetup(t)
	stellarAsset := setup.GetTokenAsset()
	accs := setup.GetAccounts()
	addrAlice := accs[0].Address()
	addrBob := accs[1].Address()
	addrList := []pwallet.Address{addrAlice, addrBob}
	perunParams, perunState := chtest.NewParamsWithAddressStateWithAsset(t, addrList, stellarAsset)

	freqAlice := pchannel.NewFundingReq(perunParams, perunState, 0, perunState.Balances)
	freqBob := pchannel.NewFundingReq(perunParams, perunState, 1, perunState.Balances)

	freqs := []*pchannel.FundingReq{freqAlice, freqBob}

	funders := setup.GetFunders()
	ctx := setup.NewCtx(chtest.DefaultTestTimeout)
	err := chtest.FundAll(ctx, funders, freqs)
	require.NoError(t, err)

	// funding complete

	// Withdrawal
	{
		adjAlice := setup.GetAdjudicators()[0]

		ctxAliceRegister := setup.NewCtx(chtest.DefaultTestTimeout)

		adjState := perunState
		next := adjState.Clone()
		next.Version++
		next.IsFinal = true
		ethState := channel.ToEthState(next)
		bytes, err := channel.EncodeEthState(&ethState)
		require.NoError(t, err)

		signAlice, err := accs[0].SignData(bytes)
		require.NoError(t, err)
		signBob, err := accs[1].SignData(bytes)
		require.NoError(t, err)
		require.NoError(t, err)
		sigs := []pwallet.Sig{signAlice, signBob}
		tx := pchannel.Transaction{State: next, Sigs: sigs}

		reqAlice := pchannel.AdjudicatorReq{
			Params:    perunParams,
			Tx:        tx,
			Acc:       map[pwallet.BackendID]pwallet.Account{2: accs[0]},
			Idx:       pchannel.Index(0),
			Secondary: false}

		require.NoError(t, adjAlice.Register(ctxAliceRegister, reqAlice, nil))

		perunAddrAlice := adjAlice.GetPerunAddr()
		stellarChanAlice, err := adjAlice.CB.GetChannelInfo(ctx, perunAddrAlice, next.ID[2])

		require.True(t, stellarChanAlice.Control.Disputed)

		require.NoError(t, err)

	}

}
