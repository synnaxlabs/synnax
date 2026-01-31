// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/confluence"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/stringer"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// WriterCommand is an enumeration of commands that can be sent to a Writer.
type WriterCommand uint8

const (
	// WriterCommandWrite represents a call to Writer.Write.
	WriterCommandWrite WriterCommand = iota + 1
	// WriterCommandCommit represents a call to Writer.Commit.
	WriterCommandCommit
	// WriterCommandSetAuthority represents a call to Writer.SetAuthority.
	WriterCommandSetAuthority
)

var validateWriterCommand = validate.NewInclusiveBoundsChecker(WriterCommandWrite, WriterCommandSetAuthority)

// WriterRequest is a request containing a frame to write to the DB.
type WriterRequest struct {
	// Config is used for updating the parameters in WriterCommandSetAuthority.
	Config WriterConfig
	// Frame is the arrow record to write to the DB.
	Frame Frame
	// SeqNum is used to match the request with the response. The sequence number should
	// be incremented with each request.
	SeqNum int
	// Command is the command to execute on the Writer.
	Command WriterCommand
}

// WriterResponse contains any errors that occurred during write execution.
type WriterResponse struct {
	// Err contains an error that occurred when attempting to execute a request on the
	// writer.
	Err error
	// SeqNum is the current sequence number of the command being executed. This value
	// will correspond to the WriterRequest.SeqNum that executed the command.
	SeqNum int
	// End is the end timestamp of the domain on commit. It is only valid during calls
	// to WriterCommit.
	End telem.TimeStamp
	// Command is the command that is being responded to.
	Command WriterCommand
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
// all in-progress requests have been served before closing the outlet. Closing the
// Writer will NOT commit any pending writes. Once the StreamWriter has released all
// resources, the output stream will be closed and the StreamWriter will return any
// accumulated error through the signal context provided to Flow.
type StreamWriter = confluence.Segment[WriterRequest, WriterResponse]

type streamWriter struct {
	confluence.UnarySink[WriterRequest]
	confluence.AbstractUnarySource[WriterResponse]
	relay           confluence.Inlet[Frame]
	accumulatedErr  error
	virtual         *virtualWriter
	updateDBControl func(ctx context.Context, u ControlUpdate) error
	internal        []*idxWriter
	WriterConfig
}

// Flow implements the confluence.Flow interface.
func (w *streamWriter) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(w.Out)
	sCtx.Go(func(ctx context.Context) (err error) {
		defer func() {
			// Call close in a deferral to make sure writer resources get released even
			// if the context is canceled or the function panics. We need to pass in a
			// new context here because the original context may have been canceled.
			// Using context.TODO() is not ideal, but it is the best we can do here.
			err = errors.Combine(err, w.close(context.TODO()))
		}()
		for {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				if w.accumulatedErr != nil {
					err = w.accumulatedErr
				}
				return
			case req, ok := <-w.In.Outlet():
				if !ok {
					err = w.accumulatedErr
					return
				}
				var commitEnd telem.TimeStamp
				if w.accumulatedErr == nil {
					commitEnd, w.accumulatedErr = w.process(ctx, req)
				}
				if err := w.maybeSendRes(ctx, req, commitEnd); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}

func (w *streamWriter) process(ctx context.Context, req WriterRequest) (commitEnd telem.TimeStamp, err error) {
	if err = validateWriterCommand(req.Command); err != nil {
		return 0, err
	}
	if req.Command == WriterCommandSetAuthority {
		err = w.setAuthority(ctx, req.Config)
		return
	}
	if req.Command == WriterCommandCommit {
		commitEnd, err = w.commit(ctx)
		return
	}
	err = w.write(ctx, req)
	return
}

func (w *streamWriter) setAuthority(ctx context.Context, cfg WriterConfig) error {
	if len(cfg.Authorities) == 0 {
		return nil
	}
	var (
		u       = ControlUpdate{Transfers: make([]control.Transfer, 0, len(w.internal))}
		getAuth = func(ch ChannelKey) (xcontrol.Authority, bool) {
			return cfg.Authorities[0], true
		}
	)

	if len(cfg.Channels) > 0 {
		values := make(map[ChannelKey]xcontrol.Authority, len(cfg.Channels))
		for i, ch := range cfg.Channels {
			values[ch] = cfg.authority(i)
		}
		getAuth = func(ch ChannelKey) (xcontrol.Authority, bool) {
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

func (w *streamWriter) maybeSendRes(
	ctx context.Context,
	req WriterRequest,
	end telem.TimeStamp,
) error {
	res := WriterResponse{Command: req.Command, SeqNum: req.SeqNum, End: end, Authorized: true}
	if w.accumulatedErr != nil && errors.Is(w.accumulatedErr, xcontrol.ErrUnauthorized) {
		w.accumulatedErr = nil
		res.Authorized = false
	}
	res.Err = w.accumulatedErr
	if res.Err == nil && req.Command == WriterCommandWrite && !*w.Sync {
		return nil
	}
	return signal.SendUnderContext(ctx, w.Out.Inlet(), res)
}

func (w *streamWriter) write(ctx context.Context, req WriterRequest) error {
	var (
		accumulatedErr      error
		err                 error
		excludeUnauthorized []ChannelKey
	)
	for _, idx := range w.internal {
		if req.Frame, err = idx.write(&excludeUnauthorized, req.Frame); err != nil {
			accumulatedErr = err
			if !errors.Is(accumulatedErr, xcontrol.ErrUnauthorized) {
				return accumulatedErr
			}
			continue
		}

		if *w.EnableAutoCommit {
			if _, err = idx.Commit(ctx); err != nil {
				accumulatedErr = err
				if !errors.Is(accumulatedErr, xcontrol.ErrUnauthorized) {
					return accumulatedErr
				}
			}
		}
	}
	if w.virtual.internal != nil {
		if req.Frame, err = w.virtual.write(&excludeUnauthorized, req.Frame); err != nil {
			accumulatedErr = err
			if !errors.Is(accumulatedErr, xcontrol.ErrUnauthorized) {
				return accumulatedErr
			}
		}
	}
	if w.Mode.Stream() {
		w.relay.Inlet() <- req.Frame.ExcludeKeys(excludeUnauthorized)
	}
	return accumulatedErr
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
	var c errors.Catcher
	parentUpdate := ControlUpdate{Transfers: make([]control.Transfer, 0, len(w.internal))}
	for _, idx := range w.internal {
		c.Exec(func() error {
			u, err := idx.Close()
			if err != nil {
				return err
			}
			parentUpdate.Transfers = append(parentUpdate.Transfers, u.Transfers...)
			return nil
		})
	}
	if w.virtual.internal != nil {
		c.Exec(func() error {
			u, err := w.virtual.Close()
			if err != nil {
				return err
			}
			parentUpdate.Transfers = append(parentUpdate.Transfers, u.Transfers...)
			return nil
		})
	}

	if len(parentUpdate.Transfers) > 0 {
		_ = w.updateDBControl(ctx, parentUpdate)
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
	unary.Writer
	timesWritten int
}

// idxWriter is a writer to a set of channels that all share the same index.
type idxWriter struct {
	// internal contains writers for each channel
	internal map[ChannelKey]*unaryWriterState
	idx      struct {
		// Index is the index used to resolve timestamps for domains in the DB.
		*index.Domain
		// Key is the channel key of the index.
		ch channel.Channel
		// highWaterMark is the highest timestamp written to the index. This watermark
		// is only relevant when writingToIdx is true.
		highWaterMark telem.TimeStamp
	}
	// numWriteCalls tracks the number of write calls made to the idxWriter.
	numWriteCalls int
	// sampleCount is the total number of samples written to the index as if it were a
	// single logical channel. i.e. N channels with M samples will result in a sample
	// count of M.
	sampleCount     int64
	start           telem.TimeStamp
	domainAlignment uint32
	// writingToIdx is true when the Write is writing to the index channel. This is
	// typically true, which allows us to avoid unnecessary lookups.
	writingToIdx bool
}

func (w *idxWriter) write(
	excludeUnauthorized *[]ChannelKey,
	fr Frame,
) (Frame, error) {
	w.numWriteCalls++
	err := w.validateWrite(fr)
	if err != nil {
		return fr, err
	}
	var (
		incrementedSampleCount bool
		accumulatedErr         error
	)
	for i, key := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(i) {
			continue
		}
		uWriter, ok := w.internal[key]
		series := fr.RawSeriesAt(i)
		if !ok || series.Len() == 0 {
			continue
		}

		if w.writingToIdx && w.idx.ch.Key == key {
			if err = w.updateHighWater(series); err != nil {
				return fr, err
			}
		}

		alignment, err := uWriter.Write(series)
		if err != nil {
			accumulatedErr = err
			if !errors.Is(accumulatedErr, xcontrol.ErrUnauthorized) {
				return fr, accumulatedErr
			}
			*excludeUnauthorized = append(*excludeUnauthorized, key)
			continue
		}
		if !incrementedSampleCount {
			w.sampleCount = int64(alignment.SampleIndex()) + series.Len()
			incrementedSampleCount = true
		}
		series.Alignment = alignment
		fr.SetRawSeriesAt(i, series)
	}
	return fr, accumulatedErr
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
	var c errors.Catcher
	for _, chW := range w.internal {
		c.Exec(func() error { return chW.CommitWithEnd(ctx, end.Lower) })
	}
	return end.Lower, c.Error()
}

func (w *idxWriter) Close() (ControlUpdate, error) {
	var c errors.Catcher
	update := ControlUpdate{
		Transfers: make([]control.Transfer, 0, len(w.internal)),
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

func invalidDataTypeError(expectedCh Channel, received telem.DataType) error {
	return errors.Wrapf(
		validate.ErrValidation,
		`invalid data type for channel %v, expected %s, got %s`,
		expectedCh,
		expectedCh.DataType,
		received,
	)
}

func oneSeriesPerChannelError(expected Channel) error {
	return errors.Wrapf(
		validate.ErrValidation,
		`frame must have exactly one series per channel, found more than one for channel %v`,
		expected,
	)
}

func sameLengthForAllSeriesError(
	expectedCh Channel,
	lengthOfFrame int64,
	series telem.Series,
) error {
	return errors.Wrapf(
		validate.ErrValidation,
		`
frame must have the same length for all series. Rest of the series in the frame have
length %d, while series for channel %v has length %d. See https://docs.synnaxlabs.com/reference/concepts/writes#rule-1
`,
		lengthOfFrame,
		expectedCh,
		series.Len(),
	)
}

func missingChannelError(
	index Channel,
	missing Channel,
	dataChannels []Channel,
) error {
	if index.Key == missing.Key {
		return errors.Wrapf(
			validate.ErrValidation,
			`received no data for index channel %v that must be provided when writing to related data channels %v`,
			missing,
			stringer.TruncateAndFormatSlice(dataChannels, 8),
		)
	}
	return errors.Wrapf(
		validate.ErrValidation,
		`frame must have exactly one series for each data channel associated with index %v, but is missing a series for channel %v`,
		index,
		missing,
	)
}

func incorrectNumberOfSeriesError(
	expected int,
	received int,

) error {
	return errors.Wrapf(
		validate.ErrValidation,
		`frame must have exactly one series for each data channel associated with an index. Expected
			%d series, got %d.
			See https://docs.synnaxlabs.com/reference/concepts/writes#the-rules-of-writes
			`,
		expected,
		received,
	)
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
			if s.DataType.Density() == telem.UnknownDensity {
				return invalidDataTypeError(uWriter.Channel, s.DataType)
			}
			lengthOfFrame = s.Len()
		}

		if uWriter.timesWritten == w.numWriteCalls {
			return oneSeriesPerChannelError(uWriter.Channel)
		}

		if s.Len() != lengthOfFrame {
			return sameLengthForAllSeriesError(uWriter.Channel, lengthOfFrame, s)
		}

		uWriter.timesWritten++
		numChannelsWrittenTo++
	}

	if numChannelsWrittenTo == 0 {
		return nil
	}

	if numChannelsWrittenTo != len(w.internal) {
		if numChannelsWrittenTo < len(w.internal) {
			keys := set.FromSlice(fr.KeysSlice())
			for k, db := range w.internal {
				if !keys.Contains(k) {
					dataChannels := make([]Channel, 0, len(keys))
					for _, db := range w.internal {
						if k != db.Channel.Key {
							dataChannels = append(dataChannels, db.Channel)
						}
					}
					return missingChannelError(w.idx.ch, db.Channel, dataChannels)
				}
			}
		}
		err := incorrectNumberOfSeriesError(len(w.internal), numChannelsWrittenTo)
		// This is an impossible condition
		zap.S().DPanic(err.Error())
		return err
	}

	return nil
}

func (w *idxWriter) updateHighWater(s telem.Series) error {
	if s.DataType != telem.TimeStampT && s.DataType != telem.Int64T {
		return invalidDataTypeError(w.idx.ch, s.DataType)
	}
	w.idx.highWaterMark = telem.ValueAt[telem.TimeStamp](s, -1)
	return nil
}

// resolveCommitEnd returns the end timestamp for a commit. For an index channel, this
// returns the high watermark. For a non-index channel, this returns a stamp to the
// approximation of the end
func (w *idxWriter) resolveCommitEnd(ctx context.Context) (index.TimeStampApproximation, error) {
	if w.writingToIdx {
		return index.Exactly(w.idx.highWaterMark), nil
	}
	return w.idx.Stamp(ctx, w.start, w.sampleCount-1, true)
}

type virtualWriter struct {
	internal  map[ChannelKey]*virtual.Writer
	digestKey channel.Key
}

func (w virtualWriter) write(filterUnauthorized *[]ChannelKey, fr Frame) (Frame, error) {
	var accumulatedErr error
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
			accumulatedErr = err
			if !errors.Is(err, xcontrol.ErrUnauthorized) {
				return fr, err
			}
			*filterUnauthorized = append(*filterUnauthorized, k)
			continue
		}
		s.Alignment = alignment
		fr.SetRawSeriesAt(rawI, s)
	}
	return fr, accumulatedErr
}

func (w virtualWriter) Close() (ControlUpdate, error) {
	var c errors.Catcher
	update := ControlUpdate{Transfers: make([]control.Transfer, 0, len(w.internal))}
	for _, chW := range w.internal {
		// We do not want to clean up the digest channel since we want to use it to send
		// updates for closures.
		if chW.Channel.Key != w.digestKey {
			c.Exec(func() error {
				transfer, err := chW.Close()
				if err != nil || !transfer.Occurred() {
					return err
				}
				update.Transfers = append(update.Transfers, transfer)
				return nil
			})
		}
	}
	return update, c.Error()
}
