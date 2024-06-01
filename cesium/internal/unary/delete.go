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
	return db.wrapError(db.delete(ctx, tr))
}

// GarbageCollect removes unused telemetry data in the unaryDB. It is NOT safe to call
// concurrently with other GarbageCollect methods.
func (db *DB) GarbageCollect(ctx context.Context) error {
	return db.wrapError(db.Domain.GarbageCollect(ctx))
}

func (db *DB) delete(ctx context.Context, tr telem.TimeRange) error {
	if !tr.Valid() {
		return errors.Newf("delete start %d cannot be after delete end %d", tr.Start, tr.End)
	}

	g, _, err := db.Controller.OpenAbsoluteGateIfUncontrolled(
		tr,
		control.Subject{Key: uuid.NewString(), Name: "delete_writer"},
		func() (controlledWriter, error) {
			return controlledWriter{Writer: nil, channelKey: db.Channel.Key}, nil
		})
	if err != nil {
		return err
	}

	g.Authorize()
	defer g.Release()

	return db.Domain.Delete(ctx, db.calculateOffset, tr, db.Channel.DataType.Density())
}

// calculateOffset calculates the distance from a domain's start to the given time stamp.
// Additionally, it "snaps" the timestamp to the nearest sample, depending on whether
// this calculation is used to calculate the startOffset or endOffset.
//
// THIS METHOD SHOULD NOT BE CALLED BY UNARY! It should only be passed as a callback
// to Domain.Delete.
func (db *DB) calculateOffset(
	ctx context.Context,
	domainStart telem.TimeStamp,
	ts *telem.TimeStamp,
	isStart bool,
) (int64, error) {
	approxDist, err := db.index().Distance(ctx, telem.TimeRange{Start: domainStart, End: *ts}, true)
	if err != nil {
		return 0, err
	}
	var (
		offset   = approxDist.Upper
		stampEnd = offset
	)
	if isStart {
		// Note that stampEnd >= 0 since offset must be >= 1, as this function is
		// only called if domainStart != ts and therefore approxDist.Upper must be
		// at least 1.
		stampEnd = offset - 1
	}
	if !approxDist.Exact() {
		approxStamp, err := db.index().Stamp(ctx, domainStart, stampEnd, true)
		if err != nil {
			return offset, err
		}
		if !approxStamp.Exact() {
			panic("cannot find exact timestamp")
		}
		*ts = approxStamp.Upper
		if isStart {
			// If we are calculating the start timestamp, we choose the start
			// timestamp to be the last non-deleted sample + 1.
			*ts += 1
		}
	}
	return offset, nil
}
