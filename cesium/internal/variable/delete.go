// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package variable

import (
	"context"

	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

func (db *DB) Delete(ctx context.Context, tr telem.TimeRange) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	return db.wrapError(db.delete(ctx, tr))
}

func (db *DB) GarbageCollect(ctx context.Context) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	err := db.domain.GarbageCollect(ctx)
	if err == nil {
		db.offsets.invalidateAll()
	}
	return db.wrapError(err)
}

func (db *DB) delete(ctx context.Context, tr telem.TimeRange) error {
	if !tr.Valid() {
		return errors.Newf("delete start %d cannot be after delete end %d", tr.Start, tr.End)
	}
	release, err := db.lockControllerForNonWriteOp(tr, "delete")
	if err != nil {
		return err
	}
	defer release()
	err = db.domain.Delete(ctx, tr, db.calculateStartOffset, db.calculateEndOffset)
	if err == nil {
		db.offsets.invalidateAll()
	}
	return err
}

func (db *DB) calculateStartOffset(
	ctx context.Context,
	domainStart telem.TimeStamp,
	ts telem.TimeStamp,
) (telem.Size, telem.TimeStamp, error) {
	approxDist, _, err := db.index().Distance(
		ctx,
		telem.TimeRange{Start: domainStart, End: ts},
		index.MustBeContinuous,
	)
	if err != nil {
		return 0, ts, err
	}
	sampleOffset := approxDist.Upper
	if !approxDist.Exact() {
		if !approxDist.StartExact && !approxDist.EndExact {
			sampleOffset = (approxDist.Lower + approxDist.Upper) / 2
			if sampleOffset == 0 {
				return 0, ts, nil
			}
			approxStamp, err := db.index().Stamp(
				ctx,
				domainStart,
				sampleOffset-1,
				index.MustBeContinuous,
			)
			if err != nil {
				return 0, 0, err
			}
			ts = approxStamp.Upper + 1
		} else if !approxDist.StartExact {
			sampleOffset = approxDist.Lower
			approxStamp, err := db.index().Stamp(
				ctx,
				domainStart,
				sampleOffset-1,
				index.MustBeContinuous,
			)
			if err != nil {
				return 0, 0, err
			}
			ts = approxStamp.Lower + 1
		} else {
			approxStamp, err := db.index().Stamp(
				ctx,
				domainStart,
				sampleOffset-1,
				index.MustBeContinuous,
			)
			if err != nil {
				return 0, 0, err
			}
			ts = approxStamp.Upper + 1
		}
	}

	byteOffset, err := db.resolveByteOffset(ctx, domainStart, sampleOffset)
	return byteOffset, ts, err
}

func (db *DB) calculateEndOffset(
	ctx context.Context,
	domainStart telem.TimeStamp,
	ts telem.TimeStamp,
) (telem.Size, telem.TimeStamp, error) {
	approxDist, _, err := db.index().Distance(
		ctx,
		telem.TimeRange{Start: domainStart, End: ts},
		index.MustBeContinuous,
	)
	if err != nil {
		return 0, ts, err
	}
	sampleOffset := approxDist.Upper
	if !approxDist.Exact() {
		if !approxDist.StartExact && !approxDist.EndExact {
			sampleOffset = (approxDist.Lower + approxDist.Upper) / 2
			approxStamp, err := db.index().Stamp(
				ctx,
				domainStart,
				sampleOffset,
				index.MustBeContinuous,
			)
			if err != nil {
				return 0, 0, err
			}
			ts = approxStamp.Lower
		} else if !approxDist.StartExact {
			sampleOffset = approxDist.Lower
			approxStamp, err := db.index().Stamp(
				ctx,
				domainStart,
				sampleOffset,
				index.MustBeContinuous,
			)
			if err != nil {
				return 0, 0, err
			}
			ts = approxStamp.Upper
		} else {
			approxStamp, err := db.index().Stamp(
				ctx,
				domainStart,
				sampleOffset,
				index.MustBeContinuous,
			)
			if err != nil {
				return 0, 0, err
			}
			ts = approxStamp.Lower
		}
	}

	byteOffset, err := db.resolveByteOffset(ctx, domainStart, sampleOffset)
	return byteOffset, ts, err
}

func (db *DB) resolveByteOffset(
	ctx context.Context,
	domainStart telem.TimeStamp,
	sampleOffset int64,
) (telem.Size, error) {
	iter := db.domain.OpenIterator(domain.IterRange(domainStart.SpanRange(telem.TimeSpanMax)))
	defer func() { _ = iter.Close() }()
	if !iter.SeekGE(ctx, domainStart) {
		return 0, errors.Newf("cannot find domain starting at %s", domainStart)
	}
	table, err := db.getOrBuildOffsetTableForDomain(ctx, iter)
	if err != nil {
		return 0, err
	}
	if sampleOffset >= table.sampleCount {
		return telem.Size(iter.Size()), nil
	}
	return table.byteOffsetAt(sampleOffset), nil
}

func (db *DB) getOrBuildOffsetTableForDomain(ctx context.Context, iter *domain.Iterator) (*offsetTable, error) {
	domainIdx := iter.Position()
	if t, ok := db.offsets.get(domainIdx); ok {
		return t, nil
	}
	r, err := iter.OpenReader(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	t, err := buildOffsetTable(r, iter.Size())
	if err != nil {
		return nil, err
	}
	db.offsets.set(domainIdx, t)
	return t, nil
}
