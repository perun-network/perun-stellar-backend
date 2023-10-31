package channel

import (
	"context"
	"errors"
	"github.com/stellar/go/keypair"
	//"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/xdr"

	"perun.network/go-perun/log"

	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/channel/env"
	//"perun.network/perun-stellar-backend/client"
	"perun.network/perun-stellar-backend/wallet"

	"perun.network/perun-stellar-backend/wire"
	"perun.network/perun-stellar-backend/wire/scval"
	"time"
)

var ErrChannelAlreadyClosed = errors.New("nonce values was out of range")

type Adjudicator struct {
	log log.Embedding
	//integrEnv       env.IntegrationTestEnv
	stellarClient   *env.StellarClient
	acc             *wallet.Account
	kpFull          *keypair.Full
	maxIters        int
	pollingInterval time.Duration
}

// NewAdjudicator returns a new Adjudicator.

func NewAdjudicator(acc *wallet.Account, stellarClient *env.StellarClient) *Adjudicator {
	return &Adjudicator{
		stellarClient:   stellarClient,
		acc:             acc,
		maxIters:        MaxIterationsUntilAbort,
		pollingInterval: DefaultPollingInterval,
	}
}

func (a Adjudicator) Subscribe(ctx context.Context, cid pchannel.ID) (pchannel.AdjudicatorSubscription, error) {
	c := a.stellarClient
	return NewAdjudicatorSub(ctx, cid, c), nil
}

func (a Adjudicator) Withdraw(ctx context.Context, req pchannel.AdjudicatorReq, smap pchannel.StateMap) error {

	cid := req.Tx.ID

	txSigner := a.stellarClient

	if req.Tx.State.IsFinal {
		log.Println("Withdraw called")
		// we listen for the close event

		// a.isConcluded

		evSub := NewAdjudicatorSub(ctx, req.Tx.ID, txSigner)
		defer evSub.Close()

		err := a.Close(ctx, req.Tx.ID, req.Tx.State, req.Tx.Sigs, req.Params)
		if err != nil {
			if err == ErrChannelAlreadyClosed {
				return a.withdraw(ctx, req)
			} else {
				return err
			}
		}
		// close has been called, now we wait for the event
		err = a.waitForClosed(ctx, evSub, cid)
		if err != nil {
			return err
		}
		//... after the event has arrived, we conclude
		return a.withdraw(ctx, req)

	} else {
		err := a.ForceClose(ctx, req.Tx.ID, req.Tx.State, req.Tx.Sigs, req.Params)
		if err != nil {
			if err == ErrChannelAlreadyClosed {
				return a.withdraw(ctx, req)
			} else {
				return err
			}
		}

		err = a.withdraw(ctx, req)
	}
	return nil
}

func (a Adjudicator) waitForClosed(ctx context.Context, evsub *AdjEventSub, cid pchannel.ID) error {
	a.log.Log().Tracef("Waiting for the channel closing event")
loop:
	for {

		select {
		case event := <-evsub.Events():

			_, ok := event.(*CloseEvent)

			if !ok {
				continue loop
			}

			evsub.Close()
			return nil

		case <-ctx.Done():
			return ctx.Err()
		case err := <-evsub.PanicErr():
			return err
		default:
			continue loop
		}

	}
}

func (a Adjudicator) BuildWithdrawTxArgs(req pchannel.AdjudicatorReq) (xdr.ScVec, error) {

	// build withdrawalargs
	chanIDStellar := req.Tx.ID[:]
	var chanid xdr.ScBytes
	copy(chanid, chanIDStellar)
	channelID, err := scval.WrapScBytes(chanIDStellar)
	if err != nil {
		return xdr.ScVec{}, err
	}

	withdrawArgs := xdr.ScVec{
		channelID,
	}
	return withdrawArgs, nil

}

