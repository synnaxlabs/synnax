package cesium

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/allocate"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/kv"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// WriteRequest is a request containing a set of segments (segment) to write to the DB.
type WriteRequest struct {
	Commit bool
	// Segments is a set of segments to write to the DB.
	Segments []Segment
}

// WriteResponse contains any errors that occurred during the execution of the write.
type WriteResponse struct {
	// Err is any err that occurred during internal execution.
	Err error
}

type StreamWriter = confluence.Segment[WriteRequest, WriteResponse]

// Writer writes segmented telemetry to the DB. Writer must be closed after use.
// Writer is not goroutine-safe, but it is safe to use multiple writers for different
// channels concurrently.
//
// Writer is asynchronous, meaning that calls to Write will return before segments are
// persisted to disk. The caller can guarantee that all segments have been persisted
// by waiting on Close to return.
type Writer interface {
	// Write writes the provided segments to the DB. If the Writer has encountered an
	// operational err, this method will return false, and the caller is expected
	// to close the Writer. Before Write returns false, subsequent calls to Write will
	// return false immediately.
	//
	// Segments must have channel keys in the set provided to DB.NewWriter. Segment data
	// must also be valid i.e. it must have non-zero length and be a multiple of the
	// channel's density. All segments must be provided in time-sorted order on a
	// per-channel basis.
	Write(segments []Segment) bool
	Commit()
	// Close closes the Writer and returns any error accumulated during execution. Close
	// will block until all segments have been persisted to the DB. It is not safe
	// to call Close concurrently with any other Writer methods.
	Close() error
}

type writer struct {
	requests  confluence.Inlet[WriteRequest]
	responses confluence.Outlet[WriteResponse]
	wg        signal.WaitGroup
	_error    error
}

func wrapStreamWriter(internal StreamWriter) *writer {
	sCtx, _ := signal.Background()
	req := confluence.NewStream[WriteRequest](1)
	res := confluence.NewStream[WriteResponse](1)
	internal.InFrom(req)
	internal.OutTo(res)
	internal.Flow(
		sCtx,
		confluence.CloseInletsOnExit(),
	)
	return &writer{requests: req, responses: res}
}

// Write implements the Writer interface.
func (w writer) Write(segments []Segment) bool {
	if w.error() != nil {
		return false
	}
	w.requests.Inlet() <- WriteRequest{Segments: segments}
	return true
}

// Close implements the Writer interface.
func (w writer) Close() (err error) {
	w.requests.Close()
	for res := range w.responses.Outlet() {
		err = res.Err
	}
	return err
}

func (w writer) Commit() {
	w.requests.Inlet() <- WriteRequest{Commit: true}
}

func (w writer) error() error {
	select {
	case res := <-w.responses.Outlet():
		w._error = res.Err
	default:
	}
	return w._error
}

type streamWriter struct {
	kv *kv.DB
	confluence.UnarySink[WriteRequest]
	confluence.AbstractUnarySource[WriteResponse]
	rateAligners    map[ChannelKey]*rateAligner
	indexAligners   map[ChannelKey]*indexAligner
	indexedAligners map[ChannelKey]*indexedAligner
	alloc           allocate.Allocator[ChannelKey, core.FileKey]
	storageWriter   storage.Writer
	kvWriter        core.MDBatch
}

func (w *streamWriter) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(w.Out)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-w.In.Outlet():
				if !ok {
					return w.close()
				}
				if req.Commit {
					if err := w.commit(); err != nil {
						w.Out.Inlet() <- WriteResponse{Err: err}
					}
				} else {
					if err := w.exec(req); err != nil {
						w.Out.Inlet() <- WriteResponse{Err: err}
					}
				}
			}
		}
	}, o.Signal...)
}

func (w *streamWriter) exec(req WriteRequest) error {
	for _, seg := range req.Segments {
		if err := w._exec(seg); err != nil {
			return err
		}
	}
	return nil
}

