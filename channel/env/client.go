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
	"github.com/stellar/go/txnbuild"
)

const HorizonURL = "http://localhost:8000"
const NETWORK_PASSPHRASE = "Standalone Network ; February 2017"

type StellarClient struct {
	hzClient  *horizonclient.Client
	keyHolder keyHolder
}

type keyHolder struct {
	kp *keypair.Full
}

func NewHorizonClient() *horizonclient.Client {
	return &horizonclient.Client{HorizonURL: HorizonURL}
}

func newKeyHolder(kp *keypair.Full) keyHolder {
	return keyHolder{kp}
}

func NewStellarClient(kp *keypair.Full) *StellarClient {
	return &StellarClient{
		hzClient:  NewHorizonClient(),
		keyHolder: newKeyHolder(kp),
	}
}

func NewHorizonMasterClient() *StellarClient {
	sourceKey := keypair.Root(NETWORK_PASSPHRASE)
	return &StellarClient{
		hzClient:  &horizonclient.Client{HorizonURL: HorizonURL},
		keyHolder: newKeyHolder(sourceKey),
	}
}

func (c *StellarClient) GetHorizonClient() *horizonclient.Client {
	return c.hzClient
}

func (h *StellarClient) GetAccount() (horizon.Account, error) {
	accountReq := horizonclient.AccountRequest{AccountID: h.GetAddress()}
	hzAccount, err := h.hzClient.AccountDetail(accountReq)
	if err != nil {
		return hzAccount, err
	}
	return hzAccount, nil
}

func (s *StellarClient) GetAddress() string {
	return s.keyHolder.kp.Address()
}

func (s *StellarClient) GetHorizonAccount() (horizon.Account, error) {
	accountReq := horizonclient.AccountRequest{AccountID: s.GetAddress()}
	hzAccount, err := s.hzClient.AccountDetail(accountReq)
	if err != nil {
		return hzAccount, err
	}
	return hzAccount, nil
}

func (s *StellarClient) CreateSignedTxFromParams(txParams txnbuild.TransactionParams) (*txnbuild.Transaction, error) {

	txUnsigned, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return nil, err
	}

	tx, err := txUnsigned.Sign(NETWORK_PASSPHRASE, s.keyHolder.kp)
	if err != nil {
		return nil, err
	}

	return tx, nil
}
