package client

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
)

const HorizonURL = "http://localhost:8000"
const NETWORK_PASSPHRASE = "Standalone Network ; February 2017"
const HorizonURLTestNet = "https://horizon-testnet.stellar.org"
const NETWORK_PASSPHRASETestNet = "Test SDF Network ; September 2015"

func NewHorizonClient(url string) *horizonclient.Client {
	return &horizonclient.Client{HorizonURL: url}
}

func newKeyHolder(kp *keypair.Full) keyHolder {
	return keyHolder{kp}
}

func NewHorizonMasterClient(passphrase string, url string) *Client {
	sourceKey := keypair.Root(passphrase)
	return &Client{
		hzClient:  &horizonclient.Client{HorizonURL: url},
		keyHolder: newKeyHolder(sourceKey),
	}
}

func (c *Client) GetHorizonClient() *horizonclient.Client {
	return c.hzClient
}

func (c *Client) GetAddress() string {
	return c.keyHolder.kp.Address()
}
