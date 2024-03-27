package client

import (
	"context"
	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/wire"
)

type Client interface {
	Open(ctx context.Context, params pchannel.Params, state *pchannel.State) error
	Abort(ctx context.Context, state *pchannel.State) error
	Fund(ctx context.Context, state *pchannel.State) error
	Dispute(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error
	Close(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error
	ForceClose(ctx context.Context, state *pchannel.State, sigs []pwallet.Sig) error
	GetChannelInfo(ctx context.Context, state *pchannel.State) (wire.Channel, error)
}
