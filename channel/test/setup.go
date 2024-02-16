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

package test

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"
	"log"
	"math/big"
	mathrand "math/rand"
	pchannel "perun.network/go-perun/channel"
	ptest "perun.network/go-perun/channel/test"
	pwallet "perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wallet"
	pkgtest "polycry.pt/poly-go/test"
	"testing"
	"time"
)

const (
	PerunContractPath        = "../testdata/perun_soroban_contract.wasm"
	StellarAssetContractPath = "../testdata/perun_soroban_token.wasm"
	initLumensBalance        = "10000000"
	initTokenBalance         = uint64(2000000)
	DefaultTestTimeout       = 200
)

type Setup struct {
	t              *testing.T
	accs           []*wallet.Account
	stellarClients []*env.StellarClient
	Rng            *mathrand.Rand
	funders        []*channel.Funder
	adjs           []*channel.Adjudicator
	assetID        pchannel.Asset
}

func (s *Setup) GetStellarClients() []*env.StellarClient {
	return s.stellarClients
}

func (s *Setup) GetFunders() []*channel.Funder {
	return s.funders
}

func (s *Setup) GetAdjudicators() []*channel.Adjudicator {
	return s.adjs
}

func (s *Setup) GetTokenAsset() pchannel.Asset {
	return s.assetID
}

func (s *Setup) GetAccounts() []*wallet.Account {
	return s.accs
}

func NewTestSetup(t *testing.T) *Setup {

	accs, kpsToFund := MakeRandPerunAccs(4)
	require.NoError(t, CreateFundStellarAccounts(kpsToFund, initLumensBalance))

	depTokenKp := kpsToFund[2]
	depPerunKp := kpsToFund[3]

	tokenAddress, _ := Deploy(t, depTokenKp, StellarAssetContractPath)
	perunAddress, _ := Deploy(t, depPerunKp, PerunContractPath)

	require.NoError(t, InitTokenContract(depTokenKp, tokenAddress))

	setupAccountsAndContracts(t, depTokenKp, kpsToFund[:2], tokenAddress, initTokenBalance)

	assetContractID, err := types.NewStellarAssetFromScAddress(tokenAddress)
	require.NoError(t, err)

	stellarClients := NewStellarClients(kpsToFund)

	aliceClient := stellarClients[0]
	bobClient := stellarClients[1]
	channelAccs := []*wallet.Account{accs[0], accs[1]}
	channelClients := []*env.StellarClient{aliceClient, bobClient}

	funders, adjs := createFundersAndAdjudicators(channelAccs, stellarClients, perunAddress, tokenAddress)

	setup := Setup{
		t:              t,
		accs:           channelAccs,
		stellarClients: channelClients,
		funders:        funders,
		adjs:           adjs,
		assetID:        assetContractID,
	}

	return &setup
}

func setupAccountsAndContracts(t *testing.T, deployerKp *keypair.Full, kps []*keypair.Full, tokenAddress xdr.ScAddress, tokenBalance uint64) {
	for _, kp := range kps {
		addr, err := types.MakeAccountAddress(kp)
		require.NoError(t, err)
		require.NoError(t, MintToken(deployerKp, tokenAddress, tokenBalance, addr))
	}
}

func createFundersAndAdjudicators(accs []*wallet.Account, clients []*env.StellarClient, perunAddress, tokenAddress xdr.ScAddress) ([]*channel.Funder, []*channel.Adjudicator) {
	funders := make([]*channel.Funder, len(accs))
	adjs := make([]*channel.Adjudicator, len(accs))
	for i, acc := range accs {
		funders[i] = channel.NewFunder(acc, clients[i], perunAddress, tokenAddress)
		adjs[i] = channel.NewAdjudicator(acc, clients[i], perunAddress, tokenAddress)
	}
	return funders, adjs
}

