// Copyright 2023 PolyCrypt GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package env

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
)

const HorizonURL = "http://localhost:8000"
const NETWORK_PASSPHRASE = "Standalone Network ; February 2017"

type StellarClient struct {
	hzClient *horizonclient.Client
	kp       *keypair.Full
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

func NewHorizonMasterClient() *StellarClient {
	sourceKey := keypair.Root(NETWORK_PASSPHRASE)
	return &StellarClient{
		hzClient: &horizonclient.Client{HorizonURL: HorizonURL},
		kp:       sourceKey,
	}
}

func (m *StellarClient) GetMaster() *horizonclient.Client {
	return m.hzClient
}

func (m *StellarClient) GetSourceKey() *keypair.Full {
	return m.kp
}

func (c *StellarClient) GetHorizonClient() *horizonclient.Client {
	return c.hzClient
}

func (h *StellarClient) GetAccount(kp *keypair.Full) (horizon.Account, error) {
	accountReq := horizonclient.AccountRequest{AccountID: kp.Address()}
	hzAccount, err := h.hzClient.AccountDetail(accountReq)
	if err != nil {
		return hzAccount, err
	}
	return hzAccount, nil
}

func (s *StellarClient) GetHorizonAccount(kp *keypair.Full) (horizon.Account, error) {
	accountReq := horizonclient.AccountRequest{AccountID: kp.Address()}
	hzAccount, err := s.hzClient.AccountDetail(accountReq)
	if err != nil {
		return hzAccount, err
	}
	return hzAccount, nil
}
