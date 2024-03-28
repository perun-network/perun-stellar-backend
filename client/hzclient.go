package client

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
)

const HorizonURL = "http://localhost:8000"
const NETWORK_PASSPHRASE = "Standalone Network ; February 2017"

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

func (c *Client) GetAccount() (horizon.Account, error) {
	accountReq := horizonclient.AccountRequest{AccountID: c.GetAddress()}
	hzAccount, err := c.hzClient.AccountDetail(accountReq)
	if err != nil {
		return hzAccount, err
	}
	return hzAccount, nil
}

func (c *Client) GetAddress() string {
	return c.keyHolder.kp.Address()
}

func (c *Client) GetHorizonAccount() (horizon.Account, error) {
	accountReq := horizonclient.AccountRequest{AccountID: c.GetAddress()}
	hzAccount, err := c.hzClient.AccountDetail(accountReq)
	if err != nil {
		return hzAccount, err
	}
	return hzAccount, nil
}
