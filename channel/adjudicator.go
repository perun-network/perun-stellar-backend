package channel

import (
	"context"
	"errors"
	"fmt"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/log"

	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/wallet"

	"time"

	"perun.network/perun-stellar-backend/wire"
	"perun.network/perun-stellar-backend/wire/scval"
)

var ErrChannelAlreadyClosed = errors.New("Channel is already closed")

type Adjudicator struct {
	log             log.Embedding
	stellarClient   *env.StellarClient
	acc             *wallet.Account
	kpFull          *keypair.Full
	assetID         xdr.ScAddress
	perunID         xdr.ScAddress
	maxIters        int
	pollingInterval time.Duration
}

// NewAdjudicator returns a new Adjudicator.

func NewAdjudicator(acc *wallet.Account, kp *keypair.Full, stellarClient *env.StellarClient, perunID xdr.ScAddress, assetID xdr.ScAddress) *Adjudicator {
	return &Adjudicator{
		stellarClient:   stellarClient,
		acc:             acc,
		kpFull:          kp,
		perunID:         perunID,
		assetID:         assetID,
		maxIters:        MaxIterationsUntilAbort,
		pollingInterval: DefaultPollingInterval,
		log:             log.MakeEmbedding(log.Default()),
	}
}

func (a *Adjudicator) GetPerunID() xdr.ScAddress {
	return a.perunID
}

func (a *Adjudicator) GetAssetID() xdr.ScAddress {
	return a.assetID
}

func (a *Adjudicator) Subscribe(ctx context.Context, cid pchannel.ID) (pchannel.AdjudicatorSubscription, error) {
	c := a.stellarClient
	perunID := a.GetPerunID()
	assetID := a.GetAssetID()
	return NewAdjudicatorSub(ctx, cid, c, perunID, assetID), nil
}

func (a *Adjudicator) Withdraw(ctx context.Context, req pchannel.AdjudicatorReq, smap pchannel.StateMap) error {

	// cid := req.Tx.State.ID

	if req.Tx.State.IsFinal {
		log.Println("Withdraw called")

		err := a.Close(ctx, req.Tx.ID, req.Tx.State, req.Tx.Sigs)
		if err != nil {
			// getChanArgs, err := env.BuildGetChannelTxArgs(req.Tx.ID)
			if err != nil {
				panic(err)
			}
			chanControl, err := a.GetChannelState(ctx, req.Tx.State)
			if err != nil {
				return err
			}

			if chanControl.Control.Closed {
				return a.withdraw(ctx, req)
			}

		}
		if err != nil {
			return err
		}
		return a.withdraw(ctx, req)

	} else {
		err := a.ForceClose(ctx, req.Tx.ID, req.Tx.State, req.Tx.Sigs, req.Params)
		log.Println("ForceClose called")
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

func (a *Adjudicator) GetChannelState(ctx context.Context, state *pchannel.State) (wire.Channel, error) {

	contractAddress := a.GetPerunID()
	kp := a.kpFull
	chanId := state.ID

	// generate tx to open the channel
	getchTxArgs, err := env.BuildGetChannelTxArgs(chanId)
	if err != nil {
		return wire.Channel{}, errors.New("error while building get_channel tx")
	}
	auth := []xdr.SorobanAuthorizationEntry{}
	txMeta, err := a.stellarClient.InvokeAndProcessHostFunction("get_channel", getchTxArgs, contractAddress, kp, auth)
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

	perunAddress := a.GetPerunID()
	kp := a.kpFull
	// hzAcc := a.stellarClient.GetHorizonAcc()

	// generate tx to open the channel
	withdrawTxArgs, err := a.BuildWithdrawTxArgs(req)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	auth := []xdr.SorobanAuthorizationEntry{}
	txMeta, err := a.stellarClient.InvokeAndProcessHostFunction("withdraw", withdrawTxArgs, perunAddress, kp, auth)
	if err != nil {
		return errors.New("error while invoking and processing host function: withdraw")
	}

	_, err = DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (a *Adjudicator) Close(ctx context.Context, id pchannel.ID, state *pchannel.State, sigs []pwallet.Sig) error {
	log.Println("Close called")
	contractAddress := a.GetPerunID()
	kp := a.kpFull
	// hzAcc := a.stellarClient.GetHorizonAcc()
	closeTxArgs, err := BuildCloseTxArgs(*state, sigs)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	auth := []xdr.SorobanAuthorizationEntry{}
	txMeta, err := a.stellarClient.InvokeAndProcessHostFunction("close", closeTxArgs, contractAddress, kp, auth)
	if err != nil {
		return errors.New("error while invoking and processing host function: close")
	}
	_, err = DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}
	return nil
}

// Register registers and disputes a channel.
func (a *Adjudicator) Register(ctx context.Context, req pchannel.AdjudicatorReq, states []pchannel.SignedState) error {
	log.Println("Register called")
	sigs := req.Tx.Sigs
	state := req.Tx.State
	err := a.Dispute(ctx, state, sigs)
	if err != nil {
		return fmt.Errorf("error while disputing channel: %w", err)
	}
	return nil
}

func (a *Adjudicator) Dispute(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error {
	contractAddress := a.GetPerunID()
	kp := a.kpFull
	// hzAcc := a.stellarClient.GetHorizonAcc()
	closeTxArgs, err := BuildDisputeTxArgs(*state, sigs)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	auth := []xdr.SorobanAuthorizationEntry{}
	txMeta, err := a.stellarClient.InvokeAndProcessHostFunction("dispute", closeTxArgs, contractAddress, kp, auth)
	if err != nil {
		return errors.New("error while invoking and processing host function: dispute")
	}
	_, err = DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}
	return nil
}

func (a *Adjudicator) ForceClose(ctx context.Context, id pchannel.ID, state *pchannel.State, sigs []pwallet.Sig, params *pchannel.Params) error {
	log.Println("ForceClose called")
	contractAddress := a.GetPerunID()
	kp := a.kpFull
	forceCloseTxArgs, err := env.BuildForceCloseTxArgs(id)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	auth := []xdr.SorobanAuthorizationEntry{}
	txMeta, err := a.stellarClient.InvokeAndProcessHostFunction("force_close", forceCloseTxArgs, contractAddress, kp, auth)
	if err != nil {
		return errors.New("error while invoking and processing host function")
	}
	_, err = DecodeEventsPerun(txMeta)
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

func BuildDisputeTxArgs(state pchannel.State, sigs []pwallet.Sig) (xdr.ScVec, error) {

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
