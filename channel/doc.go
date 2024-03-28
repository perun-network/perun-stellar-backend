// Copyright 2024 - See NOTICE file for copyright holders.
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

// Package channel contains all relevant components to generate, utilize and conclude payment channels on the Stellar blockchain.
// These main components consist of the Funder and Adjudicator interfaces that govern the interaction between the channel users.
// Additionally, the AdjEventSub interface processes the emitted events from the Soroban smart contracts, using the interfaces from go-perun.
package channel
