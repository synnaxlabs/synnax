// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package deleter

import (
	"context"

	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type leaseProxy struct {
	ServiceConfig
	keyRouter proxy.BatchFactory[channel.Key]
}

func newLeaseProxy(
	cfg ServiceConfig,
) (*leaseProxy, error) {
	p := &leaseProxy{
		ServiceConfig: cfg,
		keyRouter:     proxy.BatchFactory[channel.Key]{Host: cfg.HostResolver.HostKey()},
	}
	return p, nil
}

func (lp *leaseProxy) deleteTimeRange(
	ctx context.Context,
	keys channel.Keys,
	tr telem.TimeRange,
) error {
	batch := lp.keyRouter.Batch(keys)
	sCtx, cancel := signal.WithCancel(ctx)
	defer cancel()
	for nodeKey, entries := range batch.Peers {
		sCtx.Go(func(ctx context.Context) error {
			return lp.deleteTimeRangeRemote(ctx, nodeKey, entries, tr)
		}, signal.CancelOnFail())
	}
	sCtx.Go(func(ctx context.Context) error {
		return lp.deleteTimeRangeGateway(ctx, batch.Gateway, tr)
	}, signal.CancelOnFail())
	return sCtx.Wait()
}

func (lp *leaseProxy) deleteTimeRangeRemote(
	ctx context.Context,
	target node.Key,
	keys channel.Keys,
	tr telem.TimeRange,
) error {
	addr, err := lp.HostResolver.Resolve(target)
	if errors.Is(err, aspen.ErrNodeNotFound) {
		return errors.Wrapf(ts.ErrChannelNotFound, "channel(s) %s not found", keys)
	}
	_, err = lp.Transport.Client().Send(ctx, addr, Request{Keys: keys, Bounds: tr})
	return err
}

func (lp *leaseProxy) deleteTimeRangeGateway(
	ctx context.Context,
	keys channel.Keys,
	tr telem.TimeRange,
) error {
	return lp.TSChannel.DeleteTimeRange(ctx, keys.Storage(), tr)
}
