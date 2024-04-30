//go:build integration
// +build integration

// Copyright 2024 PolyCrypt GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package payment_test

import (
	"log"
	"perun.network/go-perun/wire"
	chtest "perun.network/perun-stellar-backend/channel/test"
	paytest "perun.network/perun-stellar-backend/payment/test"
	"testing"
)

func TestHappyPerun(t *testing.T) {
	t.Run("Honest Perun Users", func(t *testing.T) {
		runHappyPerun(t)
	})
}

func runHappyPerun(t *testing.T) {
	set := chtest.NewTestSetup(t)
	accAlice := set.GetAccounts()[0]
	accBob := set.GetAccounts()[1]
	wAlice := set.GetWallets()[0]
	wBob := set.GetWallets()[1]
	funderAlice := set.GetFunders()[0]
	funderBob := set.GetFunders()[1]
	adjAlice := set.GetAdjudicators()[0]
	adjBob := set.GetAdjudicators()[1]

	bus := wire.NewLocalBus()
	alicePerun, err := paytest.SetupPaymentClient(wAlice, accAlice, set.GetTokenAsset(), bus, funderAlice, adjAlice)
	if err != nil {
		panic(err)
	}
	bobPerun, err := paytest.SetupPaymentClient(wBob, accBob, set.GetTokenAsset(), bus, funderBob, adjBob)
	if err != nil {
		panic(err)
	}

	alicePerun.OpenChannel(bobPerun.WireAddress(), 1000)
	aliceChannel := alicePerun.Channel
	bobChannel := bobPerun.AcceptedChannel()

	aliceChannel.SendPayment(10)
	bobChannel.SendPayment(50)

	aliceChannel.Settle()
	bobChannel.Settle()

	alicePerun.Shutdown()
	bobPerun.Shutdown()

	log.Println("Done")

}
