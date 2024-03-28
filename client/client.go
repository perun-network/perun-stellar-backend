package client

import (
	"context"
	"errors"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"log"

	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"

	"perun.network/perun-stellar-backend/event"
	"perun.network/perun-stellar-backend/wire"
)

var _ StellarClient = (*Client)(nil)

type Client struct {
	hzClient  *horizonclient.Client
	keyHolder keyHolder
}
type keyHolder struct {
	kp *keypair.Full
}

func New(kp *keypair.Full) *Client {
	return &Client{
		hzClient:  NewHorizonClient(),
		keyHolder: newKeyHolder(kp),
	}
}

func (c *Client) Open(ctx context.Context, perunAddr xdr.ScAddress, params *pchannel.Params, state *pchannel.State) error {

	openTxArgs, err := buildOpenTxArgs(*params, *state)
	if err != nil {
		return errors.New("error while building open tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("open", openTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function: open")
	}

	_, err = event.DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (c *Client) Abort(ctx context.Context, perunAddr xdr.ScAddress, state *pchannel.State) error {

	chanId := state.ID
	openTxArgs, err := buildChanIdTxArgs(chanId)
	if err != nil {
		return errors.New("error while building open tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("open", openTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function: open")
	}

	_, err = event.DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (c *Client) Fund(ctx context.Context, perunAddr xdr.ScAddress, assetAddr xdr.ScAddress, chanID pchannel.ID, fudnerIdx bool) error {

	fundTxArgs, err := buildFundTxArgs(chanID, fudnerIdx)
	if err != nil {
		return errors.New("error while building fund tx")
	}

	txMeta, err := c.InvokeAndProcessHostFunction("fund", fundTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function: fund")
	}

	_, err = event.DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (c *Client) Close(ctx context.Context, perunAddr xdr.ScAddress, state *pchannel.State, sigs []pwallet.Sig) error {

	log.Println("Close called")
	closeTxArgs, err := buildSignedStateTxArgs(*state, sigs)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("close", closeTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function: close")
	}

	_, err = event.DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}

	return nil
}

func (c *Client) ForceClose(ctx context.Context, perunAddr xdr.ScAddress, chanId pchannel.ID) error {
	log.Println("ForceClose called")

	forceCloseTxArgs, err := buildChanIdTxArgs(chanId)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("force_close", forceCloseTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function")
	}
	_, err = event.DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}
	return nil
}

func (c *Client) Dispute(ctx context.Context, perunAddr xdr.ScAddress, state *pchannel.State, sigs []pwallet.Sig) error {
	closeTxArgs, err := buildSignedStateTxArgs(*state, sigs)
	if err != nil {
		return errors.New("error while building fund tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("dispute", closeTxArgs, perunAddr)
	if err != nil {
		return errors.New("error while invoking and processing host function: dispute")
	}
	_, err = event.DecodeEventsPerun(txMeta)
	if err != nil {
		return errors.New("error while decoding events")
	}
	return nil
}

func (c *Client) GetChannelInfo(ctx context.Context, perunAddr xdr.ScAddress, chanId pchannel.ID) (wire.Channel, error) {

	getchTxArgs, err := buildChanIdTxArgs(chanId)
	if err != nil {
		return wire.Channel{}, errors.New("error while building get_channel tx")
	}
	txMeta, err := c.InvokeAndProcessHostFunction("get_channel", getchTxArgs, perunAddr)
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
