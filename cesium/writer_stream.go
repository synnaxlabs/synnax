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
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"math"
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
	// WriterSetAuthority represents a call to Writer.SetAuthority.
	WriterSetAuthority
	// WriterSetMode sets the operating WriterMode for the Writer. See the WriterMode
	// documentation for more.
	WriterSetMode
)

// WriterRequest is a request containing an arrow.Record to write to the DB.
type WriterRequest struct {
	// Command is the command to execute on the Writer.
	Command WriterCommand
	// Frame is the arrow record to write to the DB.
	Frame Frame
	// Config is used for updating the parameters in WriterSetAuthority and WriterSetMode.
	Config WriterConfig
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
	// End is the end timestamp of the domain on commit. It is only valid during calls
	// to WriterCommit.
	End telem.TimeStamp
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
	relay           confluence.Inlet[Frame]
	internal        []*idxWriter
	virtual         *virtualWriter
	seqNum          int
	err             error
	updateDBControl func(ctx context.Context, u ControlUpdate)
}

// Flow implements the confluence.Flow interface.
func (w *streamWriter) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(w.Out)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return errors.CombineErrors(w.close(ctx), ctx.Err())
			case req, ok := <-w.In.Outlet():
				if !ok {
					return w.close(ctx)
				}
				w.process(ctx, req)
			}
		}
	}, o.Signal...)
}

func (w *streamWriter) process(ctx context.Context, req WriterRequest) {
	if req.Command < WriterWrite || req.Command > WriterSetAuthority {
		panic("[cesium.streamWriter] - invalid command")
	}
	if req.Command == WriterError {
		w.seqNum++
		w.sendRes(req, false, w.err, 0)
		w.err = nil
		return
	}
	if req.Command == WriterSetAuthority {
		w.seqNum++
		w.setAuthority(ctx, req.Config)
		w.sendRes(req, true, nil, 0)
		return
	}
	if req.Command == WriterSetMode {
		w.seqNum++
		w.setMode(req.Config)
		w.sendRes(req, true, nil, 0)
		return
	}
	if w.err != nil {
		w.seqNum++
		w.sendRes(req, false, nil, 0)
		return
	}
	if req.Command == WriterCommit {
		w.seqNum++
		var end telem.TimeStamp
		end, w.err = w.commit(ctx)
		w.sendRes(req, w.err == nil, nil, end)
	} else {
		if w.err = w.write(req); w.err != nil {
			w.seqNum++
			w.sendRes(req, false, nil, 0)
		}
	}
}

func (w *streamWriter) setAuthority(ctx context.Context, cfg WriterConfig) {
	if len(cfg.Authorities) == 0 {
		return
	}
	u := ControlUpdate{Transfers: make([]controller.Transfer, 0, len(w.internal))}
	if len(cfg.Channels) == 0 {
		for _, chW := range w.virtual.internal {
			transfer := chW.SetAuthority(cfg.Authorities[0])
			if transfer.Occurred() {
				u.Transfers = append(u.Transfers, transfer)
			}
		}

		for _, idx := range w.internal {
			for _, chW := range idx.internal {
				transfer := chW.SetAuthority(cfg.Authorities[0])
				if transfer.Occurred() {
					u.Transfers = append(u.Transfers, transfer)
				}
			}
		}
	} else {
		for i, ch := range cfg.Channels {
			for _, idx := range w.internal {
				for _, chW := range idx.internal {
					if chW.Channel.Key == ch {
						transfer := chW.SetAuthority(cfg.authority(i))
						if transfer.Occurred() {
							u.Transfers = append(u.Transfers, transfer)
						}
					}
				}
			}
			for _, chW := range w.virtual.internal {
				if chW.Channel.Key == ch {
					transfer := chW.SetAuthority(cfg.authority(i))
					if transfer.Occurred() {
						u.Transfers = append(u.Transfers, transfer)
					}
				}
			}
		}
	}
	if len(u.Transfers) > 0 {
		w.updateDBControl(ctx, u)
	}
}

func (w *streamWriter) setMode(cfg WriterConfig) {
	persist := cfg.Mode < WriterStreamOnly
	for _, idx := range w.internal {
		for _, chW := range idx.internal {
			chW.SetPersist(persist)
		}
	}
	w.Mode = cfg.Mode
}

func (w *streamWriter) sendRes(req WriterRequest, ack bool, err error, end telem.TimeStamp) {
	w.Out.Inlet() <- WriterResponse{
		Command: req.Command,
		Ack:     ack,
		SeqNum:  w.seqNum,
		Err:     err,
		End:     end,
	}
}

func (w *streamWriter) write(req WriterRequest) (err error) {
	for _, idx := range w.internal {
		req.Frame, err = idx.Write(req.Frame)
		if err != nil {
			if errors.Is(err, control.Unauthorized) && !*w.ErrOnUnauthorized {
				return nil
			}
			return err
		}
	}
	if w.virtual.internal != nil {
		req.Frame, err = w.virtual.write(req.Frame)
		if err != nil {
			if errors.Is(err, control.Unauthorized) && !*w.ErrOnUnauthorized {
				return nil
			}
			return err
		}
	}
	if w.Mode != WriterPersistOnly {
		w.relay.Inlet() <- req.Frame
	}
	return nil
}

func (w *streamWriter) commit(ctx context.Context) (telem.TimeStamp, error) {
	maxTS := telem.TimeStampMin
	for _, idx := range w.internal {
		ts, err := idx.Commit(ctx)
		if err != nil {
			return maxTS, err
		}
		if ts > maxTS {
			maxTS = ts
		}
	}
	return maxTS, nil
}

