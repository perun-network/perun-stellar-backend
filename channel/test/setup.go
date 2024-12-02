// Copyright 2024 PolyCrypt GmbH
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
	"path/filepath"
	pchannel "perun.network/go-perun/channel"
	ptest "perun.network/go-perun/channel/test"
	pwallet "perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wire"
	"perun.network/perun-stellar-backend/wire/scval"
	pkgtest "polycry.pt/poly-go/test"
	"runtime"
	"testing"
	"time"
)

const (
	PerunContractPath        = "testdata/perun_soroban_contract.wasm"
	StellarAssetContractPath = "testdata/perun_soroban_token.wasm"
	initLumensBalance        = "10000000"
	initTokenBalance         = uint64(2000000)
	DefaultTestTimeout       = 30
	HorizonURL               = "http://localhost:8000"
	NETWORK_PASSPHRASE       = "Standalone Network ; February 2017"
)

type Setup struct {
	t        *testing.T
	accs     []*wallet.Account
	ws       []*wallet.EphemeralWallet
	cbs      []*client.ContractBackend
	Rng      *mathrand.Rand
	funders  []*channel.Funder
	adjs     []*channel.Adjudicator
	assetIDs []pchannel.Asset
}

func (s *Setup) GetStellarClients() []*client.ContractBackend {
	return s.cbs
}

func (s *Setup) GetFunders() []*channel.Funder {
	return s.funders
}

func (s *Setup) GetAdjudicators() []*channel.Adjudicator {
	return s.adjs
}

func (s *Setup) GetTokenAsset() []pchannel.Asset {
	return s.assetIDs
}

func (s *Setup) GetAccounts() []*wallet.Account {
	return s.accs
}

func (s *Setup) GetWallets() []*wallet.EphemeralWallet {
	return s.ws
}

func getProjectRoot() (string, error) {
	_, b, _, _ := runtime.Caller(1)
	basepath := filepath.Dir(b)

	fp, err := filepath.Abs(filepath.Join(basepath, "../..")) //filepath.Abs(filepath.Join(basepath, "../.."))
	return fp, err
}

func getDataFilePath(filename string) (string, error) {
	root, err := getProjectRoot()
	if err != nil {
		return "", err
	}

	fp := filepath.Join(root, "", filename)
	return fp, nil
}

