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
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

type Deleter interface {
	DeleteTimeRange(ctx context.Context, key channel.Key, tr telem.TimeRange) error
	DeleteTimeRangeByName(ctx context.Context, name string, tr telem.TimeRange) error
	DeleteTimeRangeMany(ctx context.Context, keys []channel.Key, tr telem.TimeRange) error
	DeleteTimeRangeManyByNames(ctx context.Context, name []string, tr telem.TimeRange) error
}

type deleter struct {
	proxy *leaseProxy
	tx    gorp.Tx
}

var _ Deleter = deleter{}

func (d deleter) DeleteTimeRange(
	ctx context.Context,
	key channel.Key,
	tr telem.TimeRange,
) error {
	return d.DeleteTimeRangeMany(ctx, []channel.Key{key}, tr)
}

func (d deleter) DeleteTimeRangeByName(
	ctx context.Context,
	name string,
	tr telem.TimeRange,
) error {
	return d.DeleteTimeRangeManyByNames(ctx, []string{name}, tr)
}

func (d deleter) DeleteTimeRangeMany(
	ctx context.Context,
	keys []channel.Key,
	tr telem.TimeRange,
) error {
	return d.proxy.deleteTimeRange(ctx, d.tx, keys, tr)
}

func (d deleter) DeleteTimeRangeManyByNames(
	ctx context.Context,
	names []string,
	tr telem.TimeRange,
) error {
	return d.proxy.deleteTimeRangeByName(ctx, d.tx, names, tr)
}
