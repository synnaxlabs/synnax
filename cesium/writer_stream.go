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

	"github.com/samber/lo"
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
	"github.com/synnaxlabs/x/unsafe"
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
	// Config is used for updating the parameters in WriterCommandSetAuthority. Once
	// the request has been sent, ownership of Config (including the underlying arrays
	// of Config.Channels and Config.Authorities) transfers to the streamWriter, which
	// may mutate them in place; callers must not retain or share those slices.
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
	relay           confluence.Inlet[relayResponse]
	accumulatedErr  error
	virtual         *virtualWriter
	updateDBControl func(ctx context.Context, u ControlUpdate) error
	// keyToIdx maps every channel key the writer is responsible for to its owning
	// idxWriter.
	keyToIdx map[ChannelKey]*idxWriter
	internal []*idxWriter
	WriterConfig
	errSent bool
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
	if *w.AutoIndex {
		cfg = w.propagateAuthority(cfg)
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
		w.errSent = false
		res.Authorized = false
	}
	res.Err = w.accumulatedErr
	if req.Command == WriterCommandWrite && !*w.Sync && (res.Err == nil || w.errSent) {
		return nil
	}
	if res.Err != nil {
		w.errSent = true
	}
	return signal.SendUnderContext(ctx, w.Out.Inlet(), res)
}

func (w *streamWriter) write(ctx context.Context, req WriterRequest) error {
	if err := w.validateSeries(req.Frame); err != nil {
		return err
	}
	if *w.AutoIndex {
		req.Frame = w.autoStamp(req.Frame)
	}
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
		w.relay.Inlet() <- relayResponse{
			frame: req.Frame.ExcludeKeys(excludeUnauthorized),
			group: w.ControlSubject.Group,
		}
	}
	return accumulatedErr
}

// validateSeries checks that every series in fr targets a channel the writer is
// responsible for and has a structurally valid data buffer for its declared data
// type (see telem.Series.Validate), rejecting the write before any data is applied.
func (w *streamWriter) validateSeries(fr Frame) error {
	for i, k := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(i) {
			continue
		}
		if !w.owns(k) {
			return errors.Wrapf(
				validate.ErrValidation,
				"channel %d is not part of the writer's channel set",
				k,
			)
		}
		s := fr.RawSeriesAt(i)
		if err := s.Validate(); err != nil {
			return errors.Wrapf(err, "channel %d", k)
		}
	}
	return nil
}

// owns reports whether the writer is responsible for writes to the given channel.
func (w *streamWriter) owns(k ChannelKey) bool {
	if _, ok := w.virtual.internal[k]; ok {
		return true
	}
	for _, idx := range w.internal {
		if _, ok := idx.internal[k]; ok {
			return true
		}
	}
	return false
}

// autoStamp injects a TimeStamp series for each idxWriter whose index channel is
// referenced by the writer but absent from fr. It runs in O(KeysInFrame + IndexGroups):
// a single pass over fr's keys uses keyToIdx to resolve each key's idxWriter group in
// O(1) and records whether that group's index is present and the length of any data
// channel for the group. A subsequent pass over idxWriters counts how many stamps are
// needed, grows fr by exactly that amount, and performs the appends.
func (w *streamWriter) autoStamp(fr Frame) Frame {
	for _, idx := range w.internal {
		idx.resetAutoStampScan()
	}
	for i, k := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(i) {
			continue
		}
		idx, ok := w.keyToIdx[k]
		if !ok {
			continue
		}
		if k == idx.idx.ch.Key {
			idx.scanIdxPresent = true
			continue
		}
		if idx.scanDataLen == 0 {
			if l := fr.RawSeriesAt(i).Len(); l > 0 {
				idx.scanDataLen = l
			}
		}
	}
	var toAdd int
	for _, idx := range w.internal {
		if idx.shouldAutoStamp() {
			toAdd++
		}
	}
	if toAdd == 0 {
		return fr
	}
	fr = fr.Grow(toAdd)
	now := telem.Now()
	for _, idx := range w.internal {
		if !idx.shouldAutoStamp() {
			continue
		}
		fr = idx.appendAutoStamp(fr, now)
	}
	return fr
}

