package unary

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

// Delete deletes a timerange tr from the unary database by adding all the unwanted
// underlying pointers to tombstone.
//
// The start of the timerange is either in the found pointer, or before, i.e.:
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
// end of that pointer. The endOffset passed to domain will 0.
//
// Case 2 (* denotes tr.End):   |----------data-----*----|
// In this case, only data before tr.End from that pointer will be deleted, the
// endOffset passed to domain will be calculated via db.index().Distance().
func (db *DB) Delete(ctx context.Context, tr telem.TimeRange) error {
	if tr.Start.After(tr.End) {
		return errors.Newf("[cesium] delete start <%d> after delete end <%d>", tr.Start, tr.End)
	}

	var (
		startOffset int64 = 0
		endOffset   int64 = 0
		density           = db.Channel.DataType.Density()
	)

	g, _, err := db.Controller.OpenAbsoluteGateIfUncontrolled(tr, control.Subject{Key: "delete_writer"}, func() (controlledWriter, error) {
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
		return controller.Unauthorized(g.Subject.Name, db.Channel.Key)
	}
	defer g.Release()

	i := db.Domain.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRangeMax})
	if ok = i.SeekGE(ctx, tr.Start); !ok {
		// No domains after start: delete nothing.
		return i.Close()
	}

	if i.TimeRange().Start.AfterEq(tr.Start) {
		startOffset = 0
		tr.Start = i.TimeRange().Start
	} else {
		approxDist, err := db.index().Distance(ctx, telem.TimeRange{
			Start: i.TimeRange().Start,
			End:   tr.Start,
		}, false)
		if err != nil {
			return errors.CombineErrors(err, i.Close())
		}
		startOffset = approxDist.Upper
	}

	startPosition := i.Position()

	if ok = i.SeekLE(ctx, tr.End); !ok {
		// No domains before end: delete nothing.
		return i.Close()
	}

	if i.TimeRange().End.BeforeEq(tr.End) {
		tr.End = i.TimeRange().End
		endOffset = 0
	} else {
		approxDist, err := db.index().Distance(ctx, telem.TimeRange{
			Start: tr.End,
			End:   i.TimeRange().End,
		}, false)
		if err != nil {
			return errors.CombineErrors(err, i.Close())
		}

		// Add one to account for the fact that endOffset starts at the first index OUT
		// of the domain.
		endOffset = approxDist.Lower + 1
	}

	endPosition := i.Position()

	err = db.Domain.Delete(ctx, startPosition, endPosition, int64(density.Size(startOffset)), int64(density.Size(endOffset)), tr)

	if err != nil {
		return errors.CombineErrors(err, i.Close())
	}

	g.Release()
	return i.Close()
}

func (db *DB) GarbageCollect(ctx context.Context) error {
	// Check that there are no open iterators / writers on this channel.
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.mu.openIteratorWriters > 0 {
		return nil
	}

	// Check that there are no delete writers on this channel
	g, _, err := db.Controller.OpenAbsoluteGateIfUncontrolled(telem.TimeRangeMax, control.Subject{Key: "gc_writer"}, func() (controlledWriter, error) {
		return controlledWriter{
			Writer:     nil,
			channelKey: db.Channel.Key,
		}, nil
	})

	defer g.Release()

	if err != nil {
		return err
	}

	return db.Domain.CollectTombstones(ctx)
}
