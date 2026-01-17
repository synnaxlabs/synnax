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

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/telem"
)

type Deleter struct {
	proxy   *leaseProxy
	channel *channel.Service
}

// DeleteTimeRange deletes a time range in the specified channel. It is idempotent:
// if no data is found in the range, nil is returned. However, if the channel
// does not exist, an ErrChannelNotfound is returned.
func (d Deleter) DeleteTimeRange(
	ctx context.Context,
	key channel.Key,
	tr telem.TimeRange,
) error {
	return d.DeleteTimeRangeMany(ctx, []channel.Key{key}, tr)
}

// DeleteTimeRangeByName deletes a time range in the specified channel. It is idempotent:
// if no data is found in the range, nil is returned. However, if the channel
// does not exist, a query.ErrNotFound is returned.
// All channels with the provided name are affected.
func (d Deleter) DeleteTimeRangeByName(
	ctx context.Context,
	name string,
	tr telem.TimeRange,
) error {
	return d.DeleteTimeRangeManyByNames(ctx, []string{name}, tr)
}

// DeleteTimeRangeMany deletes a time range in the specified channels. It is idempotent:
// if no data is found in the range, that channel is skipped.
//
// It is NOT atomic: if any deletion fails after others have succeeded, the operation
// is abandoned midway.
//
// However, if any channel is not found by its name, the operation is abandoned before
// any data is deleted.
func (d Deleter) DeleteTimeRangeMany(
	ctx context.Context,
	keys []channel.Key,
	tr telem.TimeRange,
) error {
	return d.proxy.deleteTimeRange(ctx, keys, tr)
}

// DeleteTimeRangeManyByNames deletes a time range in the specified channels.
// It is idempotent: if no data is found in the range, that channel is skipped.
//
// It is NOT atomic: if any deletion fails after others have succeeded, the operation
// is abandoned midway.
//
// However, if any one channel is not found by its name, the operation is abandoned
// before any data is deleted.
// All channels with the provided name are affected.
func (d Deleter) DeleteTimeRangeManyByNames(
	ctx context.Context,
	names []string,
	tr telem.TimeRange,
) error {
	return d.proxy.deleteTimeRangeByName(ctx, d.channel, names, tr)
}