// propagateAuthority synchronizes each idxWriter's per-data-channel authority tracking
// with the incoming SetAuthority config and augments cfg with an updated authority for
// each implicit index whose referencing data channels' max may have changed. Broadcast
// calls (cfg.Channels empty) are forwarded unchanged — the caller already applies the
// broadcast across every channel in the writer, including indexes — but the tracked
// state is refreshed so the next per-channel call computes correctly. Indexes the
// caller explicitly included in cfg.Channels are left untouched.
//
// propagateAuthority may append to cfg.Channels and cfg.Authorities. The streamWriter
// treats a dequeued WriterRequest as exclusively owned, so callers must not retain or
// share the underlying arrays of req.Config.Channels / req.Config.Authorities once the
// request has been sent.
func (w *streamWriter) propagateAuthority(cfg WriterConfig) WriterConfig {
	if len(cfg.Channels) == 0 {
		for _, idx := range w.internal {
			idx.broadcastDataAuth(cfg.Authorities[0])
		}
		return cfg
	}
	for _, idx := range w.internal {
		idx.setAuthExplicit = false
	}
	nExplicit := len(cfg.Channels)
	// If the caller broadcast a single authority across multiple channels, materialize
	// a per-channel authorities slice so the loop below can index Authorities[i]
	// directly and the implicit-index append loop has the right starting length.
	if len(cfg.Authorities) < nExplicit {
		auths := make([]xcontrol.Authority, nExplicit, nExplicit+len(w.internal))
		for i := range auths {
			auths[i] = cfg.Authorities[0]
		}
		cfg.Authorities = auths
	}
	for i, k := range cfg.Channels {
		idx, ok := w.keyToIdx[k]
		if !ok {
			continue
		}
		if k == idx.idx.ch.Key {
			idx.setAuthExplicit = true
			continue
		}
		idx.setDataAuth(k, cfg.Authorities[i])
	}
	for _, idx := range w.internal {
		if !idx.writingToIdx || idx.setAuthExplicit {
			continue
		}
		cfg.Channels = append(cfg.Channels, idx.idx.ch.Key)
		cfg.Authorities = append(cfg.Authorities, idx.maxDataAuth())
	}
	return cfg
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
	var err error
	parentUpdate := ControlUpdate{Transfers: make([]control.Transfer, 0, len(w.internal))}
	for _, idx := range w.internal {
		u, errClose := idx.Close()
		if errClose != nil {
			err = errors.Join(err, errClose)
		} else {
			parentUpdate.Transfers = append(parentUpdate.Transfers, u.Transfers...)
		}
	}
	if w.virtual.internal != nil {
		u, errClose := w.virtual.Close()
		if errClose != nil {
			err = errors.Join(err, errClose)
		} else {
			parentUpdate.Transfers = append(parentUpdate.Transfers, u.Transfers...)
		}
	}

	if len(parentUpdate.Transfers) > 0 {
		_ = w.updateDBControl(ctx, parentUpdate)
	}

	if digestWriter, ok := w.virtual.internal[w.virtual.digestKey]; ok {
		// When digest writer closes, we do not (and cannot) send an update.
		if _, digestErr := digestWriter.Close(); digestErr != nil {
			return digestErr
		}
	}

	return err
}

type unaryWriterState struct {
	unary.Writer
	timesWritten int
}

