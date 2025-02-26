package test

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	pchannel "perun.network/go-perun/channel"
	pclient "perun.network/go-perun/client"
	pwallet "perun.network/go-perun/wallet"
	"perun.network/go-perun/watcher/local"
	pwire "perun.network/go-perun/wire"

	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/wallet"
	"perun.network/perun-stellar-backend/wire"
)

const StellarBackendID pwallet.BackendID = 2

type PaymentClient struct {
	perunClient *pclient.Client
	account     *wallet.Account
	currencies  []pchannel.Asset
	channels    chan *PaymentChannel
	Channel     *PaymentChannel
	wAddr       pwire.Address
	balance     *big.Int
}

func SetupPaymentClient(
	w *wallet.EphemeralWallet,
	acc *wallet.Account,
	stellarTokenIDs []pchannel.Asset,
	bus *pwire.LocalBus,
	funder *channel.Funder,
	adj *channel.Adjudicator,
) (*PaymentClient, error) {
	watcher, err := local.NewWatcher(adj)
	if err != nil {
		return nil, fmt.Errorf("intializing watcher: %w", err)
	}
	// Setup Perun client.
	wireAddr := &wire.WirePart{Participant: acc.Participant()}
	wireBackendAddrs := map[pwallet.BackendID]pwire.Address{
		StellarBackendID: wireAddr,
	}
	walletMap := map[pwallet.BackendID]pwallet.Wallet{
		StellarBackendID: w,
	}

	perunClient, err := pclient.New(wireBackendAddrs, bus, funder, adj, walletMap, watcher)
	if err != nil {
		return nil, errors.New("creating client")
	}

	c := &PaymentClient{
		perunClient: perunClient,
		account:     acc,
		currencies:  stellarTokenIDs,
		channels:    make(chan *PaymentChannel, 1),
		wAddr:       wireAddr,
		balance:     big.NewInt(0),
	}

	go perunClient.Handle(c, c)
	return c, nil
}

// startWatching starts the dispute watcher for the specified channel.
func (c *PaymentClient) startWatching(ch *pclient.Channel) {
	go func() {
		err := ch.Watch(c)
		if err != nil {
			fmt.Printf("Watcher returned with error: %v", err)
		}
	}()
}

func (c *PaymentClient) OpenChannel(peer pwire.Address, balances pchannel.Balances) {
	// We define the channel participants. The proposer has always index 0. Here
	// we use the on-chain addresses as off-chain addresses, but we could also
	// use different ones.
	proposerAddr := map[pwallet.BackendID]pwire.Address{
		StellarBackendID: c.WireAddress(),
	}

	proposerWalletAddr := map[pwallet.BackendID]pwallet.Address{
		StellarBackendID: c.account.Address(),
	}

	peerAddr := map[pwallet.BackendID]pwire.Address{
		StellarBackendID: peer,
	}
	backends := make([]pwallet.BackendID, len(c.currencies))
	for i := range c.currencies {
		backends[i] = StellarBackendID
	}
	participants := []map[pwallet.BackendID]pwire.Address{proposerAddr, peerAddr}
	initAlloc := pchannel.NewAllocation(2, backends, c.currencies...) //nolint:gomnd
	initAlloc.Balances = balances
	// Prepare the channel proposal by defining the channel parameters.
	challengeDuration := uint64(10) //nolint:gomnd
	proposal, err := pclient.NewLedgerChannelProposal(
		challengeDuration,
		proposerWalletAddr,
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
	c.Channel = newPaymentChannel(ch, c.currencies)
}

func (c *PaymentClient) WireAddress() pwire.Address {
	return c.wAddr
}

func (c *PaymentClient) AcceptedChannel() *PaymentChannel {
	return <-c.channels
}

// Shutdown gracefully shuts down the client.
func (c *PaymentClient) Shutdown() {
	c.perunClient.Close()
}
