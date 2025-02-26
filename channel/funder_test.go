//go:build integration
// +build integration

// Copyright 2025 PolyCrypt GmbH
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
	"log"
	"testing"

	"github.com/stretchr/testify/require"
	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"

	chtest "perun.network/perun-stellar-backend/channel/test"
)

func TestCrossChainFunding_Happy(t *testing.T) {
	setup := chtest.NewTestSetup(t)
	stellarAsset := setup.GetTokenAsset()
	accs := setup.GetAccounts()
	addrAlice := accs[0].Address()
	addrBob := accs[1].Address()
	// here need ethereum address and stellar address
	addrList := []pwallet.Address{addrAlice, addrBob}

	perunParams, perunState := chtest.NewParamsWithAddressStateWithAsset(t, addrList, stellarAsset)
	freqAlice := pchannel.NewFundingReq(perunParams, perunState, 0, perunState.Balances)
	freqBob := pchannel.NewFundingReq(perunParams, perunState, 1, perunState.Balances)
	freqs := []*pchannel.FundingReq{freqAlice, freqBob}
	funders := setup.GetFunders()
	ctx := setup.NewCtx(chtest.DefaultTestTimeout)
	err := chtest.FundAll(ctx, funders, freqs)
	require.NoError(t, err)
}

func TestFunding_Happy(t *testing.T) {
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
}

func TestFunding_TimeoutNotFunded(t *testing.T) {
	setup := chtest.NewTestSetup(t)
	stellarAssets := setup.GetTokenAsset()
	accs := setup.GetAccounts()
	addrAlice := accs[0].Address()
	addrBob := accs[1].Address()
	addrList := []pwallet.Address{addrAlice, addrBob}
	perunParams, perunState := chtest.NewParamsWithAddressStateWithAsset(t, addrList, stellarAssets)
	freqAlice := pchannel.NewFundingReq(perunParams, perunState, 0, perunState.Balances)
	freqBob := pchannel.NewFundingReq(perunParams, perunState, 1, perunState.Balances)
	freqs := []*pchannel.FundingReq{freqAlice, freqBob}
	funders := setup.GetFunders()
	ctxTimeout := setup.NewCtx(chtest.DefaultTestTimeout)
	gotErr := funders[0].Fund(ctxTimeout, *freqs[0])
	log.Println(gotErr)
	require.True(t, pchannel.IsFundingTimeoutError(gotErr))
}
