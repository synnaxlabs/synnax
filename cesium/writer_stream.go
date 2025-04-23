// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
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
	// WriterSetAuthority represents a call to Writer.SetAuthority.
	WriterSetAuthority
)

var validateWriterCommand = validate.NewEnumBoundsChecker(WriterWrite, WriterSetAuthority)

// WriterRequest is a request containing a frame to write to the DB.
type WriterRequest struct {
	// Command is the command to execute on the Writer.
	Command WriterCommand
	// Frame is the arrow record to write to the DB.
	Frame Frame
	// Config is used for updating the parameters in WriterSetAuthority and WriterSetMode.
	Config WriterConfig
	// SeqNum is used to match the request with the response. The sequence number should
	// be incremented with each request.
	SeqNum int
}

// WriterResponse contains any errors that occurred during write execution.
type WriterResponse struct {
	// Command is the command that is being responded to.
	Command WriterCommand
	// SeqNum is the current sequence number of the command being executed. This value
	// will correspond to the WriterRequest.SeqNum that executed the command.
	SeqNum int
	// End is the end timestamp of the domain on commit. It is only valid during calls
	// to WriterCommit.
	End telem.TimeStamp
	// Authorized flags whether the write or commit operation was authorized. It is only
	// valid during calls to WriterWrite and WriterCommit.
	Authorized bool
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
	updateDBControl func(ctx context.Context, u ControlUpdate) error
}

// Flow implements the confluence.Flow interface.
func (w *streamWriter) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(w.Out)
	sCtx.Go(func(ctx context.Context) (err error) {
		defer func() {
			// Call close in a deferral to make sure writer resources get released even
			// if the context is canceled or the function panics. We need to pass in
			// a new context here because the original context may have been canceled.
			// Using context.TODO() is not ideal, but it is the best we can do here.
			err = errors.Combine(err, w.close(context.TODO()))
		}()
		for {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return
			case req, ok := <-w.In.Outlet():
				if !ok {
					return
				}
				if err = w.process(ctx, req); err != nil {
					return
				}
			}
		}
	}, o.Signal...)
}

func (w *streamWriter) process(ctx context.Context, req WriterRequest) error {
	if err := validateWriterCommand(req.Command); err != nil {
		return err
	}
	if req.Command == WriterSetAuthority {
		if err := w.setAuthority(ctx, req.Config); err != nil {
			return err
		}
		return w.sendRes(ctx, req, 0, true)
	}
	if req.Command == WriterCommit {
		ts, err := w.commit(ctx)
		isUnauthorized := errors.Is(err, control.Unauthorized)
		if err != nil && !isUnauthorized {
			return err
		}
		return w.sendRes(ctx, req, ts, !isUnauthorized)
	}
	err := w.write(ctx, req)
	isUnauthorized := errors.Is(err, control.Unauthorized)
	if err != nil && !isUnauthorized {
		return err
	}
	if *w.Sync {
		return w.sendRes(ctx, req, 0, !isUnauthorized)
	}
	return nil
}

func (w *streamWriter) setAuthority(ctx context.Context, cfg WriterConfig) error {
	if len(cfg.Authorities) == 0 {
		return nil
	}
	var (
		u       = ControlUpdate{Transfers: make([]controller.Transfer, 0, len(w.internal))}
		getAuth = func(ch ChannelKey) (control.Authority, bool) {
			return cfg.Authorities[0], true
		}
	)

	if len(cfg.Channels) > 0 {
		values := make(map[ChannelKey]control.Authority, len(cfg.Channels))
		for i, ch := range cfg.Channels {
			values[ch] = cfg.authority(i)
		}
		getAuth = func(ch ChannelKey) (control.Authority, bool) {
			v, ok := values[ch]
			return v, ok
		}

	}
	for _, chW := range w.virtual.internal {
		if auth, ok := getAuth(chW.Channel.Key); ok {
			if t := chW.SetAuthority(auth); t.Occurred() {
				u.Transfers = append(u.Transfers, t)
			}
		}
	}

	for _, idx := range w.internal {
		for _, chW := range idx.internal {
			if auth, ok := getAuth(chW.Channel.Key); ok {
				if t := chW.SetAuthority(auth); t.Occurred() {
					u.Transfers = append(u.Transfers, t)
				}
			}
		}
	}

	if len(u.Transfers) > 0 {
		return w.updateDBControl(ctx, u)
	}
	return nil
}

