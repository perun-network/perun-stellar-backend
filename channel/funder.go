package channel

import (
	"context"
	"errors"
	"fmt"
	"github.com/stellar/go/keypair"
	"log"
	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wire"

	"time"
)

const MaxIterationsUntilAbort = 10
const DefaultPollingInterval = time.Duration(7) * time.Second

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
	// fund the channel:

	fmt.Println("funding channel in party A: ", req.Params, "state: ", req.State)

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

	//env := f.integrEnv

	contractAddress := f.stellarClient.GetContractIDAddress()
	kp := f.kpFull
	hz := f.stellarClient.GetHorizonAcc()

	// generate tx to open the channel
	openTxArgs := env.BuildOpenTxArgs(params, state)
	fmt.Println("openTxArgs: ", openTxArgs)

	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction(hz, "open", openTxArgs, contractAddress, kp)
	if err != nil {
		return errors.New("error while invoking and processing host function: open")
	}

	_, err = DecodeEvents(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (f *Funder) FundChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State, funderIdx bool) error {

	contractAddress := f.stellarClient.GetContractIDAddress()
	kp := f.kpFull
	hzAcc := f.stellarClient.GetHorizonAcc()
	chanId := state.ID

	// generate tx to open the channel
	fundTxArgs, err := env.BuildFundTxArgs(chanId, funderIdx)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	fmt.Println("funderchan args: ", contractAddress, kp, hzAcc, chanId, fundTxArgs, funderIdx)

	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction(hzAcc, "fund", fundTxArgs, contractAddress, kp)
	if err != nil {
		return errors.New("error while invoking and processing host function: fund")
	}

	_, err = DecodeEvents(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (f *Funder) AbortChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State) error {

	contractAddress := f.stellarClient.GetContractIDAddress()
	kp := f.kpFull
	reqAlice := f.stellarClient.GetHorizonAcc()
	chanId := state.ID

	// generate tx to open the channel
	openTxArgs, err := env.BuildGetChannelTxArgs(chanId)
	if err != nil {
		return errors.New("error while building get_channel tx")
	}
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction(reqAlice, "abort_funding", openTxArgs, contractAddress, kp)
	if err != nil {
		return errors.New("error while invoking and processing host function: abort_funding")
	}

	_, err = DecodeEvents(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (f *Funder) GetChannelState(ctx context.Context, params *pchannel.Params, state *pchannel.State) (wire.Channel, error) {

	contractAddress := f.stellarClient.GetContractIDAddress()
	kp := f.kpFull
	hz := f.stellarClient.GetHorizonAcc()
	chanId := state.ID

	// generate tx to open the channel
	getchTxArgs, err := env.BuildGetChannelTxArgs(chanId)
	if err != nil {
		return wire.Channel{}, errors.New("error while building get_channel tx")
	}

	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction(hz, "get_channel", getchTxArgs, contractAddress, kp)
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
