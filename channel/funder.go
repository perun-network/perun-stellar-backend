package channel

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"log"
	"math/big"

	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wire"
	"perun.network/perun-stellar-backend/wire/scval"

	"time"
)

const MaxIterationsUntilAbort = 20
const DefaultPollingInterval = time.Duration(6) * time.Second

type Funder struct {
	stellarClient   *env.StellarClient
	acc             *wallet.Account
	kpFull          *keypair.Full
	maxIters        int
	pollingInterval time.Duration
}

func NewFunder(acc *wallet.Account, kp *keypair.Full, stellarClient *env.StellarClient) *Funder {
	return &Funder{
		stellarClient:   stellarClient,
		acc:             acc,
		kpFull:          kp,
		maxIters:        MaxIterationsUntilAbort,
		pollingInterval: DefaultPollingInterval,
	}
}

func (f *Funder) Fund(ctx context.Context, req pchannel.FundingReq) error {
	log.Println("Fund called")
	switch req.Idx {
	case 0:
		return f.fundPartyA(ctx, req)
	case 1:
		return f.fundPartyB(ctx, req)
	default:
		return errors.New("invalid index")
	}
}

func (f *Funder) fundPartyA(ctx context.Context, req pchannel.FundingReq) error {
	fmt.Println("req: polling for party A: ", req)

	err := f.OpenChannel(ctx, req.Params, req.State)
	if err != nil {

		return errors.New("error while opening channel in party A")
	}
	fmt.Println("opened channel in party A, checking state")
	chanState, err := f.GetChannelState(ctx, req.Params, req.State)
	fmt.Println("chanState after opening channel: ", chanState)
	if err != nil {
		return errors.New("error while polling for opened channel A")
	}
	fmt.Println("polled chanState for PartyA: ", chanState.Control.FundedA, chanState.Control.FundedB)

	err = f.FundChannel(ctx, req.Params, req.State, false)
	if err != nil {
		return err
	}

	// await response from party B

polling:
	for i := 0; i < f.maxIters; i++ {
		select {
		case <-ctx.Done():
			return f.AbortChannel(ctx, req.Params, req.State)
		case <-time.After(f.pollingInterval):
			fmt.Println("Party A: Polling for opened channel...")
			chanState, err := f.GetChannelState(ctx, req.Params, req.State)
			if err != nil {
				continue polling
			}
			fmt.Println("Party A: chanState.Control.FundedA && chanState.Control.FundedB: ", chanState.Control.FundedA, chanState.Control.FundedB)

			if chanState.Control.FundedA && chanState.Control.FundedB {
				return nil
			}

		}
	}
	return f.AbortChannel(ctx, req.Params, req.State)
}

func (f *Funder) fundPartyB(ctx context.Context, req pchannel.FundingReq) error {
	fmt.Println("req: polling for party B: ", req)
	// err := f.OpenChannel(ctx, req.Params, req.State)
	// if err != nil {
	// 	return errors.New("error while opening channel in party A")
	// }

polling:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(f.pollingInterval):
			log.Println("Party B: Polling for opened channel...")
			chanState, err := f.GetChannelState(ctx, req.Params, req.State)
			fmt.Println("polled chanState for PartyB: ", chanState.Control.FundedA, chanState.Control.FundedB)
			if err != nil {
				log.Println("Party B: Error while polling for opened channel:", err)
				continue polling
			}
			log.Println("Party B: Found opened channel!")
			// Optional: make some channel checks here
			if chanState.Control.FundedA && chanState.Control.FundedB {
				return nil
			}
			return f.FundChannel(ctx, req.Params, req.State, true)
		}
	}
}