func (w *streamWriter) sendRes(
	ctx context.Context,
	req WriterRequest,
	end telem.TimeStamp,
	authorized bool,
) error {
	return signal.SendUnderContext(ctx, w.Out.Inlet(), WriterResponse{
		Command:    req.Command,
		SeqNum:     req.SeqNum,
		End:        end,
		Authorized: authorized,
	})
}

func (w *streamWriter) write(ctx context.Context, req WriterRequest) (err error) {
	for _, idx := range w.internal {
		req.Frame, err = idx.Write(req.Frame)
		if err != nil {
			return
		}

		if *w.EnableAutoCommit {
			if _, err = idx.Commit(ctx); err != nil {
				return
			}
		}
	}
	if w.virtual.internal != nil {
		if req.Frame, err = w.virtual.write(req.Frame); err != nil {
			return
		}
	}
	if w.Mode.Stream() {
		w.relay.Inlet() <- req.Frame
	}
	return
}

func (w *streamWriter) commit(ctx context.Context) (telem.TimeStamp, error) {
	maxTS := telem.TimeStampMin
	for _, idxW := range w.internal {
		ts, err := idxW.Commit(ctx)
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
	c := errors.NewCatcher(errors.WithAggregation())
	u := ControlUpdate{Transfers: make([]controller.Transfer, 0, len(w.internal))}
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
		_ = w.updateDBControl(ctx, u)
	}

	if digestWriter, ok := w.virtual.internal[w.virtual.digestKey]; ok {
		// When digest writer closes, we do not (and cannot) send an update.
		if _, err := digestWriter.Close(); err != nil {
			return err
		}
	}

	return c.Error()
}

type unaryWriterState struct {
	timesWritten int
	unary.Writer
}

// idxWriter is a writer to a set of channels that all share the same index.
type idxWriter struct {
	domainAlignment uint32
	start           telem.TimeStamp
	// internal contains writers for each channel
	internal map[ChannelKey]*unaryWriterState
	// writingToIdx is true when the Write is writing to the index
	// channel. This is typically true, which allows us to avoid
	// unnecessary lookups.
	writingToIdx bool
	// numWriteCalls tracks the number of write calls made to the idxWriter.
	numWriteCalls int
	idx           struct {
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
	// a single logical channel. i.e. N channels with M samples will result in a sample
	// count of M.
	sampleCount int64
}

func (w *idxWriter) Write(fr Frame) (Frame, error) {
	w.numWriteCalls++
	err := w.validateWrite(fr)
	if err != nil {
		return fr, err
	}

	var incrementedSampleCount bool

	for i, key := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(i) {
			continue
		}
		uWriter, ok := w.internal[key]
		series := fr.RawSeriesAt(i)
		if !ok || series.Len() == 0 {
			continue
		}

		if w.writingToIdx && w.idx.key == key {
			if err = w.updateHighWater(series); err != nil {
				return fr, err
			}
		}

		alignment, err := uWriter.Write(series)
		if err != nil {
			return fr, err
		}
		if !incrementedSampleCount {
			w.sampleCount = int64(alignment.SampleIndex()) + series.Len()
			incrementedSampleCount = true
		}
		series.Alignment = alignment
		fr.SetRawSeriesAt(i, series)
	}

	return fr, nil
}

func (w *idxWriter) Commit(ctx context.Context) (telem.TimeStamp, error) {
	if w.sampleCount == 0 {
		return w.start, nil
	}
	end, err := w.resolveCommitEnd(ctx)
	if err != nil {
		return 0, err
	}
	// because the range is exclusive, we need to add 1 nanosecond to the end
	end.Lower++
	c := errors.NewCatcher(errors.WithAggregation())
	for _, chW := range w.internal {
		c.Exec(func() error { return chW.CommitWithEnd(ctx, end.Lower) })
	}
	return end.Lower, c.Error()
}