func (a Adjudicator) withdraw(ctx context.Context, req pchannel.AdjudicatorReq) error {

	contractAddress := a.stellarClient.GetContractIDAddress()
	kp := a.kpFull
	hzAcc := a.stellarClient.GetHorizonAcc()
	//chanId := req.Tx.State.ID

	// generate tx to open the channel
	withdrawTxArgs, err := a.BuildWithdrawTxArgs(req)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := a.stellarClient.InvokeAndProcessHostFunction(hzAcc, "withdraw", withdrawTxArgs, contractAddress, kp)

	// build withdrawalargs

	// env := a.integrEnv
	// contractAddress := env.GetContractIDAddress()
	// kp := a.kpFull
	// acc := env.AccountDetails(kp)
	// generate tx to open the channel

	// call fct.
	// contractAddr := a.integrEnv.GetContractIDAddress()
	// //caller := a.integrEnv.Client()
	// kp := a.kpFull
	// acc := a.integrEnv.AccountDetails(kp)

	// invokeHostFunctionOp := env.BuildContractCallOp(acc, "withdraw", withdrawTxArgs, contractAddr)

	// preFlightOp, minFee := a.integrEnv.PreflightHostFunctions(&acc, *invokeHostFunctionOp)

	// tx, err := a.integrEnv.SubmitOperationsWithFee(&acc, kp, minFee, &preFlightOp)
	// if err != nil {
	// 	return errors.New("error while submitting operations with fee")
	// }

	// // read out decoded Events and interpret them
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

func (a Adjudicator) Close(ctx context.Context, id pchannel.ID, state *pchannel.State, sigs []pwallet.Sig, params *pchannel.Params) error {
	//env := a.integrEnv

	contractAddress := a.stellarClient.GetContractIDAddress()
	kp := a.kpFull
	hzAcc := a.stellarClient.GetHorizonAcc()
	closeTxArgs, err := BuildCloseTxArgs(*state, sigs)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := a.stellarClient.InvokeAndProcessHostFunction(hzAcc, "close", closeTxArgs, contractAddress, kp)

	// contractAddress := a.integrEnv.GetContractIDAddress()
	// kp := a.kpFull
	// acc := a.integrEnv.AccountDetails(kp)
	// // generate tx to open the channel
	// fundTxArgs, err := BuildCloseTxArgs(*state, sigs)
	// if err != nil {
	// 	return errors.New("error while building fund tx")
	// }
	// invokeHostFunctionOp := env.BuildContractCallOp(acc, "close", fundTxArgs, contractAddress)

	// preFlightOp, minFee := a.integrEnv.PreflightHostFunctions(&acc, *invokeHostFunctionOp)

	// tx, err := a.integrEnv.SubmitOperationsWithFee(&acc, kp, minFee, &preFlightOp)
	// if err != nil {
	// 	return errors.New("error while submitting operations with fee")
	// }

	// // read out decoded Events and interpret them
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

// Register registers and disputes a channel.
func (a Adjudicator) Register(ctx context.Context, req pchannel.AdjudicatorReq, states []pchannel.SignedState) error {
	panic("implement me")
}

func (a Adjudicator) ForceClose(ctx context.Context, id pchannel.ID, state *pchannel.State, sigs []pwallet.Sig, params *pchannel.Params) error {

	contractAddress := a.stellarClient.GetContractIDAddress()
	kp := a.kpFull
	hzAcc := a.stellarClient.GetHorizonAcc()
	forceCloseTxArgs, err := env.BuildForceCloseTxArgs(id)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := a.stellarClient.InvokeAndProcessHostFunction(hzAcc, "force_close", forceCloseTxArgs, contractAddress, kp)

	// //env := a.integrEnv
	// contractAddress := a.integrEnv.GetContractIDAddress()
	// kp := a.kpFull
	// acc := a.integrEnv.AccountDetails(kp)
	// // generate tx to open the channel
	// fundTxArgs, err := BuildCloseTxArgs(*state, sigs)
	// if err != nil {
	// 	return errors.New("error while building fund tx")
	// }
	// invokeHostFunctionOp := env.BuildContractCallOp(acc, "force_close", fundTxArgs, contractAddress)

	// preFlightOp, minFee := a.integrEnv.PreflightHostFunctions(&acc, *invokeHostFunctionOp)

	// tx, err := a.integrEnv.SubmitOperationsWithFee(&acc, kp, minFee, &preFlightOp)
	// if err != nil {
	// 	return errors.New("error while submitting operations with fee")
	// }

	// // read out decoded Events and interpret them
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

func (a Adjudicator) Progress(ctx context.Context, req pchannel.ProgressReq) error {
	// only relevant for AppChannels
	return nil
}