func NewTestSetup(t *testing.T, options ...bool) *Setup {

	oneWithdrawer := false

	if len(options) > 0 {
		oneWithdrawer = options[0]
	}

	_, kpsToFund, _ := MakeRandPerunAccsWallets(5)
	// kpsToFund[2], _ = keypair.ParseFull("SD4XPDWFDY25V7NRMF47QE4WT6WOFWUJIZGFRMMCRHGVINJ3RMMDG6WS")
	// kpsToFund[3], _ = keypair.ParseFull("SDHDGJMVERIXSN5LQ5KDLW3F2QIVM2D6CLP3BDHSKWBAYX53YDEY3FND")
	require.NoError(t, CreateFundStellarAccounts(kpsToFund, initLumensBalance))

	depTokenOneKp := kpsToFund[2]
	depTokenTwoKp := kpsToFund[3]

	depTokenKps := []*keypair.Full{depTokenOneKp, depTokenTwoKp}

	depPerunKp := kpsToFund[4]

	relPathPerun, err := getDataFilePath(PerunContractPath)
	require.NoError(t, err)
	relPathAsset, err := getDataFilePath(StellarAssetContractPath)
	require.NoError(t, err)

	perunAddress, _ := Deploy(t, depPerunKp, relPathPerun, HorizonURL)

	tokenAddressOne, _ := Deploy(t, depTokenOneKp, relPathAsset, HorizonURL)
	tokenAddressTwo, _ := Deploy(t, depTokenTwoKp, relPathAsset, HorizonURL)

	tokenAddresses := []xdr.ScAddress{tokenAddressOne, tokenAddressTwo}
	tokenVector, err := MakeCrossAssetVector(tokenAddresses)
	require.NoError(t, err)

	require.NoError(t, InitTokenContract(depTokenOneKp, tokenAddressOne, HorizonURL))
	require.NoError(t, InitTokenContract(depTokenTwoKp, tokenAddressTwo, HorizonURL))

	// acc0 := wallet.NewAccount("5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a", *kpsToFund[0].FromAddress(), [20]byte([]byte{86, 253, 40, 156, 238, 113, 74, 94, 71, 28, 65, 132, 54, 239, 166, 62, 120, 13, 122, 135}))
	// acc1 := wallet.NewAccount("7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6", *kpsToFund[0].FromAddress(), [20]byte([]byte{101, 54, 66, 91, 233, 90, 102, 97, 246, 198, 246, 141, 112, 155, 107, 225, 82, 120, 93, 246}))
	acc0, err := wallet.NewRandomAccountWithAddress(mathrand.New(mathrand.NewSource(0)), kpsToFund[0].FromAddress())
	acc1, err := wallet.NewRandomAccountWithAddress(mathrand.New(mathrand.NewSource(0)), kpsToFund[1].FromAddress())
	w0 := wallet.NewEphemeralWallet()
	err = w0.AddAccount(acc0)
	if err != nil {
		return nil
	}
	w1 := wallet.NewEphemeralWallet()
	err = w1.AddAccount(acc1)
	if err != nil {
		return nil
	}

	SetupAccountsAndContracts(t, depTokenKps, kpsToFund[:2], tokenAddresses, initTokenBalance)

	var assetContractIDs []pchannel.Asset

	for _, tokenAddress := range tokenAddresses {
		assetContractID, err := types.NewStellarAssetFromScAddress(tokenAddress)
		require.NoError(t, err)
		assetContractIDs = append(assetContractIDs, assetContractID)
	}

	cbs := NewContractBackendsFromKeys(kpsToFund[:2], []pwallet.Account{acc0, acc1}, HorizonURL)

	aliceCB := cbs[0]
	aliceWallet := w0

	bobCB := cbs[1]
	bobWallet := w1

	channelAccs := []*wallet.Account{acc0, acc1}
	channelCBs := []*client.ContractBackend{aliceCB, bobCB}
	channelWallets := []*wallet.EphemeralWallet{aliceWallet, bobWallet}

	funders, adjs := CreateFundersAndAdjudicators(channelAccs, cbs, perunAddress, tokenVector, oneWithdrawer)

	setup := Setup{
		t:        t,
		accs:     channelAccs,
		ws:       channelWallets,
		cbs:      channelCBs,
		funders:  funders,
		adjs:     adjs,
		assetIDs: assetContractIDs,
	}

	return &setup
}

func SetupAccountsAndContracts(t *testing.T, deployerKps []*keypair.Full, kps []*keypair.Full, tokenAddresses []xdr.ScAddress, tokenBalance uint64) {

	require.Equal(t, len(deployerKps), len(tokenAddresses))

	for i := range deployerKps {
		for _, kp := range kps {
			addr, err := types.MakeAccountAddress(kp)
			require.NoError(t, err)
			require.NoError(t, MintToken(deployerKps[i], tokenAddresses[i], tokenBalance, addr, HorizonURL))

		}
	}
}
func CreateFundersAndAdjudicators(accs []*wallet.Account, cbs []*client.ContractBackend, perunAddress xdr.ScAddress, tokenScAddresses []xdr.ScVal, oneWithdrawer bool) ([]*channel.Funder, []*channel.Adjudicator) {
	funders := make([]*channel.Funder, len(accs))
	adjs := make([]*channel.Adjudicator, len(accs))

	for i, acc := range accs {
		funders[i] = channel.NewFunder(acc, cbs[i], perunAddress, tokenScAddresses)
		adjs[i] = channel.NewAdjudicator(acc, cbs[i], perunAddress, tokenScAddresses, oneWithdrawer)
	}
	return funders, adjs
}

func NewContractBackendsFromKeys(kps []*keypair.Full, acc []pwallet.Account, url string) []*client.ContractBackend {
	cbs := make([]*client.ContractBackend, len(kps))
	// generate Configs
	for i, kp := range kps {
		cbs[i] = NewContractBackendFromKey(kp, &acc[i], url)
	}
	return cbs
}

func NewContractBackendFromKey(kp *keypair.Full, acc *pwallet.Account, url string) *client.ContractBackend {
	trConfig := client.TransactorConfig{}
	trConfig.SetKeyPair(kp)
	trConfig.SetHorizonURL(url)
	if acc != nil {
		trConfig.SetAccount(*acc)
	}
	return client.NewContractBackend(&trConfig)
}