func (f *Funder) OpenChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State) error {

	contractAddress := f.stellarClient.GetPerunAddress()
	kp := f.kpFull
	hz := f.stellarClient.GetHorizonAcc()

	// generate tx to open the channel
	openTxArgs := env.BuildOpenTxArgs(params, state)
	auth := []xdr.SorobanAuthorizationEntry{}
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction(hz, "open", openTxArgs, contractAddress, kp, auth)
	if err != nil {
		return errors.New("error while invoking and processing host function: open")
	}

	_, err = DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (f *Funder) FundChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State, funderIdx bool) error {

	perunContractAddress := f.stellarClient.GetPerunAddress()
	tokenContractAddress := f.stellarClient.GetTokenAddress()

	kp := f.kpFull
	hzAcc := f.stellarClient.GetHorizonAcc()
	chanId := state.ID

	// generate tx to open the channel
	fundTxArgs, err := env.BuildFundTxArgs(chanId, funderIdx)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	balsStellar, err := wire.MakeBalances(state.Allocation)
	if err != nil {
		return errors.New("error while making balances")
	}

	tokenIDAddrFromBals := balsStellar.Token

	sameContractTokenID := tokenIDAddrFromBals.Equals(tokenContractAddress)
	if !sameContractTokenID {
		panic("tokenIDAddrFromBals not equal to tokenContractAddress")
	}

	tokenAddr := tokenContractAddress
	tokenAddrXdr := scval.MustWrapScAddress(tokenAddr)

	testArgs := xdr.ScVec{tokenAddrXdr}
	hzAcctInt0 := f.stellarClient.GetHorizonAcc()
	_, err = f.stellarClient.InvokeAndProcessHostFunction(hzAcctInt0, "testinteract", testArgs, perunContractAddress, kp, []xdr.SorobanAuthorizationEntry{}) // authFundClx) // []xdr.SorobanAuthorizationEntry{}) //authFundClx
	if err != nil {
		return errors.New("error while invoking and processing host function: testinteract")
	}

	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction(hzAcc, "fund", fundTxArgs, perunContractAddress, kp, []xdr.SorobanAuthorizationEntry{}) // authFundClx) // []xdr.SorobanAuthorizationEntry{}) //authFundClx
	if err != nil {
		return errors.New("error while invoking and processing host function: fund")
	}

	_, err = DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func makePreImgAuth(passphrase string, rootInv xdr.SorobanAuthorizedInvocation) (xdr.HashIdPreimage, error) {
	max := big.NewInt(0).SetInt64(int64(^uint64(0) >> 1))
	randomPart, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}
	networkId := xdr.Hash(sha256.Sum256([]byte(passphrase)))
	ledgerEntry := uint32(100)
	lEntryXdr := xdr.Uint32(ledgerEntry)
	nonce := randomPart.Int64()
	ncXdr := xdr.Int64(nonce)

	srbPreImgAuth := xdr.HashIdPreimageSorobanAuthorization{
		NetworkId:                 networkId,
		Nonce:                     ncXdr,
		SignatureExpirationLedger: lEntryXdr,
		Invocation:                rootInv,
	}

	srbAuth := xdr.HashIdPreimage{
		Type:                 xdr.EnvelopeTypeEnvelopeTypeSorobanAuthorization,
		SorobanAuthorization: &srbPreImgAuth}

	return srbAuth, nil
}

func (f *Funder) AbortChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State) error {

	contractAddress := f.stellarClient.GetPerunAddress()
	kp := f.kpFull
	reqAlice := f.stellarClient.GetHorizonAcc()
	chanId := state.ID

	// generate tx to open the channel
	openTxArgs, err := env.BuildGetChannelTxArgs(chanId)
	if err != nil {
		return errors.New("error while building get_channel tx")
	}
	auth := []xdr.SorobanAuthorizationEntry{}
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction(reqAlice, "abort_funding", openTxArgs, contractAddress, kp, auth)
	if err != nil {
		return errors.New("error while invoking and processing host function: abort_funding")
	}

	_, err = DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (f *Funder) GetChannelState(ctx context.Context, params *pchannel.Params, state *pchannel.State) (wire.Channel, error) {

	contractAddress := f.stellarClient.GetPerunAddress()
	kp := f.kpFull
	hz := f.stellarClient.GetHorizonAcc()
	chanId := state.ID

	// generate tx to open the channel
	getchTxArgs, err := env.BuildGetChannelTxArgs(chanId)
	if err != nil {
		return wire.Channel{}, errors.New("error while building get_channel tx")
	}
	auth := []xdr.SorobanAuthorizationEntry{}
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction(hz, "get_channel", getchTxArgs, contractAddress, kp, auth)
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
