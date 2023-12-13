package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/stellar/go/keypair"
	"math/big"
	pchannel "perun.network/go-perun/channel"
	"perun.network/go-perun/client"
	"perun.network/go-perun/watcher/local"
	"perun.network/go-perun/wire"
	"perun.network/go-perun/wire/net/simple"

	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wallet"
)

type PaymentClient struct {
	perunClient *client.Client
	account     *wallet.Account
	currency    pchannel.Asset
	channels    chan *PaymentChannel
	Channel     *PaymentChannel
	wAddr       wire.Address
	balance     *big.Int
}

func SetupPaymentClient(
	stellarEnv *env.IntegrationTestEnv,
	w *wallet.EphemeralWallet, // w is the wallet used to resolve addresses to accounts for channels.
	acc *wallet.Account,
	stellarKp *keypair.Full,
	stellarTokenID *types.StellarAsset,
	bus *wire.LocalBus,

) (*PaymentClient, error) {

	// Connect to Perun pallet and get funder + adjudicator from it.

	perunConn := env.NewStellarClient(stellarEnv, stellarKp)
	funder := channel.NewFunder(acc, stellarKp, perunConn)
	adj := channel.NewAdjudicator(acc, stellarKp, perunConn)

	// Setup dispute watcher.
	watcher, err := local.NewWatcher(adj)
	if err != nil {
		return nil, fmt.Errorf("intializing watcher: %w", err)
	}

	// Setup Perun client.
	wireAddr := simple.NewAddress(acc.Address().String())
	perunClient, err := client.New(wireAddr, bus, funder, adj, w, watcher)
	if err != nil {
		return nil, errors.New("creating client")
	}

	// Create client and start request handler.
	c := &PaymentClient{
		perunClient: perunClient,
		account:     acc,
		currency:    stellarTokenID,
		channels:    make(chan *PaymentChannel, 1),
		wAddr:       wireAddr,
		balance:     big.NewInt(0),
	}

	go perunClient.Handle(c, c)
	return c, nil
}

// startWatching starts the dispute watcher for the specified channel.
func (c *PaymentClient) startWatching(ch *client.Channel) {
	go func() {
		err := ch.Watch(c)
		if err != nil {
			fmt.Printf("Watcher returned with error: %v", err)
		}
	}()
}

// OpenChannel opens a new channel with the specified peer and funding.
func (c *PaymentClient) OpenChannel(peer wire.Address, amount float64) { //*PaymentChannel
	// We define the channel participants. The proposer has always index 0. Here
	// we use the on-chain addresses as off-chain addresses, but we could also
	// use different ones.

	participants := []wire.Address{c.WireAddress(), peer}

	// We create an initial allocation which defines the starting balances.
	initBal := big.NewInt(int64(amount))

	initAlloc := pchannel.NewAllocation(2, c.currency)
	initAlloc.SetAssetBalances(c.currency, []pchannel.Bal{
		initBal, // Our initial balance.
		initBal, // Peer's initial balance.
	})

	// Prepare the channel proposal by defining the channel parameters.
	challengeDuration := uint64(10) // On-chain challenge duration in seconds.
	proposal, err := client.NewLedgerChannelProposal(
		challengeDuration,
		c.account.Address(),
		initAlloc,
		participants,
	)
	if err != nil {
		panic(err)
	}

	// Send the proposal.
	ch, err := c.perunClient.ProposeChannel(context.TODO(), proposal)
	if err != nil {
		panic(err)
	}

	// Start the on-chain event watcher. It automatically handles disputes.
	c.startWatching(ch)
	c.Channel = newPaymentChannel(ch, c.currency)
	//c.Channel.ch.OnUpdate(c.NotifyAllState)
	//c.NotifyAllState(nil, ch.State())

}

// AcceptedChannel returns the next accepted channel.
func (c *PaymentClient) AcceptedChannel() *PaymentChannel {
	return <-c.channels
}

func (p *PaymentClient) WireAddress() wire.Address {
	return p.wAddr
}

// Shutdown gracefully shuts down the client.
func (c *PaymentClient) Shutdown() {
	c.perunClient.Close()
}