func (w *streamWriter) close(ctx context.Context) error {
	c := errutil.NewCatch(errutil.WithAggregation())
	u := ControlUpdate{Transfers: make([]controller.Transfer, 0, len(w.internal)+1)}
	for _, idx := range w.internal {
		c.Exec(func() error {
			u_, err := idx.Close()
			if err != nil {
				return err
			}
			u.Transfers = append(u.Transfers, u_.Transfers...)
			return nil
		})
	}
	if w.virtual.internal != nil {
		c.Exec(func() error {
			u_, err := w.virtual.Close()
			if err != nil {
				return err
			}
			u.Transfers = append(u.Transfers, u_.Transfers...)
			return nil
		})
	}

	if len(u.Transfers) > 0 {
		w.updateDBControl(ctx, u)
	}
	return errors.CombineErrors(w.err, c.Error())
}

type unaryWriterState struct {
	unary.Writer
	count int64
}

// idxWriter is a writer to a set of channels that all share the same index.
type idxWriter struct {
	start telem.TimeStamp
	// internal contains writers for each channel
	internal map[ChannelKey]*unaryWriterState
	// writingToIdx is true when the Write is writing to the index
	// channel. This is typically true, which allows us to avoid
	// unnecessary lookups.
	writingToIdx bool
	// writeNum tracks the number of calls to Write that have been made.
	writeNum int64
	idx      struct {
		// Index is the index used to resolve timestamps for domains in the DB.
		index.Index
		// Key is the channel key of the index. This field is not applicable when
		// the index is rate based.
		key core.ChannelKey
		// highWaterMark is the highest timestamp written to the index. This watermark
		// is only relevant when writingToIdx is true.
		highWaterMark telem.TimeStamp
	}
	// sampleCount is the total number of samples written to the index as if it were
	// a single logical channel. I.E. N channels with M samples will result in a sample
	// count of M.
	sampleCount int64
}

func (w *idxWriter) Write(fr Frame) (Frame, error) {
	var (
		l int64 = -1
		c       = 0
	)
	w.writeNum++
	for i, k := range fr.Keys {
		s, ok := w.internal[k]
		if !ok {
			continue
		}

		if l == -1 {
			l = fr.Series[i].Len()
		}

		if s.count == w.writeNum {
			return fr, errors.Wrapf(
				validate.Error,
				"frame must have one and only one series per channel, duplicate channel %s",
				k,
			)
		}

		s.count++
		c++

		if fr.Series[i].Len() != l {
			return fr, errors.Wrapf(
				validate.Error,
				"frame must have the same length for all series, expected %d, got %d",
				l,
				fr.Series[i].Len(),
			)
		}
	}

	if c == 0 {
		return fr, nil
	}

	if c != len(w.internal) {
		return fr, errors.Wrapf(
			validate.Error,
			"frame must have one and only one series per channel, expected %d, got %d",
			len(w.internal),
			c,
		)

	}

	for i, series := range fr.Series {
		key := fr.Keys[i]
		chW, ok := w.internal[key]

		if !ok {
			continue
		}

		if w.writingToIdx && w.idx.key == key {
			if err := w.updateHighWater(series); err != nil {
				return fr, err
			}
		}

		alignment, err := chW.Write(series)
		if err != nil {
			return fr, err
		}
		if i == 0 {
			w.sampleCount = int64(alignment) + series.Len()
		}
		series.Alignment = alignment
		fr.Series[i] = series
	}

	return fr, nil
}

func (w *idxWriter) Commit(ctx context.Context) (telem.TimeStamp, error) {
	end, err := w.resolveCommitEnd(ctx)
	if err != nil {
		return end.Lower, err
	}
	// because the range is exclusive, we need to add 1 nanosecond to the end
	end.Lower++
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, chW := range w.internal {
		c.Exec(func() error { return chW.CommitWithEnd(ctx, end.Lower) })
	}
	return end.Lower, c.Error()
}

func (w *idxWriter) Close() (ControlUpdate, error) {
	c := errutil.NewCatch(errutil.WithAggregation())
	update := ControlUpdate{
		Transfers: make([]controller.Transfer, 0, len(w.internal)),
	}
	for _, chW := range w.internal {
		c.Exec(func() error {
			transfer, err := chW.Close()
			if err != nil || !transfer.Occurred() {
				return err
			}
			update.Transfers = append(update.Transfers, transfer)
			return nil
		})
	}
	return update, c.Error()
}

func (w *idxWriter) updateHighWater(col telem.Series) error {
	if col.DataType != telem.TimeStampT && col.DataType != telem.Int64T {
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

func (w *idxWriter) resolveCommitEnd(ctx context.Context) (index.TimeStampApproximation, error) {
	if w.writingToIdx {
		return index.Exactly(w.idx.highWaterMark), nil
	}
	return w.idx.Stamp(ctx, w.start, w.sampleCount-1, true)
}

type virtualWriter struct {
	internal map[ChannelKey]*virtual.Writer
}

func (w virtualWriter) write(fr Frame) (Frame, error) {
	for i, k := range fr.Keys {
		v, ok := w.internal[k]
		if !ok {
			continue
		}
		series := fr.Series[i]
		alignment, err := v.Write(series)
		if err != nil {
			return fr, err
		}
		series.Alignment = alignment
		fr.Series[i] = series
	}
	return fr, nil
}

func (w virtualWriter) Close() (ControlUpdate, error) {
	c := errutil.NewCatch(errutil.WithAggregation())
	update := ControlUpdate{
		Transfers: make([]controller.Transfer, 0, len(w.internal)),
	}
	for _, chW := range w.internal {
		if chW.Channel.Key == math.MaxUint32 {
			continue
		}
		c.Exec(func() error {
			transfer, err := chW.Close()
			if err != nil || !transfer.Occurred() {
				return err
			}
			update.Transfers = append(update.Transfers, transfer)
			return nil
		})
	}
	return update, c.Error()
}
