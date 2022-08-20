package aspen

import (
	"context"
	"github.com/arya-analytics/aspen/internal/cluster"
	"github.com/arya-analytics/aspen/internal/kv"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/kv/pebblekv"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/pebble"
)

func Open(
	ctx context.Context,
	dirname string,
	addr address.Address,
	peers []address.Address,
	opts ...Option,
) (DB, error) {
	_ctx, shutdown := signal.WithCancel(ctx)

	o := newOptions(dirname, addr, peers, opts...)

	if err := openKV(o); err != nil {
		return nil, err
	}

	if err := configureTransport(_ctx, o); err != nil {
		return nil, err
	}

	clust, err := cluster.Join(_ctx, o.cluster)
	if err != nil {
		return nil, err
	}
	o.kv.Cluster = clust

	kve, err := kv.Open(_ctx, o.kv)
	if err != nil {
		return nil, err
	}

	return &db{Cluster: clust, DB: kve, wg: _ctx, shutdown: shutdown, options: o}, nil
}

func openKV(opts *options) error {
	if opts.kv.Engine == nil {
		pebbleDB, err := pebble.Open(opts.dirname, &pebble.Options{FS: opts.fs})
		if err != nil {
			return err
		}
		opts.kv.Engine = pebblekv.Wrap(pebbleDB)
	}
	opts.cluster.Storage = opts.kv.Engine
	return nil
}

func configureTransport(ctx signal.Context, o *options) error {
	if err := o.transport.Configure(ctx, o.addr, o.externalTransport); err != nil {
		return err
	}
	o.cluster.Gossip.Transport = o.transport.Cluster()
	o.cluster.Pledge.Transport = o.transport.Pledge()
	o.kv.OperationsTransport = o.transport.Operations()
	o.kv.LeaseTransport = o.transport.Lease()
	o.kv.FeedbackTransport = o.transport.Feedback()
	return nil
}
