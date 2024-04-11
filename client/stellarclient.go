package client

import (
	"context"

	"github.com/stellar/go/xdr"
	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"
	"perun.network/perun-stellar-backend/event"
	"perun.network/perun-stellar-backend/wire"
)

type StellarClient interface {
	Open(ctx context.Context, perunAddr xdr.ScAddress, params *pchannel.Params, state *pchannel.State) error
	Abort(ctx context.Context, perunAddr xdr.ScAddress, state *pchannel.State) error
	Fund(ctx context.Context, perunAddr xdr.ScAddress, assetAddr xdr.ScAddress, chanId pchannel.ID, funderIdx bool) error
	Dispute(ctx context.Context, perunAddr xdr.ScAddress, state *pchannel.State, sigs []pwallet.Sig) error
	Close(ctx context.Context, perunAddr xdr.ScAddress, state *pchannel.State, sigs []pwallet.Sig) ([]event.PerunEvent, error)
	ForceClose(ctx context.Context, perunAddr xdr.ScAddress, chanId pchannel.ID) error
	GetChannelInfo(ctx context.Context, perunAddr xdr.ScAddress, chanId pchannel.ID) (wire.Channel, error)
}