// idxWriter is a writer to a set of channels that all share the same index.
type idxWriter struct {
	internal map[ChannelKey]*unaryWriterState
	// dataAuth tracks the most recent control authority for each data channel in this
	// group (i.e. the keys of internal excluding the index itself). Populated only when
	// the streamWriter has AutoIndex enabled and writingToIdx is true; updated by
	// SetAuthority calls so that maxDataAuth can recompute the implicit index's
	// authority as the max across its referencing data channels.
	dataAuth map[ChannelKey]xcontrol.Authority
	idx      struct {
		// Index is the index used to resolve timestamps for domains in the DB.
		*index.Domain
		// Key is the channel key of the index.
		ch channel.Channel
		// highWaterMark is the last timestamp written to the index — whether by
		// auto-stamping or by a caller-provided series. Drives commit resolution
		// (resolveCommitEnd). Only relevant when writingToIdx is true.
		highWaterMark telem.TimeStamp
		// autoStampClock is the last timestamp emitted by appendAutoStamp on this
		// index. Advanced only by auto-stamping; caller-provided index timestamps do
		// not move it. Used to keep auto-stamps strictly monotonic across Write calls
		// without inheriting future timestamps the caller wrote explicitly.
		autoStampClock telem.TimeStamp
	}
	// lastCommitEnd stores the end timestamp from the last successful commit,
	// returned when Commit is called with no new data to commit.
	lastCommitEnd telem.TimeStamp
	// sampleCount is the total number of samples written to the index as if it were a
	// single logical channel. i.e. N channels with M samples will result in a sample
	// count of M.
	sampleCount int64
	start       telem.TimeStamp
	// scanDataLen is a transient scratch field set during the single frame scan in
	// streamWriter.autoStamp. Holds the length of the first non-empty data channel
	// observed for this group (the size of the auto-stamped series). Reset to zero at
	// the start of every autoStamp call.
	scanDataLen int64
	// numWriteCalls tracks the number of write calls made to the idxWriter.
	numWriteCalls   int
	domainAlignment uint32
	// writingToIdx is true when the Write is writing to the index channel. This is
	// typically true, which allows us to avoid unnecessary lookups.
	writingToIdx bool
	// hasUncommittedData tracks whether new data has been written since the last
	// successful commit. This prevents stale commits when a control transfer
	// advances the domain writer's prevCommit beyond this writer's highWaterMark.
	hasUncommittedData bool
	// scanIdxPresent is a transient scratch field set during the single frame scan in
	// streamWriter.autoStamp. True when the caller's frame already contains this
	// idxWriter's index key (in which case no stamping is needed). Reset to false at
	// the start of every autoStamp call.
	scanIdxPresent bool
	// setAuthExplicit is a transient scratch field used during the per-channel scan in
	// streamWriter.propagateAuthority. True when the caller explicitly named this
	// idxWriter's index key in the SetAuthority config, in which case the index
	// authority is left untouched. Reset to false at the start of every
	// propagateAuthority call.
	setAuthExplicit bool
}

func (w *idxWriter) resetAutoStampScan() { w.scanIdxPresent = false; w.scanDataLen = 0 }

func (w *idxWriter) shouldAutoStamp() bool {
	return w.writingToIdx && !w.scanIdxPresent && w.scanDataLen > 0
}

func (w *idxWriter) appendAutoStamp(fr Frame, now telem.TimeStamp) Frame {
	t0 := max(now, w.idx.autoStampClock+1)
	// Allocate the Series's byte buffer once and reinterpret it as []TimeStamp via
	// unsafe.CastSlice so timestamps are written directly into the backing array. Going
	// through NewSeriesV(stamps...) would perform two allocations instead of one.
	series := telem.MakeSeries(telem.TimeStampT, int(w.scanDataLen))
	stamps := unsafe.CastSlice[byte, telem.TimeStamp](series.Data)
	for j := range stamps {
		stamps[j] = t0 + telem.TimeStamp(j)
	}
	w.idx.autoStampClock = stamps[len(stamps)-1]
	return fr.Append(w.idx.ch.Key, series)
}

// setDataAuth records auth as the authority for k if k is one of this group's data
// channels and returns true. Returns false if k is not in this group (the streamWriter
// is expected to try the next idxWriter in that case).
func (w *idxWriter) setDataAuth(k ChannelKey, auth xcontrol.Authority) bool {
	if _, isData := w.dataAuth[k]; !isData {
		return false
	}
	w.dataAuth[k] = auth
	return true
}

// broadcastDataAuth sets every data channel's authority in this group to auth. Used
// when the caller issues a writer-wide SetAuthority (cfg.Channels empty), which the
// downstream layer applies to every channel; this keeps the per-data-channel record
// consistent so a subsequent per-channel SetAuthority computes the implicit index's max
// correctly.
func (w *idxWriter) broadcastDataAuth(auth xcontrol.Authority) {
	for k := range w.dataAuth {
		w.dataAuth[k] = auth
	}
}

// maxDataAuth returns the maximum recorded authority across this group's data channels.
// Returns 0 when the group has no data channels (which cannot occur for a writingToIdx
// idxWriter under AutoIndex).
func (w *idxWriter) maxDataAuth() xcontrol.Authority {
	var max xcontrol.Authority
	for _, a := range w.dataAuth {
		if a > max {
			max = a
		}
	}
	return max
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
		series := fr.RawSeriesAt(i)
		if series.Len() == 0 {
			continue
		}
		uWriter, ok := w.internal[key]
		if !ok {
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
			w.hasUncommittedData = true
		}
		series.Alignment = alignment
		fr.SetRawSeriesAt(i, series)
	}
	if errors.Is(accumulatedErr, xcontrol.ErrUnauthorized) {
		w.hasUncommittedData = false
	}
	return fr, accumulatedErr
}