func MakeRandPerunAccsWallets(count int) ([]*wallet.Account, []*keypair.Full, []*wallet.EphemeralWallet) {
	accs := make([]*wallet.Account, count)
	kps := make([]*keypair.Full, count)
	ws := make([]*wallet.EphemeralWallet, count)

	for i := 0; i < count; i++ {
		acc, kp, w := MakeRandPerunAccWallet()
		accs[i] = acc
		kps[i] = kp
		ws[i] = w
	}
	return accs, kps, ws
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

func MakeRandPerunAccWallet() (*wallet.Account, *keypair.Full, *wallet.EphemeralWallet) {
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
	return acc, kp, w
}

func CreateFundStellarAccounts(pairs []*keypair.Full, initialBalance string) error {

	numKps := len(pairs)

	masterClient := client.NewHorizonMasterClient(NETWORK_PASSPHRASE, HorizonURL)
	masterHzClient := masterClient.GetHorizonClient()
	sourceKey := keypair.Root(NETWORK_PASSPHRASE)

	hzClient := client.NewHorizonClient(HorizonURL)

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

	txParams := client.GetBaseTransactionParamsWithFee(&masterAccount, txnbuild.MinBaseFee+100, ops...)

	txSigned, err := client.CreateSignedTransactionWithParams([]*keypair.Full{sourceKey}, txParams, NETWORK_PASSPHRASE)

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
		log.Printf("Funded Stellar L1 account %s (%s) with %s XLM.\n",
			keys.Seed(), keys.Address(), initialBalance)
	}

	return nil
}

func NewParamsWithAddressStateWithAsset(t *testing.T, partsAddr []pwallet.Address, assets []pchannel.Asset) (*pchannel.Params, *pchannel.State) {

	rng := pkgtest.Prng(t)

	numParts := 2
	partsMapSlice := make([]map[pwallet.BackendID]pwallet.Address, len(partsAddr))
	for i, addr := range partsAddr {
		partsMapSlice[i] = map[pwallet.BackendID]pwallet.Address{
			pwallet.BackendID(2): addr,
		}
	}
	return ptest.NewRandomParamsAndState(rng, ptest.WithNumLocked(0).Append(
		ptest.WithAssets(assets...),
		ptest.WithBackend(2),
		ptest.WithNumAssets(len(assets)),
		ptest.WithVersion(0),
		ptest.WithNumParts(numParts),
		ptest.WithParts(partsMapSlice),
		ptest.WithIsFinal(false),
		ptest.WithLedgerChannel(true),
		ptest.WithVirtualChannel(false),
		ptest.WithoutApp(),
		ptest.WithBalances([]pchannel.Bal{big.NewInt(100), big.NewInt(150)}, []pchannel.Bal{big.NewInt(200), big.NewInt(250)}),
	))
}

func (s *Setup) NewCtx(testTimeout float64) context.Context {
	timeout := time.Duration(testTimeout * float64(time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	s.t.Cleanup(cancel)
	return ctx
}

const (
	SymbolTokensStellarAddress xdr.ScSymbol = "stellar_address"
	SymbolTokensEthAddress     xdr.ScSymbol = "eth_address"
	SymbolTokensChain          xdr.ScSymbol = "chain"
)

func MakeCrossAssetVector(addresses []xdr.ScAddress) ([]xdr.ScVal, error) {
	var vec []xdr.ScVal
	lidvalXdrValue := xdr.Uint64(2)
	lidval, err := scval.WrapUint64(lidvalXdrValue)
	if err != nil {
		return nil, err
	}
	lidvec := xdr.ScVec{lidval}
	for _, addr := range addresses {
		var err error
		chain, err := scval.WrapVec(lidvec)
		if err != nil {
			return nil, err
		}
		stellarAddr, err := scval.WrapScAddress(addr)
		if err != nil {
			return nil, err
		}
		defAddr := make([]byte, 20)
		ethAddr, err := scval.WrapScBytes(defAddr)
		if err != nil {
			return nil, err
		}
		m, err := wire.MakeSymbolScMap(
			[]xdr.ScSymbol{
				SymbolTokensChain,
				SymbolTokensStellarAddress,
				SymbolTokensEthAddress,
			},
			[]xdr.ScVal{chain, stellarAddr, ethAddr},
		)
		if err != nil {
			return nil, err
		}
		tokensVecVal, err := scval.WrapScMap(m)
		if err != nil {
			return nil, err
		}

		vec = append(vec, tokensVecVal)
	}
	return vec, nil
}
