// Copyright 2025 PolyCrypt GmbH
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

package test

import (
	"context"

	pchannel "perun.network/go-perun/channel"
	pkgerrors "polycry.pt/poly-go/errors"

	"perun.network/perun-stellar-backend/channel"
)

// FundAll funds all funders with the given funding requests.
func FundAll(ctx context.Context, funders []*channel.Funder, reqs []*pchannel.FundingReq) error {
	g := pkgerrors.NewGatherer()
	for i := range funders {
		i := i
		g.Go(func() error {
			return funders[i].Fund(ctx, *reqs[i])
		})
	}

	if g.WaitDoneOrFailedCtx(ctx) {
		return ctx.Err()
	}
	return g.Err()
}
