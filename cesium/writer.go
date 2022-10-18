package cesium

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/lock"
	"github.com/synnaxlabs/x/signal"
)

// WriteRequest is a request containing a set of segments (segment) to write to the DB.
type WriteRequest struct {
	// Segments is a set of segments to write to the DB.
	Segments []Segment
}

// WriteResponse contains any errors that occurred during the execution of the Create Query.
type WriteResponse struct {
	// Err is any err that occurred during internal execution.
	Err error
}

type StreamWriter = confluence.Segment[WriteRequest, WriteResponse]

// Writer writes segmented telemetry to the DB. FileKey internal must be closed after use. Writer
// is not goroutine-safe, but it is safe to use multiple writers for different Channels
// concurrently.
//
// Writer is asynchronous, meaning that calls to Write will return before segments are
// persisted to disk. The caller can guarantee that all segments have been persisted
// by calling Close.
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
	// Close closes the Writer and returns any err accumulated during execution. Close
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
		confluence.CancelOnExitErr(),
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

func (w writer) error() error {
	select {
	case res := <-w.responses.Outlet():
		w._error = res.Err
	default:
	}
	return w._error
}

// storageWriter writes segment data to the DB.
type storageWriter struct {
	internal storage.Writer
	confluence.LinearTransform[[]core.SugaredSegment, []core.SugaredSegment]
}

func newStorageWriter(internal storage.Writer) *storageWriter {
	s := &storageWriter{internal: internal}
	s.Transform = s.transform
	return s
}

func (s *storageWriter) transform(
	ctx context.Context,
	segments []core.SugaredSegment,
) ([]core.SugaredSegment, bool, error) {
	mds, err := s.internal.Write(segments)
	if err != nil {
		return segments, false, err
	}
	for i, seg := range segments {
		seg.SegmentMD = mds[i]
		segments[i] = seg
	}
	return segments, true, nil
}

// mdWriter is a writer that writes metadata to the DB.
type mdWriter struct {
	internal core.MDWriter
	keys     []ChannelKey
	lock     lock.Keys[ChannelKey]
	confluence.LinearTransform[[]core.SugaredSegment, WriteResponse]
}

func newMDWriter(writer core.MDWriter, keys []ChannelKey, lock lock.Keys[ChannelKey]) *mdWriter {
	md := &mdWriter{internal: writer, keys: keys, lock: lock}
	md.Transform = md.transform
	return md
}

func (m *mdWriter) Flow(ctx signal.Context, opts ...confluence.Option) {
	m.LinearTransform.Flow(
		ctx,
		append(opts, confluence.Defer(func() {
			m.internal.Commit()
			m.lock.Unlock(m.keys...)
		}))...,
	)
}

func (m *mdWriter) transform(
	ctx context.Context,
	segments []core.SugaredSegment,
) (WriteResponse, bool, error) {
	mds := make([]core.SegmentMD, len(segments))
	for i, seg := range segments {
		mds[i] = seg.SegmentMD
	}
	err := m.internal.Write(mds)
	return WriteResponse{Err: err}, err != nil, err
}
