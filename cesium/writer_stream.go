package cesium

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/allocate"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/legindex"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/lock"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// WriterCommand is an enumeration of commands that can be sent to a writer.
type WriterCommand uint8

const (
	// WriterWrite represents a call to Writer.Write.
	WriterWrite WriterCommand = iota + 1
	// WriterCommit represents a call to Writer.Commit.
	WriterCommit
	// WriterError represents a call to Writer.Error.
	WriterError
)

// WriteRequest is a request containing a set of segments (segment) to write to the DB.
type WriteRequest struct {
	// Command is the command to execute on the writer.
	Command WriterCommand
	// Segments is a set of segments to write. Segments is ignored during calls
	// to WriterCommit and WriterError.
	Segments []Segment
}

// WriteResponse contains any errors that occurred during write execution.
type WriteResponse struct {
	// Command is the command that is being responded to.
	Command WriterCommand
	// Ack represents the return value of the command.
	Ack bool
	// SeqNum is the current sequence number of the command being executed. SeqNum is
	// incremented for WriterError and WriterCommit calls, but NOT WriterWrite calls.
	SeqNum int
	// Err is the return value of WriterError. Err is nil during calls to
	// WriterWrite and WriterCommit.
	Err error
}

// StreamWriter provides a streaming interface for writing segmented telemetry to a DB.
// StreamWriter provides the underlying functionality for Writer, and has almost exactly
// the same semantics. The streaming interface is exposed as a confluence segment that
// can accept one input stream and one output stream.
//
// To write segments, issue a WriteRequest to the StreamWriter's inlet. The StreamWriter
// will return any errors encountered during write execution through the outlet as a
// WriteResponse of variant WriteErrResponse. If the write was successful, the StreamWriter
// will not send any response.
//
// To commit the write, issue a WriteRequest with command WriterCommit to the StreamWriter's
// inlet. The StreamWriter will respond with a WriteResponse of variant WriteCommitResponse
// containing any errors encountered during commit execution. If the commit was successful,
// the StreamWriter will send a CommitResponse with a nil error.
//
// To close the StreamWriter, simply close the inlet. The StreamWriter will ensure that all
// in-progress requests have been served before closing the outlet. Closing the writer
// will NOT commit any pending writes. Once the StreamWriter has released all resources,
// the output stream will be closed and the StreamWriter will return any accumulated err
// through the signal context provided to Flow.
type StreamWriter = confluence.Segment[WriteRequest, WriteResponse]

type streamWriter struct {
	confluence.UnarySink[WriteRequest]
	confluence.AbstractUnarySource[WriteResponse]
	idxProcessors map[ChannelKey]indexProcessor
	batches       []*legindex.Batch
	idxReleasers  []legindex.Releaser
	lockReleaser  lock.Releaser
	alloc         allocate.Allocator[ChannelKey, core.FileKey]
	storageWriter storage.Writer
	mdBatch       core.MDBatch
	err           error
	seqNum        int
}

// Flow implements the confluence.Flow interface.
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
				if req.Command == WriterError {
					w.seqNum++
					w.Out.Inlet() <- WriteResponse{Command: WriterError, Err: w.err, SeqNum: w.seqNum}
					w.err = nil
				}
				if w.err != nil {
					w.Out.Inlet() <- WriteResponse{Command: req.Command, Ack: false, SeqNum: w.seqNum}
					continue
				}
				if req.Command == WriterCommit {
					w.seqNum++
					if err := w.commit(); err != nil {
						w.err = err
					}
					w.Out.Inlet() <- WriteResponse{Command: WriterCommit, Ack: w.err == nil, SeqNum: w.seqNum}
				} else {
					if err := w.write(req); err != nil {
						w.err = err
						w.Out.Inlet() <- WriteResponse{Command: WriterWrite, Ack: false, SeqNum: w.seqNum}
					}
				}
			}
		}
	}, o.Signal...)
}

func (w *streamWriter) write(req WriteRequest) error {
	sugared := w.sugar(req.Segments)
	// This guarantees that we write to indexes before we write to any channels
	// that are indexed by them.
	for _, b := range w.batches {
		indexSegments, ok := sugared[b.Key()]
		if ok {
			if err := w._write(b.Key(), indexSegments); err != nil {
				return err
			}
			delete(sugared, b.Key())
		}
	}
	return iter.MapForEachUntilError(w.sugar(req.Segments), w._write)
}

func (w *streamWriter) sugar(segments []Segment) map[ChannelKey][]core.SugaredSegment {
	sugared := make(map[ChannelKey][]core.SugaredSegment, len(segments))
	for _, seg := range segments {
		sugared[seg.ChannelKey] = append(
			sugared[seg.ChannelKey],
			core.SugaredSegment{
				Data: seg.Data,
				SegmentMD: core.SegmentMD{
					Start:      seg.Start,
					ChannelKey: seg.ChannelKey,
					Size:       telem.Size(len(seg.Data)),
				},
			})
	}
	return sugared
}

func (w *streamWriter) _write(key ChannelKey, segments []core.SugaredSegment) error {
	idxProcessor, ok := w.idxProcessors[key]
	if !ok {
		return errors.Wrap(
			validate.Error,
			"segment ch key not in set provided to NewBatch",
		)
	}
	segments, err := idxProcessor.process(segments)
	if err != nil {
		return err
	}
	fileKeys, err := w.alloc.Allocate(core.AllocationItems(segments)...)
	for i, fileKey := range fileKeys {
		segments[i].FileKey = fileKey
	}
	md, err := w.storageWriter.Write(segments)
	if err != nil {
		return err
	}
	return w.mdBatch.Write(md)
}

func (w *streamWriter) commit() error {
	for _, idxW := range w.batches {
		minHwm := position.Max
		for _, idxW2 := range w.idxProcessors {
			if idxW2.Key() == idxW.Key() && idxW2.highWaterMark().Before(minHwm) {
				minHwm = idxW2.highWaterMark()
			}
		}
		if err := idxW.CommitBelowThreshold(minHwm); err != nil {
			return err
		}
	}
	if err := w.mdBatch.Commit(); err != nil {
		return err
	}
	return nil
}

func (w *streamWriter) close() error {
	for _, idxW := range w.batches {
		if err := idxW.Commit(); err != nil {
			return err
		}
	}
	if err := w.mdBatch.Close(); err != nil {
		return err
	}
	for _, idx := range w.idxReleasers {
		if err := idx.Release(); err != nil {
			return err
		}
	}
	w.lockReleaser.Release()
	return w.err
}
