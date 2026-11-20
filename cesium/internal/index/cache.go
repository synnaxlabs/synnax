// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

import (
	"context"

	"github.com/synnaxlabs/x/telem"
)

// Cache memoizes lookups on Domain indexes, so iterators that read channels on one
// index resolve each lookup once. A cached result can be stale after the index changes,
// so Reset the Cache before each read. A nil Cache passes each lookup to the index. A
// Cache is not safe for concurrent use.
type Cache struct {
	distances map[distanceKey]DistanceApproximation
	stamps    map[stampKey]TimeStampApproximation
	segments  map[segmentsKey][]Segment
}

type distanceKey struct {
	idx        *Domain
	tr         telem.TimeRange
	continuous bool
}

type stampKey struct {
	idx        *Domain
	ref        telem.TimeStamp
	offset     int64
	continuous bool
}

type segmentsKey struct {
	idx        *Domain
	ts         telem.TimeStamp
	start, end int64
}

// NewCache returns an empty Cache.
func NewCache() *Cache {
	return &Cache{
		distances: make(map[distanceKey]DistanceApproximation),
		stamps:    make(map[stampKey]TimeStampApproximation),
		segments:  make(map[segmentsKey][]Segment),
	}
}

// Reset removes every cached result.
func (c *Cache) Reset() {
	clear(c.distances)
	clear(c.stamps)
	clear(c.segments)
}

// Distance returns idx.Distance(ctx, tr, continuous), from the Cache when it holds the
// result.
func (c *Cache) Distance(
	ctx context.Context,
	idx *Domain,
	tr telem.TimeRange,
	continuous continuousPolicy,
) (DistanceApproximation, error) {
	if c == nil {
		return idx.Distance(ctx, tr, continuous)
	}
	key := distanceKey{idx: idx, tr: tr, continuous: continuous}
	if approx, ok := c.distances[key]; ok {
		return approx, nil
	}
	approx, err := idx.Distance(ctx, tr, continuous)
	if err == nil {
		c.distances[key] = approx
	}
	return approx, err
}

// Stamp returns idx.Stamp(ctx, ref, offset, continuous), from the Cache when it holds
// the result.
func (c *Cache) Stamp(
	ctx context.Context,
	idx *Domain,
	ref telem.TimeStamp,
	offset int64,
	continuous continuousPolicy,
) (TimeStampApproximation, error) {
	if c == nil {
		return idx.Stamp(ctx, ref, offset, continuous)
	}
	key := stampKey{idx: idx, ref: ref, offset: offset, continuous: continuous}
	if approx, ok := c.stamps[key]; ok {
		return approx, nil
	}
	approx, err := idx.Stamp(ctx, ref, offset, continuous)
	if err == nil {
		c.stamps[key] = approx
	}
	return approx, err
}

// AppendSegments returns idx.AppendSegments(ctx, dst, ts, start, end), from the Cache
// when it holds the segments.
func (c *Cache) AppendSegments(
	ctx context.Context,
	idx *Domain,
	dst []Segment,
	ts telem.TimeStamp,
	start, end int64,
) ([]Segment, error) {
	if c == nil {
		return idx.AppendSegments(ctx, dst, ts, start, end)
	}
	key := segmentsKey{idx: idx, ts: ts, start: start, end: end}
	if segments, ok := c.segments[key]; ok {
		return append(dst, segments...), nil
	}
	n := len(dst)
	dst, err := idx.AppendSegments(ctx, dst, ts, start, end)
	if err == nil {
		c.segments[key] = append([]Segment(nil), dst[n:]...)
	}
	return dst, err
}
