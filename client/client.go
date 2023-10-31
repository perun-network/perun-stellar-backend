package client

import (
	"fmt"
	"github.com/stellar/go/keypair"
	"math/big"
	"perun.network/go-perun/wire/net/simple"

	"errors"
	pchannel "perun.network/go-perun/channel"
	"perun.network/go-perun/client"
	"perun.network/go-perun/watcher/local"
	"perun.network/go-perun/wire"

	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/types"

	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/wallet"
)

type PaymentClient struct {
	perunClient *client.Client
	account     *wallet.Account
	currency    pchannel.Asset
	channels    chan *PaymentChannel
	Channel     *PaymentChannel
	//ICConn      *chanconn.Connector
	wAddr   wire.Address
	balance *big.Int
}

func SetupPaymentClient(
	stellarEnv *env.IntegrationTestEnv,
	w *wallet.EphemeralWallet, // w is the wallet used to resolve addresses to accounts for channels.
	acc *wallet.Account,
	stellarKp *keypair.Full,
	stellarTokenID types.StellarAsset,
	//acc *wallet.Account,
	bus *wire.LocalBus,

) (*PaymentClient, error) {

	// Connect to Perun pallet and get funder + adjudicator from it.

	perunConn := env.NewStellarClient(stellarEnv, stellarKp)

	funder := channel.NewFunder(acc, perunConn)
	adj := channel.NewAdjudicator(acc, perunConn)

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
	//asset := types.NewStellarAsset(big.NewInt(int64(chainID)), common.Address(assetaddr))

	// Create client and start request handler.
	c := &PaymentClient{
		perunClient: perunClient,
		account:     acc,
		currency:    &stellarTokenID,
		channels:    make(chan *PaymentChannel, 1),
		//ICConn:      perunConn,
		wAddr:   wireAddr,
		balance: big.NewInt(0),
	}

	//go c.PollBalances()
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

// func SetupPaymentClient(
// 	name string,
// 	w *wallet.FsWallet, // w is the wallet used to resolve addresses to accounts for channels.
// 	bus *wire.LocalBus,
// 	perunID string,
// 	ledgerID string,
// 	host string,
// 	port int,
// 	accountPath string,
// ) (*PaymentClient, error) {

// 	acc := w.NewAccount()

// 	// Connect to Perun pallet and get funder + adjudicator from it.

// 	perunConn := chanconn.NewICConnector(perunID, ledgerID, accountPath, host, port)

// 	funder := channel.NewFunder(acc, perunConn)
// 	adj := channel.NewAdjudicator(acc, perunConn)

// 	// Setup dispute watcher.
// 	watcher, err := local.NewWatcher(adj)
// 	if err != nil {
// 		return nil, fmt.Errorf("intializing watcher: %w", err)
// 	}

// 	// Setup Perun client.
// 	wireAddr := simple.NewAddress(acc.Address().String())
// 	perunClient, err := client.New(wireAddr, bus, funder, adj, w, watcher)
// 	if err != nil {
// 		return nil, errors.WithMessage(err, "creating client")
// 	}

// 	// Create client and start request handler.
// 	c := &PaymentClient{
// 		Name:        name,
// 		perunClient: perunClient,
// 		account:     &acc,
// 		currency:    channel.Asset,
// 		channels:    make(chan *PaymentChannel, 1),
// 		ICConn:      perunConn,
// 		wAddr:       wireAddr,
// 		balance:     big.NewInt(0),
// 	}

// 	go c.PollBalances()
// 	go perunClient.Handle(c, c)
// 	return c, nil
// }

// // SetupPaymentClient creates a new payment client.
// func SetupPaymentClient(
// 	bus wire.Bus, // bus is used of off-chain communication.
// 	w *swallet.Wallet, // w is the wallet used for signing transactions.
// 	acc common.Address, // acc is the address of the account to be used for signing transactions.
// 	eaddress *ethwallet.Address, // eaddress is the address of the Ethereum account to be used for signing transactions.
// 	nodeURL string, // nodeURL is the URL of the blockchain node.
// 	chainID uint64, // chainID is the identifier of the blockchain.
// 	adjudicator common.Address, // adjudicator is the address of the adjudicator.
// 	assetaddr ethwallet.Address, // asset is the address of the asset holder for our payment channels.
// ) (*PaymentClient, error) {
// 	// Create Ethereum client and contract backend.
// 	cb, err := CreateContractBackend(nodeURL, chainID, w)
// 	if err != nil {
// 		return nil, fmt.Errorf("creating contract backend: %w", err)
// 	}

// 	// Validate contracts.
// 	err = ethchannel.ValidateAdjudicator(context.TODO(), cb, adjudicator)
// 	if err != nil {
// 		return nil, fmt.Errorf("validating adjudicator: %w", err)
// 	}
// 	err = ethchannel.ValidateAssetHolderETH(context.TODO(), cb, common.Address(assetaddr), adjudicator)
// 	if err != nil {
// 		return nil, fmt.Errorf("validating adjudicator: %w", err)
// 	}

// 	// Setup funder.
// 	funder := ethchannel.NewFunder(cb)
// 	dep := ethchannel.NewETHDepositor()
// 	ethAcc := accounts.Account{Address: acc}
// 	asset := ethchannel.NewAsset(big.NewInt(int64(chainID)), common.Address(assetaddr))
// 	funder.RegisterAsset(*asset, dep, ethAcc)

// 	// Setup adjudicator.
// 	adj := ethchannel.NewAdjudicator(cb, adjudicator, acc, ethAcc)

// 	// Setup dispute watcher.
// 	watcher, err := local.NewWatcher(adj)
// 	if err != nil {
// 		return nil, fmt.Errorf("intializing watcher: %w", err)
// 	}

// 	// Setup Perun client.
// 	waddr := &ethwire.Address{Address: eaddress}
// 	perunClient, err := client.New(waddr, bus, funder, adj, w, watcher)
// 	if err != nil {
// 		return nil, errors.WithMessage(err, "creating client")
// 	}

// 	// Create client and start request handler.
// 	c := &PaymentClient{
// 		perunClient: perunClient,
// 		account:     eaddress,
// 		waddress:    waddr,
// 		currency:    asset,
// 		channels:    make(chan *PaymentChannel, 1),
// 	}
// 	go perunClient.Handle(c, c)

// 	return c, nil
// }
