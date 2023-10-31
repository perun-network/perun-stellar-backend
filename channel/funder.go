package channel

import (
	"context"
	"errors"
	//"fmt"
	"github.com/stellar/go/keypair"
	"log"
	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-stellar-backend/channel/env"
	//"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wire"

	"time"
)

const MaxIterationsUntilAbort = 10
const DefaultPollingInterval = time.Duration(5) * time.Second

type Funder struct {
	//integrEnv       env.IntegrationTestEnv
	stellarClient   *env.StellarClient
	acc             *wallet.Account
	kpFull          *keypair.Full
	maxIters        int
	pollingInterval time.Duration
}

func NewFunder(acc *wallet.Account, stellarClient *env.StellarClient) *Funder {
	return &Funder{
		//integrEnv:       *env.NewBackendEnv(),
		stellarClient:   stellarClient,
		acc:             acc,
		maxIters:        MaxIterationsUntilAbort,
		pollingInterval: DefaultPollingInterval,
	}
}

func (f Funder) Fund(ctx context.Context, req pchannel.FundingReq) error {
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

func (f Funder) fundPartyA(ctx context.Context, req pchannel.FundingReq) error {
	err := f.OpenChannel(ctx, req.Params, req.State)
	if err != nil {
		return err
	}

	// fund the channel:

	err = f.FundChannel(ctx, req.Params, req.State, false)
	if err != nil {
		return err
	}

	// look for channel ID in events
	//chanID := req.State.ID

	// await response from party B

polling:
	for i := 0; i < f.maxIters; i++ {
		select {
		case <-ctx.Done():
			return f.AbortChannel(ctx, req.Params, req.State)
		case <-time.After(f.pollingInterval):
			chanState, err := f.GetChannelState(ctx, req.Params, req.State)
			if err != nil {
				continue polling
			}

			if chanState.Control.FundedA && chanState.Control.FundedB {
				return nil
			}

		}
	}
	return f.AbortChannel(ctx, req.Params, req.State)
}

func (f Funder) fundPartyB(ctx context.Context, req pchannel.FundingReq) error {
polling:

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(f.pollingInterval):
			log.Println("Party B: Polling for opened channel...")
			chanState, err := f.GetChannelState(ctx, req.Params, req.State)
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

func (f Funder) OpenChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State) error {

	//env := f.integrEnv

	contractAddress := f.stellarClient.GetContractIDAddress()
	kp := f.kpFull
	reqAlice := f.stellarClient.GetHorizonAcc()

	// generate tx to open the channel
	openTxArgs := env.BuildOpenTxArgs(params, state)
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction(reqAlice, "open", openTxArgs, contractAddress, kp)

	// invokeHostFunctionOp := env.BuildContractCallOp(reqAlice, "open", openTxArgs, contractAddress)

	// preFlightOp, minFee := f.integrEnv.PreflightHostFunctions(&reqAlice, *invokeHostFunctionOp)

	// tx, err := f.integrEnv.SubmitOperationsWithFee(&reqAlice, kp, minFee, &preFlightOp)
	// if err != nil {
	// 	return errors.New("error while submitting operations with fee")
	// }

	// read out decoded Events and interpret them
	// txMeta, err := env.DecodeTxMeta(tx)
	// if err != nil {
	// 	return errors.New("error while decoding tx meta")
	// }
	_, err = DecodeEvents(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (f Funder) FundChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State, funderIdx bool) error {

	//env := f.integrEnv

	contractAddress := f.stellarClient.GetContractIDAddress()
	kp := f.kpFull
	reqAlice := f.stellarClient.GetHorizonAcc()
	chanId := state.ID

	// generate tx to open the channel
	openTxArgs, err := env.BuildFundTxArgs(chanId, funderIdx)
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction(reqAlice, "fund", openTxArgs, contractAddress, kp)

	// invokeHostFunctionOp := env.BuildContractCallOp(reqAlice, "open", openTxArgs, contractAddress)

	// preFlightOp, minFee := f.integrEnv.PreflightHostFunctions(&reqAlice, *invokeHostFunctionOp)

	// tx, err := f.integrEnv.SubmitOperationsWithFee(&reqAlice, kp, minFee, &preFlightOp)
	// if err != nil {
	// 	return errors.New("error while submitting operations with fee")
	// }

	// read out decoded Events and interpret them
	// txMeta, err := env.DecodeTxMeta(tx)
	// if err != nil {
	// 	return errors.New("error while decoding tx meta")
	// }
	_, err = DecodeEvents(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

// func (f Funder) FundChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State, funderIdx bool) error {

// 	//env := f.integrEnv
// 	contractAddress := f.integrEnv.GetContractIDAddress()
// 	kp := f.kpFull
// 	acc := f.integrEnv.AccountDetails(kp)
// 	chanID := state.ID
// 	// generate tx to open the channel
// 	fundTxArgs, err := env.BuildFundTxArgs(chanID, funderIdx)
// 	if err != nil {
// 		return errors.New("error while building fund tx")
// 	}
// 	invokeHostFunctionOp := env.BuildContractCallOp(acc, "fund", fundTxArgs, contractAddress)

// 	preFlightOp, minFee := f.integrEnv.PreflightHostFunctions(&acc, *invokeHostFunctionOp)

// 	tx, err := f.integrEnv.SubmitOperationsWithFee(&acc, kp, minFee, &preFlightOp)
// 	if err != nil {
// 		return errors.New("error while submitting operations with fee")
// 	}

// 	// read out decoded Events and interpret them
// 	txMeta, err := env.DecodeTxMeta(tx)
// 	if err != nil {
// 		return errors.New("error while decoding tx meta")
// 	}
// 	_, err = DecodeEvents(txMeta)
// 	if err != nil {
// 		return errors.New("error while decoding events")
// 	}

// 	return nil
// }

func (f Funder) AbortChannel(ctx context.Context, params *pchannel.Params, state *pchannel.State) error {

	contractAddress := f.stellarClient.GetContractIDAddress()
	kp := f.kpFull
	reqAlice := f.stellarClient.GetHorizonAcc()
	chanId := state.ID

	// generate tx to open the channel
	openTxArgs, err := env.BuildGetChannelTxArgs(chanId)
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction(reqAlice, "abort_funding", openTxArgs, contractAddress, kp)

	// invokeHostFunctionOp := env.BuildContractCallOp(reqAlice, "open", openTxArgs, contractAddress)

	// preFlightOp, minFee := f.integrEnv.PreflightHostFunctions(&reqAlice, *invokeHostFunctionOp)

	// tx, err := f.integrEnv.SubmitOperationsWithFee(&reqAlice, kp, minFee, &preFlightOp)
	// if err != nil {
	// 	return errors.New("error while submitting operations with fee")
	// }

	// read out decoded Events and interpret them
	// txMeta, err := env.DecodeTxMeta(tx)
	// if err != nil {
	// 	return errors.New("error while decoding tx meta")
	// }
	_, err = DecodeEvents(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

// func (f Funder) Abort(ctx context.Context, params *pchannel.Params, state *pchannel.State) error {

// 	//env := f.integrEnv
// 	contractAddress := f.integrEnv.GetContractIDAddress()
// 	kp := f.kpFull
// 	reqAlice := f.integrEnv.AccountDetails(kp)
// 	// generate tx to open the channel
// 	openTxArgs := env.BuildGetChannelTxArgs(chanId)
// 	invokeHostFunctionOp := env.BuildContractCallOp(reqAlice, "abort_channel", openTxArgs, contractAddress)

// 	preFlightOp, minFee := f.integrEnv.PreflightHostFunctions(&reqAlice, *invokeHostFunctionOp)

// 	tx, err := f.integrEnv.SubmitOperationsWithFee(&reqAlice, kp, minFee, &preFlightOp)
// 	if err != nil {
// 		return errors.New("error while submitting operations with fee")
// 	}

// 	// read out decoded Events and interpret them
// 	txMeta, err := env.DecodeTxMeta(tx)
// 	if err != nil {
// 		return errors.New("error while decoding tx meta")
// 	}
// 	_, err = DecodeEvents(txMeta)
// 	if err != nil {
// 		return errors.New("error while decoding events")
// 	}

// 	return nil
// }

func (f Funder) GetChannelState(ctx context.Context, params *pchannel.Params, state *pchannel.State) (wire.Channel, error) {

	//env := f.integrEnv

	contractAddress := f.stellarClient.GetContractIDAddress()
	kp := f.kpFull
	reqAlice := f.stellarClient.GetHorizonAcc()
	chanId := state.ID

	// generate tx to open the channel
	openTxArgs, err := env.BuildGetChannelTxArgs(chanId)
	txMeta, err := f.stellarClient.InvokeAndProcessHostFunction(reqAlice, "abort_funding", openTxArgs, contractAddress, kp)

	// invokeHostFunctionOp := env.BuildContractCallOp(reqAlice, "open", openTxArgs, contractAddress)

	// preFlightOp, minFee := f.integrEnv.PreflightHostFunctions(&reqAlice, *invokeHostFunctionOp)

	// tx, err := f.integrEnv.SubmitOperationsWithFee(&reqAlice, kp, minFee, &preFlightOp)
	// if err != nil {
	// 	return errors.New("error while submitting operations with fee")
	// }

	// read out decoded Events and interpret them

	retVal := txMeta.V3.SorobanMeta.ReturnValue
	var getChan wire.Channel

	err = getChan.FromScVal(retVal)
	if err != nil {
		return wire.Channel{}, errors.New("error while decoding return value")
	}
	return getChan, nil

}

// func (f Funder) GetChannelState(ctx context.Context, params *pchannel.Params, state *pchannel.State) (wire.Channel, error) {

// 	//env := f.integrEnv
// 	contractAddress := f.integrEnv.GetContractIDAddress()
// 	kp := f.kpFull
// 	acc := f.integrEnv.AccountDetails(kp)
// 	chanID := state.ID
// 	// generate tx to open the channel
// 	getChTxArgs, err := env.BuildGetChannelTxArgs(chanID)
// 	if err != nil {
// 		return wire.Channel{}, errors.New("error while building fund tx")
// 	}
// 	invokeHostFunctionOp := env.BuildContractCallOp(acc, "get_channel", getChTxArgs, contractAddress)

// 	preFlightOp, minFee := f.integrEnv.PreflightHostFunctions(&acc, *invokeHostFunctionOp)

// 	tx, err := f.integrEnv.SubmitOperationsWithFee(&acc, kp, minFee, &preFlightOp)
// 	if err != nil {
// 		return wire.Channel{}, errors.New("error while submitting operations with fee")
// 	}

// 	txMeta, err := env.DecodeTxMeta(tx)
// 	if err != nil {
// 		return wire.Channel{}, errors.New("error while decoding tx meta")
// 	}

// 	retVal := txMeta.V3.SorobanMeta.ReturnValue
// 	var getChan wire.Channel

// 	err = getChan.FromScVal(retVal)
// 	if err != nil {
// 		return wire.Channel{}, errors.New("error while decoding return value")
// 	}
// 	return wire.Channel{}, nil
// }
