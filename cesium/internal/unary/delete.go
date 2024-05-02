package unary

import (
	"context"
	"errors"
	errors2 "github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

// Delete deletes a timerange tr from the unary database by adding all the unwanted
// underlying pointers to tombstone.
//
// The start of the timerange is either in the found pointer, or after, i.e.:
//
// Case 1 (* denotes tr.Start):   *   |---------data---------|
// In this case, that entire pointer will be deleted, and tr.Start will be set to the
// start of that pointer. The startOffset passed to domain will be 0.
//
// Case 2 (* denotes tr.Start):   |----------data-----*----|
// In this case, only data after tr.Start from that pointer will be deleted, the
// startOffset passed to domain will be calculated via db.index().Distance().
//
// The same goes for the end pointer, but in the opposite direction (pointer will be
// before or contains tr.End):
//
// Case 1 (* denotes tr.End):   |---------data---------|    *
// In this case, that entire pointer will be deleted, and tr.End will be set to the
// end of that pointer. The endOffset passed to domain will be the pointer's length.
//
// Case 2 (* denotes tr.End):   |----------data-----*----|
// In this case, only data before tr.End from that pointer will be deleted, the
// endOffset passed to domain will be calculated via db.index().Distance().
func (db *DB) Delete(ctx context.Context, tr telem.TimeRange) error {
	var (
		startOffset int64 = 0
		endOffset   int64 = 0
	)

	g, _, err := db.Controller.OpenAbsoluteGateIfUncontrolled(tr, control.Subject{Key: "Delete Writer"}, func() (controlledWriter, error) {
		return controlledWriter{
			Writer:     nil,
			channelKey: db.Channel.Key,
		}, nil
	})

	if err != nil {
		return err
	}

	_, ok := g.Authorize()
	if !ok {
		g.Release()
		return controller.Unauthorized(g.Subject.Name, db.Channel.Key)
	}

	i := db.Domain.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
	if ok := i.SeekGE(ctx, tr.Start); !ok {
		return errors2.CombineErrors(i.Close(), errors.New("[cesium] Deletion Start TS not found"))
	}

	if i.TimeRange().Start.After(tr.Start) {
		startOffset = 0
		tr.Start = i.TimeRange().Start
	} else {
		approxDist, err := db.index().Distance(ctx, telem.TimeRange{
			Start: i.TimeRange().Start,
			End:   tr.Start,
		}, false)
		if err != nil {
			return err
		}
		startOffset = approxDist.Upper
	}

	startPosition := i.Position()

	if ok := i.SeekLE(ctx, tr.End); !ok {
		return errors2.CombineErrors(i.Close(), errors.New("[cesium] Deletion End TS not found"))
	}

	if i.TimeRange().End.Before(tr.End) {
		tr.End = i.TimeRange().End
		endOffset = -1
	} else {
		approxDist, err := db.index().Distance(ctx, telem.TimeRange{
			Start: i.TimeRange().Start,
			End:   tr.End,
		}, false)
		if err != nil {
			return errors2.CombineErrors(i.Close(), err)
		}

		endOffset = approxDist.Upper
	}

	endPosition := i.Position()

	if endOffset == -1 {
		err = db.Domain.Delete(ctx, startPosition, endPosition, startOffset*int64(db.Channel.DataType.Density()), endOffset, tr)
	} else {
		err = db.Domain.Delete(ctx, startPosition, endPosition, startOffset*int64(db.Channel.DataType.Density()), endOffset*int64(db.Channel.DataType.Density()), tr)
	}

	if err != nil {
		return errors2.CombineErrors(i.Close(), err)
	}

	g.Release()
	return i.Close()
}

func (db *DB) GarbageCollect(ctx context.Context, maxSizeRead uint32) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.mu.openIteratorWriters > 0 {
		return nil
	}
	err := db.Domain.CollectTombstones(ctx, maxSizeRead)
	return err
}
