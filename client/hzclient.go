package client

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
)

const HorizonURL = "https://horizon-testnet.stellar.org"
const NETWORK_PASSPHRASE = "Test SDF Network ; September 2015"

func NewHorizonClient() *horizonclient.Client {
	return &horizonclient.Client{HorizonURL: HorizonURL}
}

func newKeyHolder(kp *keypair.Full) keyHolder {
	return keyHolder{kp}
}

func NewHorizonMasterClient() *Client {
	sourceKey := keypair.Root(NETWORK_PASSPHRASE)
	return &Client{
		hzClient:  &horizonclient.Client{HorizonURL: HorizonURL},
		keyHolder: newKeyHolder(sourceKey),
	}
}

func (c *Client) GetHorizonClient() *horizonclient.Client {
	return c.hzClient
}

func (c *Client) GetAddress() string {
	return c.keyHolder.kp.Address()
}
