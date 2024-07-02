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
	db.entityCount.add(1)
	defer db.entityCount.add(-1)
	return db.wrapError(db.Domain.GarbageCollect(ctx))
}

func (db *DB) delete(ctx context.Context, tr telem.TimeRange) error {
	if !tr.Valid() {
		return errors.Newf("delete start %d cannot be after delete end %d", tr.Start, tr.End)
	}

	// Open an absolute gate to avoid deleting a time range in write.
	g, _, err := db.Controller.OpenAbsoluteGateIfUncontrolled(
		tr,
		control.Subject{Key: uuid.NewString(), Name: "delete_writer"},
		func() (controlledWriter, error) {
			return controlledWriter{Writer: nil, channelKey: db.Channel.Key}, nil
		})
	if err != nil {
		return err
	}

	_, err = g.Authorize()
	if err != nil {
		return err
	}
	defer g.Release()

	return db.Domain.Delete(
		ctx,
		db.calculateStartOffset,
		db.calculateEndOffset,
		tr,
		db.Channel.DataType.Density(),
	)
}

// calculateStartOffset calculates the distance from a domain's start to the given time stamp.
// Additionally, it "snaps" the time stamp to the nearest previous sample + 1.
// calculateOffset returns the calculated offset, the "snapped" time stamp, and any errors.
//
// **THIS METHOD SHOULD NOT BE CALLED BY UNARY!** It should only be passed as a closure
// to Domain.Delete.
func (db *DB) calculateStartOffset(
	ctx context.Context,
	domainStart telem.TimeStamp,
	ts telem.TimeStamp,
) (int64, telem.TimeStamp, error) {
	approxDist, err := db.index().Distance(ctx, telem.TimeRange{Start: domainStart, End: ts}, true)
	if err != nil {
		return 0, ts, err
	}
	offset := approxDist.Upper
	if !approxDist.Exact() {
		// We stamp to offset - 1 here since if we are approximating the start offset,
		// we want to stamp the last written sample.
		approxStamp, err := db.index().Stamp(ctx, domainStart, offset-1, true)
		if err != nil {
			return offset, ts, err
		}
		if !approxStamp.Exact() {
			panic("cannot find exact timestamp")
		}
		ts = approxStamp.Upper + 1
	}
	return offset, ts, nil
}

// calculateStartOffset calculates the distance from a domain's start to the given time stamp.
// Additionally, it "snaps" the time stamp to the nearest next sample.
// calculateOffset returns the calculated offset, the "snapped" time stamp, and any errors.
//
// **THIS METHOD SHOULD NOT BE CALLED BY UNARY!** It should only be passed as a closure
// to Domain.Delete.
func (db *DB) calculateEndOffset(
	ctx context.Context,
	domainStart telem.TimeStamp,
	ts telem.TimeStamp,
) (int64, telem.TimeStamp, error) {
	approxDist, err := db.index().Distance(ctx, telem.TimeRange{Start: domainStart, End: ts}, true)
	if err != nil {
		return 0, ts, err
	}
	offset := approxDist.Upper
	if !approxDist.Exact() {
		approxStamp, err := db.index().Stamp(ctx, domainStart, offset, true)
		if err != nil {
			return offset, ts, err
		}
		if !approxStamp.Exact() {
			panic("cannot find exact timestamp")
		}
		ts = approxStamp.Upper
	}
	return offset, ts, nil
}
