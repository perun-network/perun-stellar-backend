package env

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	// "github.com/stellar/go/txnbuild"
	// "log"
)

const HorizonURL = "http://localhost:8000"
const NETWORK_PASSPHRASE = "Standalone Network ; February 2017"

type HorizonMasterClient struct {
	master    *horizonclient.Client
	sourceKey *keypair.Full
}

type StellarClient struct {
	hzClient *horizonclient.Client
	kp       *keypair.Full
	// clientKey *keypair.Full
}

func NewHorizonClient() *horizonclient.Client {
	return &horizonclient.Client{HorizonURL: HorizonURL}
}

func NewStellarClient(kp *keypair.Full) *StellarClient {
	return &StellarClient{
		hzClient: NewHorizonClient(),
		kp:       kp,
	}
}

func (s *StellarClient) GetKeyPair() *keypair.Full {
	return s.kp
}

func NewHorizonMasterClient() *HorizonMasterClient {
	sourceKey := keypair.Root(NETWORK_PASSPHRASE)
	return &HorizonMasterClient{
		master:    &horizonclient.Client{HorizonURL: HorizonURL},
		sourceKey: sourceKey,
	}
}

func (m *HorizonMasterClient) GetMaster() *horizonclient.Client {
	return m.master
}

func (m *HorizonMasterClient) GetSourceKey() *keypair.Full {
	return m.sourceKey
}

func (c *StellarClient) GetHorizonClient() *horizonclient.Client {
	return c.hzClient
}

func (h *HorizonMasterClient) GetAccount(kp *keypair.Full) horizon.Account {
	accountReq := horizonclient.AccountRequest{AccountID: kp.Address()}
	hzAccount, err := h.master.AccountDetail(accountReq)
	if err != nil {
		panic(err)
	}
	return hzAccount
}

func (s *StellarClient) GetHorizonAccount(kp *keypair.Full) horizon.Account {
	accountReq := horizonclient.AccountRequest{AccountID: kp.Address()}
	hzAccount, err := s.hzClient.AccountDetail(accountReq)
	if err != nil {
		panic(err)
	}
	return hzAccount
}
