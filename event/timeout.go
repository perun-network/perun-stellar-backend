// Copyright 2024 PolyCrypt GmbH
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

package event

import (
	pchannel "perun.network/go-perun/channel"
	"time"
)

// DefaultTimeoutPollInterval default value for the PollInterval of a Timeout.
const DefaultTimeoutPollInterval = 1 * time.Second

// NewTimeTimeout returns a new Timeout which expires at the given time.
func NewTimeTimeout(when time.Time) pchannel.Timeout {
	return &pchannel.TimeTimeout{Time: when}
}

// MakeTimeout creates a new timeout.
func MakeTimeout(challDurSec uint64) pchannel.Timeout {
	expirationTime := time.Now().Add(MakeTime(challDurSec))
	return NewTimeTimeout(expirationTime)
}

// MakeTime creates a new time from the argument.
func MakeTime(challDurSec uint64) time.Duration {
	return time.Duration(challDurSec) * time.Second
}
