// Copyright 2025 PolyCrypt GmbH
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

package test

import (
	pchtest "perun.network/go-perun/channel/test"
	pwtest "perun.network/go-perun/wallet/test"

	wtest "perun.network/perun-stellar-backend/wallet/test"
	"perun.network/perun-stellar-backend/wallet/types"
)

func init() {
	pchtest.SetRandomizer(&Randomizer{}, types.StellarBackendID)
	walletRdz := wtest.Randomizer{}
	pwtest.SetRandomizer(&walletRdz, types.StellarBackendID)
}
