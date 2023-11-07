package channel

import (
	"context"
	"errors"
	"fmt"
	"github.com/stellar/go/keypair"
	//"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/log"
	"reflect"

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
	log             log.Embedding
	stellarClient   *env.StellarClient
	acc             *wallet.Account
	kpFull          *keypair.Full
	maxIters        int
	pollingInterval time.Duration
}

// NewAdjudicator returns a new Adjudicator.

func NewAdjudicator(acc *wallet.Account, kp *keypair.Full, stellarClient *env.StellarClient) *Adjudicator {
	return &Adjudicator{
		stellarClient:   stellarClient,
		acc:             acc,
		kpFull:          kp,
		maxIters:        MaxIterationsUntilAbort,
		pollingInterval: DefaultPollingInterval,
		log:             log.MakeEmbedding(log.Default()),
	}
}

func (a *Adjudicator) Subscribe(ctx context.Context, cid pchannel.ID) (pchannel.AdjudicatorSubscription, error) {
	c := a.stellarClient
	return NewAdjudicatorSub(ctx, cid, c), nil
}

func (a *Adjudicator) Withdraw(ctx context.Context, req pchannel.AdjudicatorReq, smap pchannel.StateMap) error {

	//cid := req.Tx.ID

	//txSigner := a.stellarClient

	if req.Tx.State.IsFinal {
		log.Println("Withdraw called")
		// we listen for the close event

		// a.isConcluded

		//evSub := NewAdjudicatorSub(ctx, req.Tx.ID, txSigner)
		//defer evSub.Close()

		err := a.Close(ctx, req.Tx.ID, req.Tx.State, req.Tx.Sigs, req.Params)
		if err != nil {
			getChanArgs, err := env.BuildGetChannelTxArgs(req.Tx.ID)
			if err != nil {
				panic(err)
			}
			chanControl, err := a.stellarClient.GetChannelState(getChanArgs)
			if err != nil {
				return err
			}

			if chanControl.Control.Closed {
				return a.withdraw(ctx, req)
			}

			// if err == ErrChannelAlreadyClosed {
			// 	return a.withdraw(ctx, req)
			// } else {
			// 	return err
			// }
		}
		//close has been called, now we wait for the event
		//err = a.waitForClosed(ctx, evSub, cid)
		if err != nil {
			return err
		}
		//... after the event has arrived, we conclude
		return a.withdraw(ctx, req)

	} else {
		err := a.ForceClose(ctx, req.Tx.ID, req.Tx.State, req.Tx.Sigs, req.Params)
		fmt.Println("ForceClose called")
		if err != nil {
			if err == ErrChannelAlreadyClosed {
				return a.withdraw(ctx, req)
			} else {
				return err
			}
		}

		err = a.withdraw(ctx, req)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Adjudicator) waitForClosed(ctx context.Context, evsub *AdjEventSub, cid pchannel.ID) error {
	a.log.Log().Tracef("Waiting for the channel closing event")
	fmt.Println("Waiting for the channel closing event")

loop:
	for {

		select {
		case event := <-evsub.Events():
			fmt.Println("reflect.TypeOf(event): ", reflect.TypeOf(event))
			_, ok := event.(*CloseEvent)

			if !ok {
				continue loop
			}
			fmt.Println("closeevent received: ", event)

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

func (a *Adjudicator) BuildWithdrawTxArgs(req pchannel.AdjudicatorReq) (xdr.ScVec, error) {

	// build withdrawalargs
	chanIDStellar := req.Tx.ID[:]
	partyIdx := req.Idx
	var withdrawIdx xdr.ScVal
	if partyIdx == 0 {
		withdrawIdx = scval.MustWrapBool(false)
	} else if partyIdx == 1 {
		withdrawIdx = scval.MustWrapBool(true)
	} else {
		panic("partyIdx must be 0 or 1")
	}
	var chanid xdr.ScBytes
	copy(chanid, chanIDStellar)
	channelID, err := scval.WrapScBytes(chanIDStellar)
	if err != nil {
		return xdr.ScVec{}, err
	}

	withdrawArgs := xdr.ScVec{
		channelID,
		withdrawIdx,
	}
	return withdrawArgs, nil

}

func (a *Adjudicator) withdraw(ctx context.Context, req pchannel.AdjudicatorReq) error {

	contractAddress := a.stellarClient.GetContractIDAddress()
	kp := a.kpFull
	hzAcc := a.stellarClient.GetHorizonAcc()

	// generate tx to open the channel
	withdrawTxArgs, err := a.BuildWithdrawTxArgs(req)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := a.stellarClient.InvokeAndProcessHostFunction(hzAcc, "withdraw", withdrawTxArgs, contractAddress, kp)
	if err != nil {
		return errors.New("error while invoking and processing host function: withdraw")
	}

	_, err = DecodeEvents(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (a *Adjudicator) Close(ctx context.Context, id pchannel.ID, state *pchannel.State, sigs []pwallet.Sig, params *pchannel.Params) error {
	fmt.Println("Close called")
	contractAddress := a.stellarClient.GetContractIDAddress()
	kp := a.kpFull
	hzAcc := a.stellarClient.GetHorizonAcc()
	closeTxArgs, err := BuildCloseTxArgs(*state, sigs)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := a.stellarClient.InvokeAndProcessHostFunction(hzAcc, "close", closeTxArgs, contractAddress, kp)
	if err != nil {
		return errors.New("error while invoking and processing host function: close")
	}
	_, err = DecodeEvents(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}
	return nil
}

// Register registers and disputes a channel.
func (a *Adjudicator) Register(ctx context.Context, req pchannel.AdjudicatorReq, states []pchannel.SignedState) error {
	panic("implement me")
}

func (a *Adjudicator) ForceClose(ctx context.Context, id pchannel.ID, state *pchannel.State, sigs []pwallet.Sig, params *pchannel.Params) error {
	fmt.Println("ForceClose called")
	contractAddress := a.stellarClient.GetContractIDAddress()
	kp := a.kpFull
	hzAcc := a.stellarClient.GetHorizonAcc()
	forceCloseTxArgs, err := env.BuildForceCloseTxArgs(id)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := a.stellarClient.InvokeAndProcessHostFunction(hzAcc, "force_close", forceCloseTxArgs, contractAddress, kp)
	if err != nil {
		return errors.New("error while invoking and processing host function")
	}
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
