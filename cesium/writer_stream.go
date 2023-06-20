// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// WriterCommand is an enumeration of commands that can be sent to a Writer.
type WriterCommand uint8

const (
	// WriterWrite represents a call to Writer.Write.
	WriterWrite WriterCommand = iota + 1
	// WriterCommit represents a call to Writer.Commit.
	WriterCommit
	// WriterError represents a call to Writer.Error.
	WriterError
)

// WriterRequest is a request containing an arrow.Record to write to the DB.
type WriterRequest struct {
	// Command is the command to execute on the Writer.
	Command WriterCommand
	// Frame is the arrow record to write to the DB.
	Frame Frame
}

// WriterResponse contains any errors that occurred during write execution.
type WriterResponse struct {
	// Command is the command that is being responded to.
	Command WriterCommand
	// Ack represents the return frame of the command.
	Ack bool
	// SeqNum is the current sequence number of the command being executed. SeqNum is
	// incremented for WriterError and WriterCommit calls, but NOT WriterWrite calls.
	SeqNum int
	// Err is the return frame of WriterError. Err is nil during calls to
	// WriterWrite and WriterCommit.
	Err error
}

// StreamWriter provides a streaming interface for writing telemetry to the DB.
// StreamWriter provides the underlying functionality for Writer, and has almost exactly
// the same semantics. The streaming interface is exposed as a confluence segment that
// can accept one input stream and one output stream.
//
// To write a record, issue a WriterRequest to the StreamWriter's inlet. If the write
// fails, the StreamWriter will send a WriterResponse with a negative WriterResponse.Ack
// frame. All future writes will fail until the error is resolved. To resolve the error,
// issue a WriterRequest with a WriterError command to the StreamWriter's inlet. The
// StreamWriter will increment WriterResponse.SeqNum and send a WriterResponse with the
// error. The error will be considered resolved, and the StreamWriter will resume normal
// operation.
//
// StreamWriter is atomic, meaning the caller must issue a set with a WriterCommit
// command to commit the write. If the commit fails, the StreamWriter will send a
// WriterResponse with a negative WriterResponse.Ack frame. All future writes will fail
// until the error is resolved. To resolve the error, see the above paragraph.
//
// To close the StreamWriter, simply close the inlet. The StreamWriter will ensure that
// all in-progress requests have been served before closing the outlet. Closing the Writer
// will NOT commit any pending writes. Once the StreamWriter has released all resources,
// the output stream will be closed and the StreamWriter will return any accumulated error
// through the signal context provided to Flow.
type StreamWriter = confluence.Segment[WriterRequest, WriterResponse]

type streamWriter struct {
	WriterConfig
	confluence.UnarySink[WriterRequest]
	confluence.AbstractUnarySource[WriterResponse]
	// internal contains writers for each channel
	internal map[core.ChannelKey]unary.Writer
	// writingToIdx is true when the write is writing to the index
	// channel. This is typically true, which allows us to avoid
	// unnecessary lookups.
	writingToIdx bool
	idx          struct {
		index.Index
		key           core.ChannelKey
		highWaterMark telem.TimeStamp
	}
	sampleCount int64
	seqNum      int
	err         error
	relay       confluence.Inlet[Frame]
}

// Flow implements the confluence.Flow interface.
func (w *streamWriter) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(w.Out)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return errors.CombineErrors(w.close(), ctx.Err())
			case req, ok := <-w.In.Outlet():
				if !ok {
					return w.close()
				}
				if req.Command < WriterWrite || req.Command > WriterError {
					panic("[cesium.streamWriter] - invalid command")
				}
				if req.Command == WriterError {
					w.seqNum++
					w.sendRes(req, false, w.err)
					w.err = nil
					continue
				}
				if w.err != nil {
					w.seqNum++
					w.sendRes(req, false, nil)
					continue
				}
				if req.Command == WriterCommit {
					w.seqNum++
					w.err = w.commit(ctx)
					w.sendRes(req, w.err == nil, nil)
				} else {
					if w.err = w.write(req); w.err != nil {
						w.seqNum++
						w.sendRes(req, false, nil)
					}
				}
			}
		}
	}, o.Signal...)
}

func (w *streamWriter) sendRes(req WriterRequest, ack bool, err error) {
	w.Out.Inlet() <- WriterResponse{Command: req.Command, Ack: ack, SeqNum: w.seqNum, Err: err}
}

func (w *streamWriter) write(req WriterRequest) error {
	if !req.Frame.Even() {
		return errors.Wrapf(validate.Error, "cannot write uneven frame")
	}

	if !req.Frame.Unary() {
		return errors.Wrapf(validate.Error, "cannot write frame with duplicate channels")
	}

	if len(req.Frame.Keys) != len(w.internal) {
		return errors.Wrapf(validate.Error, "cannot write frame without data for all channels")
	}

	w.sampleCount += req.Frame.Len()

	w.relay.Inlet() <- req.Frame

	for i, series := range req.Frame.Series {
		key := req.Frame.Key(i)
		_chW, ok := w.internal[req.Frame.Keys[i]]
		if !ok {
			return errors.Wrapf(
				validate.Error,
				"cannot write array for channel %s that was not specified when opening the Writer",
				key,
			)
		}

		chW := &_chW

		if w.writingToIdx && w.idx.key == key {
			if err := w.updateHighWater(series); err != nil {
				return err
			}
		}

		if err := chW.Write(series); err != nil {
			return err
		}
	}

	return nil
}

func (w *streamWriter) commit(ctx context.Context) (err error) {
	end, err := w.resolveCommitEnd(ctx)
	if err != nil {
		return err
	}
	// because the range is exclusive, we need to add 1 nanosecond to the end
	end.Lower++
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, chW := range w.internal {
		c.Exec(func() error { return chW.CommitWithEnd(ctx, end.Lower) })
	}
	w.err = c.Error()
	return
}

func (w *streamWriter) updateHighWater(col telem.Series) error {
	if col.DataType != telem.TimeStampT {
		return errors.Wrapf(
			validate.Error,
			"invalid data type for channel %s, expected %s, got %s",
			w.idx.key, telem.TimeStampT,
			col.DataType,
		)
	}
	w.idx.highWaterMark = telem.ValueAt[telem.TimeStamp](col, col.Len()-1)
	return nil
}

func (w *streamWriter) resolveCommitEnd(ctx context.Context) (index.TimeStampApproximation, error) {
	if w.writingToIdx {
		return index.Exactly(w.idx.highWaterMark), nil
	}
	return w.idx.Stamp(ctx, w.Start, w.sampleCount-1, true)
}

func (w *streamWriter) close() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, chW := range w.internal {
		c.Exec(chW.Close)
	}
	return errors.CombineErrors(w.err, c.Error())
}
