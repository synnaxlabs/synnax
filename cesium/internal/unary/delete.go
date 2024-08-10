// Copyright 2024 Synnax Labs, Inc.
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
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

// Delete deletes the specified time range from the database. Note that the start of the
// time range is inclusive whereas the end is note.
func (db *DB) Delete(ctx context.Context, tr telem.TimeRange) error {
	if db.closed.Load() {
		return errDBClosed
	}
	return db.wrapError(db.delete(ctx, tr))
}

// GarbageCollect removes unused telemetry data in the unaryDB. It is NOT safe to call
// concurrently with other GarbageCollect methods.
func (db *DB) GarbageCollect(ctx context.Context) error {
	if db.closed.Load() {
		return errDBClosed
	}
	return db.wrapError(db.domain.GarbageCollect(ctx))
}

func (db *DB) delete(ctx context.Context, tr telem.TimeRange) error {
	if !tr.Valid() {
		return errors.Newf("delete start %d cannot be after delete end %d", tr.Start, tr.End)
	}

	// Open an absolute gate to avoid deleting a time range in write.
	g, _, err := db.controller.OpenAbsoluteGateIfUncontrolled(
		tr,
		control.Subject{Key: uuid.NewString(), Name: "delete_writer"},
		func() (*controlledWriter, error) {
			return &controlledWriter{Writer: nil, channelKey: db.cfg.Channel.Key}, nil
		})
	if err != nil {
		return err
	}

	_, err = g.Authorize()
	if err != nil {
		return err
	}
	defer g.Release()

	return db.domain.Delete(
		ctx,
		db.calculateStartOffset,
		db.calculateEndOffset,
		tr,
		db.cfg.Channel.DataType.Density(),
	)
}

// calculateStartOffset calculates the distance from a domain's start to the given time stamp.
// Additionally, it "snaps" the time stamp to the nearest previous sample + 1.
// calculateOffset returns the calculated offset, the "snapped" time stamp, and any errors.
//
// **THIS METHOD SHOULD NOT BE CALLED BY UNARY!** It should only be passed as a closure
// to domain.Delete.
//
// The logic here is complicated due to the four possible cases with regard to the
// distance approximation:
//
//   - Case 1: Start of domain is exact, target ts is exact
//     This is the simplest case: the distance approximation is exact, and the
//     target timestamp does not need to be snapped to the nearest sample.
//
//   - Case 2: Start of domain is exact, target ts is inexact
//     This case is also simple: we want to delete as little as possible, so we use
//     the upper distance approximation as the distance offset and use the previous
//     sample as the end of the previous domain. For example, if the index looks
//     like: 1 3 5 7 9, the domain starts at 1, and the target is 8, we would use
//     the upper offset (4) and treat 9 as the first sample to delete, while using
//     the timestamp of the previous sample as the end of the previous domain, i.e.
//     7 * Second + 1.
//
//   - Case 3: Start of domain is inexact, target is exact
//     This case happens thanks to index cutoffs, which makes it possible that the
//     start timestamp of a domain is not an actual sample in the domain. In this
//     case, we would use the lower distance approximation instead. For example:
//     if the index is 11 13 15 17 19, but the domain starts at 9 * Second + 1,
//     the start of the domain is inexact. With a target of 17, we would use the
//     lower offset 3 as the delete offset, and use the lower timestamp approximation
//     of 15 * Second + 1.
//
//   - Case 4: Start of domain is inexact, target is inexact
//     Again use the example of 11 13 15 17 19 with the domain starting at
//     9 * Second + 1. If the target is 16, then the offset approximation is
//     (Lower: 2, Upper: 4). However, the actual offset we want is 3, therefore,
//     we must compute (Lower + Upper) / 2. As for the timestamp, we still use the
//     upper approximation for timestamp, i.e. 15 * Second + 1.
//
//     Note the edge case here: if the target timestamp is before the first sample, then
//     the offset approximation is (-1, 1), which results in an error when we stamp with
//     offset-1. Therefore, in this case, we mark the offset to 0 and do not try
//     to snap to a timestamp.
func (db *DB) calculateStartOffset(
	ctx context.Context,
	domainStart telem.TimeStamp,
	ts telem.TimeStamp,
) (int64, telem.TimeStamp, error) {
	var (
		offset      int64
		approxStamp index.TimeStampApproximation
	)

	approxDist, _, err := db.index().Distance(ctx, telem.TimeRange{Start: domainStart, End: ts}, true)
	if err != nil {
		return 0, ts, err
	}
	offset = approxDist.Upper
	if !approxDist.Exact() {
		if !approxDist.StartExact && !approxDist.EndExact {
			// If both start and end are inexact, offset is in between the two. (Note
			// that the start is only inexact because of domain cutoff).
			offset = (approxDist.Lower + approxDist.Upper) / 2
			// We stamp to offset - 1 here since if we are approximating the start offset,
			// we want to stamp the last written sample.
			if offset == 0 {
				return offset, ts, err
			}
			approxStamp, err = db.index().Stamp(ctx, domainStart, offset-1, true)
			if err != nil {
				return offset, ts, err
			}
			ts = approxStamp.Upper + 1
		} else if !approxDist.StartExact {
			// If start is inexact, we must use the lower approximation. (Note that the
			// start is only inexact because of domain cutoff).
			offset = approxDist.Lower
			approxStamp, err = db.index().Stamp(ctx, domainStart, offset-1, true)
			if err != nil {
				return offset, ts, err
			}
			ts = approxStamp.Lower + 1
		} else {
			approxStamp, err = db.index().Stamp(ctx, domainStart, offset-1, true)
			if err != nil {
				return offset, ts, err
			}
			ts = approxStamp.Upper + 1
		}
	}
	return offset, ts, nil
}