func (w *streamWriter) _exec(seg Segment) error {
	sug := core.SugaredSegment{}
	sug.Start = seg.Start
	sug.Data = seg.Data
	sug.ChannelKey = seg.ChannelKey
	sug.SegmentMD.Size = telem.Size(len(seg.Data))
	var err error
	if a, ok := w.rateAligners[seg.ChannelKey]; ok {
		if sug.Alignment, err = a.exec(sug); err != nil {
			return err
		}
	} else if a, ok := w.indexedAligners[seg.ChannelKey]; ok {
		if sug.Alignment, err = a.exec(sug); err != nil {
			return err
		}
	} else if a, ok := w.indexAligners[seg.ChannelKey]; ok {
		if sug.Alignment, err = a.exec(sug); err != nil {
			return err
		}
	} else {
		return errors.New("unexpected channel")
	}

	fileKeys, err := w.alloc.Allocate(allocate.Item[ChannelKey]{
		Key:  sug.ChannelKey,
		Size: sug.Size(),
	})
	if err != nil {
		return err
	}
	sug.FileKey = fileKeys[0]
	md, err := w.storageWriter.Write([]core.SugaredSegment{sug})
	if err != nil {
		return err
	}
	return w.kvWriter.Write(md)
}

func (w *streamWriter) commit() error {
	for _, idxW := range w.indexAligners {
		// get the hwm of every indexed writers whose channel.Index is equal to
		// idxW.Key()
		hws := []position.Position{position.Max}
		for _, idxW2 := range w.indexedAligners {
			if idxW2.channel.Index == idxW.channel.Index {
				hws = append(hws, idxW2.hwm)
			}
		}
		minHwm := lo.Min(hws)
		if err := idxW.buffer.Commit(minHwm); err != nil {
			return err
		}
	}
	if err := w.kvWriter.Commit(); err != nil {
		return err
	}
	if err := w.kvWriter.Close(); err != nil {
		return err
	}
	w.kvWriter = w.kv.NewWriter()
	return nil
}

func (w *streamWriter) close() error {
	if err := w.kvWriter.Close(); err != nil {
		return err
	}
	return nil
}

type indexedAligner struct {
	channel  Channel
	searcher index.Searcher
	iter     core.PositionIterator
	hwm      position.Position
}

func (s *indexedAligner) exec(seg core.SugaredSegment) (position.Position, error) {
	alignment, err := s.searcher.SearchP(seg.Start, position.Uncertain)
	if err != nil {
		return 0, err
	}
	if !alignment.Exact() {
		return 0, errors.New("contiguity error")
	}
	if alignment.Start.Before(s.hwm) {
		return 0, errors.New("overlapping segments")
	}
	seg.Alignment = alignment.Start
	s.iter.SetBounds(seg.Range(s.channel.Density))
	if !s.iter.SeekFirst() {
		panic("not first")
	}
	s.iter.Next(position.SpanMax)
	v := s.iter.Value()
	if len(v) > 1 {
		return 0, errors.New("segment too large")
	}
	if len(v) == 0 {
		return 0, errors.New("undefined")
	}
	if err != nil {
		return 0, err
	}
	s.hwm = seg.Range(s.channel.Density).End
	return seg.Alignment, nil
}

type rateAligner struct {
	channel  Channel
	searcher index.Searcher
	hwm      position.Position
}

func (s *rateAligner) exec(seg core.SugaredSegment) (position.Position, error) {
	alignment, _ := s.searcher.SearchP(seg.Start, position.Uncertain)
	if !alignment.Exact() {
		return 0, errors.New("contiguity error")
	}
	if alignment.Start.Before(s.hwm) {
		return 0, errors.New("overlapping segments")
	}
	seg.Alignment = alignment.Start
	s.hwm = seg.Range(s.channel.Density).End
	return alignment.Start, nil
}

type indexAligner struct {
	channel  Channel
	searcher index.Searcher
	buffer   *index.Buffered
	hwm      telem.TimeStamp
}

func (s *indexAligner) exec(seg core.SugaredSegment) (position.Position, error) {
	if seg.Start.Before(s.hwm) {
		return 0, errors.New("overlapping segments")
	}
	alignments, err := index.DecodeAlignments([]core.SugaredSegment{seg})
	if seg.Start != alignments[0].Stamp {
		return 0, errors.New("bad seg")
	}
	if err != nil {
		return 0, err
	}
	if err := s.buffer.Write(alignments); err != nil {
		return 0, err
	}
	s.hwm = alignments[len(alignments)-1].Stamp
	alignment, err := s.searcher.SearchP(seg.Start, position.Uncertain)
	if err != nil {
		return 0, err
	}
	return alignment.Start, nil
}