func (w *idxWriter) Commit(ctx context.Context) (telem.TimeStamp, error) {
	if w.sampleCount == 0 || !w.hasUncommittedData {
		return w.lastCommitEnd, nil
	}
	end, err := w.resolveCommitEnd(ctx)
	if err != nil {
		return 0, err
	}
	// because the range is exclusive, we need to add 1 nanosecond to the end
	end.Lower++
	for _, chW := range w.internal {
		err = errors.Join(err, chW.CommitWithEnd(ctx, end.Lower))
	}
	if err == nil {
		w.lastCommitEnd = end.Lower
	}
	// Reset on ErrUnauthorized as well: after a control transfer, the new controller
	// may advance prevCommit, making any future commit with this writer's stale
	// highWaterMark invalid.
	if err == nil || errors.Is(err, xcontrol.ErrUnauthorized) {
		w.hasUncommittedData = false
	}
	return end.Lower, err
}

func (w *idxWriter) Close() (ControlUpdate, error) {
	var err error
	update := ControlUpdate{
		Transfers: make([]control.Transfer, 0, len(w.internal)),
	}
	for _, uWriter := range w.internal {
		transfer, closeErr := uWriter.Close()
		if closeErr != nil {
			err = errors.Join(err, closeErr)
		} else if transfer.Occurred() {
			update.Transfers = append(update.Transfers, transfer)
		}
	}
	return update, err
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
		expectedChannels           = len(w.internal)
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
		ch := uWriter.Channel
		if lengthOfFrame == -1 {
			if !ch.DataType.IsVariable() && s.DataType.Density() == telem.UnknownDensity {
				return invalidDataTypeError(ch, s.DataType)
			}
			lengthOfFrame = s.Len()
		}
		if uWriter.timesWritten == w.numWriteCalls {
			return oneSeriesPerChannelError(ch)
		}
		uWriter.timesWritten++

		if s.Len() != lengthOfFrame {
			return sameLengthForAllSeriesError(ch, lengthOfFrame, s)
		}
		numChannelsWrittenTo++
	}

	if numChannelsWrittenTo == 0 {
		return nil
	}

	if numChannelsWrittenTo != expectedChannels {
		if numChannelsWrittenTo < expectedChannels {
			keys := set.New(fr.KeysSlice()...)
			for k, db := range w.internal {
				if !keys.Contains(k) {
					dataChannels := make([]Channel, 0, expectedChannels)
					for otherK, other := range w.internal {
						if otherK != k {
							dataChannels = append(dataChannels, other.Channel)
						}
					}
					return missingChannelError(w.idx.ch, db.Channel, dataChannels)
				}
			}
		}
		err := incorrectNumberOfSeriesError(expectedChannels, numChannelsWrittenTo)
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

// virtualGroup tracks the shared write alignment for an index group of virtual
// channels within a single writer. Every member of the group is stamped with the
// group's current alignment, and writes to the group's index channel advance the
// sample position, so series written across the group correlate sample-for-sample.
type virtualGroup struct {
	// indexKey is the key of the group's index channel.
	indexKey ChannelKey
	// alignment is the group's current write position.
	alignment telem.Alignment
}

type virtualWriter struct {
	internal map[ChannelKey]*virtual.Writer
	// groups maps each virtual channel belonging to an index group to the group's
	// shared alignment state. Channels without an index are absent.
	groups    map[ChannelKey]*virtualGroup
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
		if g, grouped := w.groups[k]; grouped {
			alignment = g.alignment
		}
		s.Alignment = alignment
		fr.SetRawSeriesAt(rawI, s)
	}
	// Advance group alignments only after stamping the whole frame so that every
	// series in it, including the index series itself, shares the same alignment.
	for rawI, k := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) || lo.Contains(*filterUnauthorized, k) {
			continue
		}
		if g, grouped := w.groups[k]; grouped && g.indexKey == k {
			g.alignment = g.alignment.AddSamples(uint32(fr.RawSeriesAt(rawI).Len()))
		}
	}
	return fr, accumulatedErr
}

func (w virtualWriter) Close() (ControlUpdate, error) {
	var err error
	update := ControlUpdate{Transfers: make([]control.Transfer, 0, len(w.internal))}
	for _, chW := range w.internal {
		// We do not want to clean up the digest channel since we want to use it to send
		// updates for closures.
		if chW.Channel.Key != w.digestKey {
			transfer, closeErr := chW.Close()
			if closeErr != nil {
				err = errors.Join(err, closeErr)
			} else if transfer.Occurred() {
				update.Transfers = append(update.Transfers, transfer)
			}
		}
	}
	return update, err
}