func (w *idxWriter) Close() (ControlUpdate, error) {
	c := errors.NewCatcher(errors.WithAggregation())
	update := ControlUpdate{
		Transfers: make([]controller.Transfer, 0, len(w.internal)),
	}
	for _, unaryWriter := range w.internal {
		c.Exec(func() error {
			transfer, err := unaryWriter.Close()
			if err != nil || !transfer.Occurred() {
				return err
			}
			update.Transfers = append(update.Transfers, transfer)
			return nil
		})
	}
	return update, c.Error()
}

func (w *idxWriter) validateWrite(fr Frame) error {
	var (
		lengthOfFrame        int64 = -1
		numChannelsWrittenTo       = 0
	)
	for rawI, k := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		s := fr.RawSeriesAt(rawI)
		uWriter, ok := w.internal[k]
		if !ok {
			continue
		}

		if lengthOfFrame == -1 {
			// Data type of first series must be known since we use it to calculate the
			// length of series in the frame
			if s.DataType.Density() == telem.DensityUnknown {
				return errors.Wrapf(
					validate.Error,
					"invalid data type for channel %d, expected %s, got %s",
					k, uWriter.Channel.DataType, s.DataType)
			}
			lengthOfFrame = s.Len()
		}

		if uWriter.timesWritten == w.numWriteCalls {
			return errors.Wrapf(
				validate.Error,
				"frame must have exactly one series per channel, duplicate channel %d",
				k,
			)
		}

		uWriter.timesWritten++
		numChannelsWrittenTo++

		if s.Len() != lengthOfFrame {
			return errors.Wrapf(
				validate.Error,
				`frame must have the same length for all series, expected %d, got %d. \n
				See https://docs.synnaxlabs.com/reference/concepts/writes#rule-1
				`,
				lengthOfFrame,
				s.Len(),
			)
		}
	}

	if numChannelsWrittenTo == 0 {
		return nil
	}

	if numChannelsWrittenTo != len(w.internal) {
		return errors.Wrapf(
			validate.Error,
			`frame must have exactly one series for each data channel associated with an index. Expected
			%d series, got %d.
			See https://docs.synnaxlabs.com/reference/concepts/writes#the-rules-of-writes
			`,
			len(w.internal),
			numChannelsWrittenTo,
		)
	}

	return nil
}

func (w *idxWriter) updateHighWater(s telem.Series) error {
	if s.DataType != telem.TimeStampT && s.DataType != telem.Int64T {
		return errors.Wrapf(
			validate.Error,
			"invalid data type for channel %d, expected %s, got %s",
			w.idx.key,
			telem.TimeStampT,
			s.DataType,
		)
	}
	w.idx.highWaterMark = telem.ValueAt[telem.TimeStamp](s, -1)
	return nil
}

// resolveCommitEnd returns the end timestamp for a commit.
// For an index channel, this returns the high watermark.
// For a non-index channel, this returns a stamp to the approximation of the end
func (w *idxWriter) resolveCommitEnd(ctx context.Context) (index.TimeStampApproximation, error) {
	if w.writingToIdx {
		return index.Exactly(w.idx.highWaterMark), nil
	}
	return w.idx.Stamp(ctx, w.start, w.sampleCount-1, true)
}

type virtualWriter struct {
	internal  map[ChannelKey]*virtual.Writer
	digestKey core.ChannelKey
}

func (w virtualWriter) write(fr Frame) (Frame, error) {
	for rawI, k := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		v, ok := w.internal[k]
		if !ok {
			continue
		}
		s := fr.RawSeriesAt(rawI)
		alignment, err := v.Write(s)
		if err != nil {
			return fr, err
		}
		s.Alignment = alignment
		fr.SetRawSeriesAt(rawI, s)
	}
	return fr, nil
}

func (w virtualWriter) Close() (ControlUpdate, error) {
	c := errors.NewCatcher(errors.WithAggregation())
	update := ControlUpdate{
		Transfers: make([]controller.Transfer, 0, len(w.internal)),
	}
	for _, chW := range w.internal {
		// We do not want to clean up digest channel since we want to use it to
		// send updates for closures.
		if chW.Channel.Key == w.digestKey {
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
