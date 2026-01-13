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

	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/errors"
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
	for nodeKey, entries := range batch.Peers {
		err := lp.deleteTimeRangeRemote(ctx, nodeKey, entries, tr)
		if err != nil {
			return err
		}
	}

	return lp.deleteTimeRangeGateway(ctx, batch.Gateway, tr)
}

func (lp *leaseProxy) deleteTimeRangeByName(
	ctx context.Context,
	channelSvc *channel.Service,
	names []string,
	tr telem.TimeRange,
) error {
	res := make([]channel.Channel, 0, len(names))
	if err := channelSvc.
		NewRetrieve().
		Entries(&res).
		WhereNames(names...).
		Exec(ctx, nil); err != nil {
		return err
	}

	resultNames := lo.Map(res, func(item channel.Channel, _ int) string { return item.Name })
	if len(lo.Uniq(resultNames)) < len(names) {
		_, diff := lo.Difference(names, resultNames)
		return errors.Wrapf(ts.ErrChannelNotfound, "channel(s) %s not found", diff)
	}

	keys := channel.KeysFromChannels(res)
	return lp.deleteTimeRange(ctx, keys, tr)
}

func (lp *leaseProxy) deleteTimeRangeRemote(
	ctx context.Context,
	target cluster.NodeKey,
	keys channel.Keys,
	tr telem.TimeRange,
) error {
	addr, err := lp.HostResolver.Resolve(target)
	if errors.Is(err, aspen.ErrNodeNotFound) {
		return errors.Wrapf(ts.ErrChannelNotfound, "channel(s) %s not found", keys)
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
