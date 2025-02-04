// Copyright 2025 Synnax Labs, Inc.
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
)

type Deleter interface {
	// DeleteTimeRange deletes a time range in the specified channel. It is idempotent:
	// if no data is found in the range, nil is returned. However, if the channel
	// does not exist, an ErrChannelNotfound is returned.
	DeleteTimeRange(ctx context.Context, key channel.Key, tr telem.TimeRange) error
	// DeleteTimeRangeByName deletes a time range in the specified channel. It is idempotent:
	// if no data is found in the range, nil is returned. However, if the channel
	// does not exist, a query.NotFound is returned.
	// All channels with the provided name are affected.
	DeleteTimeRangeByName(ctx context.Context, name string, tr telem.TimeRange) error
	// DeleteTimeRangeMany deletes a time range in the specified channels. It is idempotent:
	// if no data is found in the range, that channel is skipped.
	// It is NOT atomic: if any deletion fails after others have succeeded, the operation
	// is abandoned midway.
	// However, if any channel is not found by its name, the operation is abandoned before
	// any data is deleted.
	DeleteTimeRangeMany(ctx context.Context, keys []channel.Key, tr telem.TimeRange) error
	// DeleteTimeRangeManyByNames deletes a time range in the specified channels.
	// It is idempotent: if no data is found in the range, that channel is skipped.
	// It is NOT atomic: if any deletion fails after others have succeeded, the operation
	// is abandoned midway.
	// However, if any one channel is not found by its name, the operation is abandoned
	// before any data is deleted.
	// All channels with the provided name are affected.
	DeleteTimeRangeManyByNames(ctx context.Context, name []string, tr telem.TimeRange) error
}

type deleter struct {
	proxy         *leaseProxy
	channelReader channel.Readable
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
	return d.proxy.deleteTimeRange(ctx, keys, tr)
}

func (d deleter) DeleteTimeRangeManyByNames(
	ctx context.Context,
	names []string,
	tr telem.TimeRange,
) error {
	return d.proxy.deleteTimeRangeByName(ctx, d.channelReader, names, tr)
}
