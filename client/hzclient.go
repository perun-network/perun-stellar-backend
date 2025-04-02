package client

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
)

const (
	HorizonURL                = "http://localhost:8000"
	NETWORK_PASSPHRASE        = "Standalone Network ; February 2017"  //nolint:golint,stylecheck
	HorizonURLTestNet         = "https://horizon-testnet.stellar.org" //nolint:golint,stylecheck
	NETWORK_PASSPHRASETestNet = "Test SDF Network ; September 2015"   //nolint:golint,stylecheck
)

// NewHorizonClient creates a new horizon client.
func NewHorizonClient(url string) *horizonclient.Client {
	return &horizonclient.Client{HorizonURL: url}
}

func newKeyHolder(kp *keypair.Full) keyHolder {
	return keyHolder{kp}
}

// NewHorizonMasterClient creates a new horizon client with a master keypair.
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
