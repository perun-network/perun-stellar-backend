package test

import (
	"context"

	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
)

// PaymentChannel represents a payment channel.
type PaymentChannel struct {
	ch         *client.Channel
	currencies []channel.Asset
}

// GetChannel returns the channel.
func (c *PaymentChannel) GetChannel() *client.Channel {
	return c.ch
}

// GetChannelParams returns the channel parameters.
func (c *PaymentChannel) GetChannelParams() *channel.Params {
	return c.ch.Params()
}

// GetChannelState returns the channel state.
func (c *PaymentChannel) GetChannelState() *channel.State {
	return c.ch.State()
}

func newPaymentChannel(ch *client.Channel, currencies []channel.Asset) *PaymentChannel {
	return &PaymentChannel{
		ch:         ch,
		currencies: currencies,
	}
}

// PerformSwap performs a swap by "swapping" the balances of the two
// participants for both assets.
func (c PaymentChannel) PerformSwap() {
	err := c.ch.Update(context.TODO(), func(state *channel.State) { // We use context.TODO to keep the code simple.
		// We simply swap the balances for the two assets.
		state.Balances = channel.Balances{
			{state.Balances[0][1], state.Balances[0][0]},
			{state.Balances[1][1], state.Balances[1][0]},
		}

		// Set the state to final because we do not expect any other updates
		// than this swap.
		state.IsFinal = true
	})
	if err != nil {
		panic(err) // We panic on error to keep the code simple.
	}
}

// Settle settles the payment channel and withdraws the funds.
func (c PaymentChannel) Settle() {
	// If the channel is not finalized: Finalize the channel to enable fast settlement.

	if !c.ch.State().IsFinal {
		err := c.ch.Update(context.TODO(), func(state *channel.State) {
			state.IsFinal = true
		})
		if err != nil {
			panic(err)
		}
	}

	// Settle concludes the channel and withdraws the funds.
	err := c.ch.Settle(context.TODO(), false)
	if err != nil {
		panic(err)
	}

	// Close frees up channel resources.
	c.ch.Close()
}