// calculateEndOffset calculates the distance from a domain's start to the given time stamp.
// Additionally, it "snaps" the time stamp to the nearest next sample.
// calculateOffset returns the calculated offset, the "snapped" time stamp, and any errors.
//
// **THIS METHOD SHOULD NOT BE CALLED BY UNARY!** It should only be passed as a closure
// to domain.Delete.
func (db *DB) calculateEndOffset(
	ctx context.Context,
	domainStart telem.TimeStamp,
	ts telem.TimeStamp,
) (int64, telem.TimeStamp, error) {
	var (
		offset      int64
		approxStamp index.TimeStampApproximation
	)

	approxDist, _, err := db.index().Distance(ctx, telem.TimeRange{Start: domainStart, End: ts}, true)
	if err != nil {
		return 0, ts, err
	}
	offset = approxDist.Upper
	if !approxDist.Exact() {
		if !approxDist.StartExact && !approxDist.EndExact {
			// If both start and end are inexact, offset is in between the two. (Note
			// that the start is only inexact because of domain cutoff).
			offset = (approxDist.Lower + approxDist.Upper) / 2
			// We stamp to offset - 1 here since if we are approximating the start offset,
			// we want to stamp the last written sample.
			approxStamp, err = db.index().Stamp(ctx, domainStart, offset, true)
			if err != nil {
				return offset, ts, err
			}
			ts = approxStamp.Lower
		} else if !approxDist.StartExact {
			// If start is inexact, we must use the lower approximation. (Note that the
			// start is only inexact because of domain cutoff).
			offset = approxDist.Lower
			approxStamp, err = db.index().Stamp(ctx, domainStart, offset, true)
			if err != nil {
				return offset, ts, err
			}
			ts = approxStamp.Upper
		} else {
			approxStamp, err = db.index().Stamp(ctx, domainStart, offset, true)
			if err != nil {
				return offset, ts, err
			}
			ts = approxStamp.Lower
		}
	}
	return offset, ts, nil
}
