package cesium

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

// Writer writes segmented telemetry to the DB. Writer must be closed after use. Writer
// acquires an exclusive write lock on the channels it writes to.
//
// Writer is not goroutine-safe, but it is safe to use multiple writers for different
// channels concurrently.
//
// Writer is asynchronous, meaning that calls to Write will return before segments are
// persisted to disk. The caller can guarantee that all segments have been persisted
// by calling Commit and waiting for it to return true.
type Writer interface {
	// Write writes the provided segments to the DB. If the Writer has encountered an
	// error, Write will return false. After Write returns false, subsequent calls to
	// Write will immediately return false until the user acknowledges the error by
	// calling Error, or by closing the Writer.
	//
	// Segments must have channel keys in the set provided to DB.NewWriter. Segment's must
	// meet the following requirements:
	//
	//		1. Index Channels (Channel.IsIndex == true):
	//			- Must contain ordered int64 values.
	//			- The first timestamp must equal the `Start` field of the Segment.
	//			- Must not overlap with any other segment in the ch.
	//
	//		2. Indexed Channels (Channel.Index != 0):
	//			- Must have the same starting timestamp and size as a segment written
	//			  to the index ch.
	//         	- Must not overlap with any other segment in the ch.
	//
	//		3. Rate Based Channels (Channel.Index == 0 && Channel.Rate != 0):
	//			- Must not overlap with any other segment in the ch.
	//
	// When writing to channels that are indexed, the caller should write to the index
	// ch first, and then write to any channels that are indexed by it.
	Write(segments []Segment) bool
	// Commit commits all uncommitted segments to the DB. This method will block until all
	// segments have been committed to disk. If the Writer has accumulated an error.
	// this method will return false, and no segments will be committed. After Commit
	// returns false, subsequent calls to Commit will return false immediately.
	// Commit can be called multiple times, and should be called periodically during
	// long-running writes to reduce memory usage.
	Commit() bool
	// Error returns the error that the Writer has accumulated. If the Writer has not
	// accumulated an error, this method will return nil. After Error is called, the
	// Writer will be reset, and subsequent calls to Write will return true.
	Error() error
	// Close closes the Writer and returns any error accumulated during execution. Close
	// will block until all segments have been persisted to the DB. It is not safe
	// to call Close concurrently with any other Writer methods.
	Close() error
}

type writer struct {
	requests          confluence.Inlet[WriteRequest]
	responses         confluence.Outlet[WriteResponse]
	wg                signal.WaitGroup
	logger            *zap.Logger
	hasAccumulatedErr bool
}

const writerUnexpectedEarlyClosure = "[cesium.writer] - unexpected early closure of writer response stream"

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
	return &writer{requests: req, responses: res, wg: sCtx}
}

// Write implements the Writer interface.
func (w *writer) Write(segments []Segment) bool {
	if w.hasAccumulatedErr {
		return false
	}
	select {
	case <-w.responses.Outlet():
		w.hasAccumulatedErr = true
		return false
	case w.requests.Inlet() <- WriteRequest{Segments: segments, Command: WriterWrite}:
		return true
	}
}

// Commit implements the Writer interface.
func (w *writer) Commit() bool {
	if w.hasAccumulatedErr {
		return false
	}
	select {
	case <-w.responses.Outlet():
		w.hasAccumulatedErr = true
		return false
	case w.requests.Inlet() <- WriteRequest{Command: WriterCommit}:
	}
	for res := range w.responses.Outlet() {
		if res.Command == WriterCommit {
			return res.Ack
		}
	}
	w.logger.DPanic(writerUnexpectedEarlyClosure)
	return false
}

func (w *writer) Error() error {
	w.requests.Inlet() <- WriteRequest{Command: WriterError}
	for res := range w.responses.Outlet() {
		if res.Command == WriterError {
			w.hasAccumulatedErr = false
			return res.Err
		}
	}
	w.logger.DPanic(writerUnexpectedEarlyClosure)
	return errors.New(writerUnexpectedEarlyClosure)
}

// Close implements the Writer interface.
func (w *writer) Close() (err error) {
	w.requests.Close()
	for range w.responses.Outlet() {
	}
	return w.wg.Wait()
}
