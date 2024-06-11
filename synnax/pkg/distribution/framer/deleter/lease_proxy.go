// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/telem"

	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
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
	retriever channel.Readable,
	names []string,
	tr telem.TimeRange,
) error {
	var res []channel.Channel
	retriever.NewRetrieve().WhereKeys()
	if err := retriever.
		NewRetrieve().
		Entries(&res).
		WhereNames(names...).
		Exec(ctx, nil); err != nil {
		return err
	}

	keys := channel.KeysFromChannels(res)
	return lp.deleteTimeRange(ctx, keys, tr)
}

func (lp *leaseProxy) deleteTimeRangeRemote(
	ctx context.Context,
	target core.NodeKey,
	keys channel.Keys,
	tr telem.TimeRange,
) error {
	addr, err := lp.HostResolver.Resolve(target)
	if err != nil {
		return err
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