func NewStellarClients(kps []*keypair.Full) []*env.StellarClient {
	clients := make([]*env.StellarClient, len(kps))
	for i, kp := range kps {
		clients[i] = env.NewStellarClient(kp)
	}
	return clients
}

func MakeRandPerunAccs(count int) ([]*wallet.Account, []*keypair.Full) {
	accs := make([]*wallet.Account, count)
	kps := make([]*keypair.Full, count)

	for i := 0; i < count; i++ {
		acc, kp := MakeRandPerunAcc()
		accs[i] = acc
		kps[i] = kp
	}
	return accs, kps
}

func MakeRandPerunAcc() (*wallet.Account, *keypair.Full) {
	w := wallet.NewEphemeralWallet()

	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}

	seed := binary.LittleEndian.Uint64(b[:])

	r := mathrand.New(mathrand.NewSource(int64(seed)))

	acc, kp, err := w.AddNewAccount(r)
	if err != nil {
		panic(err)
	}
	return acc, kp
}

func CreateFundStellarAccounts(pairs []*keypair.Full, initialBalance string) error {

	numKps := len(pairs)

	masterClient := env.NewHorizonMasterClient()
	masterHzClient := masterClient.GetHorizonClient()
	sourceKey := keypair.Root(env.NETWORK_PASSPHRASE)

	hzClient := env.NewHorizonClient()

	ops := make([]txnbuild.Operation, numKps)

	accReq := horizonclient.AccountRequest{AccountID: masterClient.GetAddress()}
	sourceAccount, err := masterHzClient.AccountDetail(accReq)
	if err != nil {
		panic(err)
	}

	masterAccount := txnbuild.SimpleAccount{
		AccountID: masterClient.GetAddress(),
		Sequence:  sourceAccount.Sequence,
	}

	for i := 0; i < numKps; i++ {
		pair := pairs[i]

		ops[i] = &txnbuild.CreateAccount{
			SourceAccount: masterAccount.AccountID,
			Destination:   pair.Address(),
			Amount:        initialBalance,
		}
	}

	txParams := env.GetBaseTransactionParamsWithFee(&masterAccount, txnbuild.MinBaseFee, ops...)

	txSigned, err := env.CreateSignedTransactionWithParams([]*keypair.Full{sourceKey}, txParams)

	if err != nil {
		panic(err)
	}
	_, err = hzClient.SubmitTransaction(txSigned)
	if err != nil {
		panic(err)
	}

	accounts := make([]txnbuild.Account, numKps)
	for i, kp := range pairs {
		request := horizonclient.AccountRequest{AccountID: kp.Address()}
		account, err := hzClient.AccountDetail(request)
		if err != nil {
			panic(err)
		}

		accounts[i] = &account
	}

	for _, keys := range pairs {
		log.Printf("Funded %s (%s) with %s XLM.\n",
			keys.Seed(), keys.Address(), initialBalance)
	}

	return nil
}

func NewParamsWithAddressStateWithAsset(t *testing.T, partsAddr []pwallet.Address, asset pchannel.Asset) (*pchannel.Params, *pchannel.State) {

	rng := pkgtest.Prng(t)

	numParts := 2

	return ptest.NewRandomParamsAndState(rng, ptest.WithNumLocked(0).Append(
		ptest.WithAssets(asset),
		ptest.WithVersion(0),
		ptest.WithNumParts(numParts),
		ptest.WithParts(partsAddr...),
		ptest.WithIsFinal(false),
		ptest.WithLedgerChannel(true),
		ptest.WithVirtualChannel(false),
		ptest.WithNumAssets(1),
		ptest.WithoutApp(),
		ptest.WithBalancesInRange(big.NewInt(0).Mul(big.NewInt(1), big.NewInt(100_000)), big.NewInt(0).Mul(big.NewInt(1), big.NewInt(100_000))),
	))
}

func (s *Setup) NewCtx() context.Context {
	timeout := time.Duration(float64(DefaultTestTimeout) * float64(time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	s.t.Cleanup(cancel)
	return ctx
}
