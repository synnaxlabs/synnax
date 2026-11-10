// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary

import (
	"context"

	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// TruncateToIndex truncates each domain to the samples that its index has timestamps
// for, and logs each truncation. A crash between the saves of a channel and its index
// leaves such domains. TruncateToIndex must run before any writer or iterator opens on
// the DB. It does nothing for an index channel.
func (db *DB) TruncateToIndex(ctx context.Context) error {
	if db.cfg.Channel.IsIndex {
		return nil
	}
	cuts, err := db.findCuts(ctx)
	if err != nil {
		return db.wrapError(err)
	}
	for _, c := range cuts {
		cutStart := func(
			context.Context,
			telem.TimeStamp,
			telem.TimeStamp,
		) (telem.Size, telem.TimeStamp, error) {
			return c.offset, c.start, nil
		}
		// Delete calls the end resolver only for a domain that starts where the cut
		// ends. That domain stays whole.
		keepEnd := func(
			_ context.Context,
			_ telem.TimeStamp,
			ts telem.TimeStamp,
		) (telem.Size, telem.TimeStamp, error) {
			return 0, ts, nil
		}
		if err = db.domain.Delete(
			ctx,
			c.start.Range(c.domain.End),
			cutStart,
			keepEnd,
		); err != nil {
			return db.wrapError(errors.Wrapf(err, "domain %v", c.domain))
		}
		db.cfg.L.Warn(
			"truncated a domain to the samples its index has timestamps for",
			zap.Stringer("channel", db.cfg.Channel),
			zap.Stringer("domain", c.domain),
			zap.Int64("dropped_samples", c.dropped),
		)
	}
	db.resolver.invalidate()
	return nil
}

// cut is the tail of a domain that its index has no timestamps for.
type cut struct {
	domain  telem.TimeRange
	start   telem.TimeStamp
	offset  telem.Size
	dropped int64
}

func (db *DB) findCuts(ctx context.Context) (cuts []cut, err error) {
	iter := db.domain.OpenIterator(domain.IterRange(telem.TimeRangeMax))
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for ok := iter.SeekFirst(ctx); ok; ok = iter.Next() {
		var (
			c     cut
			found bool
		)
		if c, found, err = db.findCut(ctx, iter); err != nil {
			return nil, errors.Wrapf(err, "domain %v", iter.TimeRange())
		}
		if found {
			cuts = append(cuts, c)
		}
	}
	return cuts, nil
}

func (db *DB) findCut(ctx context.Context, iter *domain.Iterator) (cut, bool, error) {
	tr := iter.TimeRange()
	end, covered, err := db.index().Coverage(ctx, tr)
	if err != nil {
		return cut{}, false, err
	}
	// A domain ends 1 ns after its last sample, so a domain that ends inside the run of
	// its index has a timestamp for every sample.
	if tr.End <= end {
		return cut{}, false, nil
	}
	count, err := db.resolver.domainSampleCount(ctx, iter)
	if err != nil || count <= covered {
		return cut{}, false, err
	}
	offset, err := db.resolver.byteOffset(ctx, iter, covered)
	if err != nil {
		return cut{}, false, err
	}
	return cut{
		domain:  tr,
		start:   max(end, tr.Start),
		offset:  offset,
		dropped: count - covered,
	}, true, nil
}
