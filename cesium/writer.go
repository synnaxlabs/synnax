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
	"slices"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/confluence"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/stringer"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Writer writes frames to a set of channels and commits them. A Writer is not safe for
// concurrent use.
//
// The first error that is not control.ErrUnauthorized breaks the writer: every later
// call returns that error, and Close returns it after it releases the writer. An error
// that wraps control.ErrUnauthorized leaves the writer open, so that it can regain
// control.
type Writer struct {
	WriterConfig
	internal []*idxWriter
	virtual  *virtualWriter
	// keyToIdx maps every channel key the writer is responsible for to its owning
	// idxWriter.
	keyToIdx           map[ChannelKey]*idxWriter
	relay              confluence.Inlet[relayResponse]
	relayStopped       <-chan struct{}
	updateDBControl    func(u ControlUpdate)
	authorizeEndDomain func(index ChannelKey, own map[ChannelKey]*unaryWriterState) error
	// err is the error that broke the writer.
	err    error
	closed bool
}

// ErrWriterClosed is returned by a call to a Writer that is closed.
var ErrWriterClosed = resource.NewClosedError("cesium.writer")

// check returns the error that a call to the writer must return before it runs.
func (w *Writer) check() error {
	if w.err != nil {
		return w.err
	}
	if w.closed {
		return ErrWriterClosed
	}
	return nil
}

// fail breaks the writer with err unless err wraps control.ErrUnauthorized, and
// returns err.
func (w *Writer) fail(err error) error {
	if err != nil && !errors.Is(err, xcontrol.ErrUnauthorized) && w.err == nil {
		w.err = err
	}
	return err
}

// Close releases the writer. It does not commit. It returns the error that broke the
// writer, if any, joined with any error from the release. A second Close returns the
// same error.
func (w *Writer) Close() error {
	if w.closed {
		return w.err
	}
	w.closed = true
	var (
		err    error
		update = ControlUpdate{
			Transfers: make([]xcontrol.Transfer[ChannelKey], 0, len(w.internal)),
		}
	)
	for _, idx := range w.internal {
		u, errClose := idx.Close()
		if errClose != nil {
			err = errors.Join(err, errClose)
		} else {
			update.Transfers = append(update.Transfers, u.Transfers...)
		}
	}
	if w.virtual.internal != nil {
		u, errClose := w.virtual.Close()
		if errClose != nil {
			err = errors.Join(err, errClose)
		} else {
			update.Transfers = append(update.Transfers, u.Transfers...)
		}
	}
	if len(update.Transfers) > 0 {
		w.updateDBControl(update)
	}
	w.err = errors.Combine(w.err, err)
	return w.err
}

// SetAuthority sets the control authority of the writer over its channels. Naming a
// data channel also moves its written index to the max across that index's data
// channels, unless the index is named explicitly.
func (w *Writer) SetAuthority(cfg WriterConfig) error {
	if err := w.check(); err != nil {
		return err
	}
	if len(cfg.Authorities) == 0 {
		return nil
	}
	if len(cfg.Authorities) != 1 &&
		len(cfg.Authorities) != len(cfg.Channels) {
		return w.fail(errors.Wrap(
			validate.ErrValidation,
			"authority count must be 1 or equal to channel count",
		))
	}
	cfg = w.propagateAuthority(cfg)
	var (
		u = ControlUpdate{
			Transfers: make([]xcontrol.Transfer[ChannelKey], 0, len(w.internal)),
		}
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
		w.updateDBControl(u)
	}
	return nil
}

// Write writes fr and returns it with the alignment and time range of each written
// series set. When the writer lacks control of some channels, Write writes the others
// and returns the frame without the refused channels, with an error that wraps
// control.ErrUnauthorized.
func (w *Writer) Write(ctx context.Context, fr Frame) (Frame, error) {
	if err := w.check(); err != nil {
		return Frame{}, err
	}
	if *w.AutoIndex {
		fr = w.autoStamp(fr)
	}
	var (
		unauthorized        error
		err                 error
		excludeUnauthorized []ChannelKey
	)
	for _, idx := range w.internal {
		if fr, err = idx.write(&excludeUnauthorized, fr); err != nil {
			if !errors.Is(err, xcontrol.ErrUnauthorized) {
				return Frame{}, w.fail(err)
			}
			unauthorized = err
			continue
		}
		if *w.EnableAutoCommit {
			if _, err = idx.Commit(ctx); err != nil {
				if !errors.Is(err, xcontrol.ErrUnauthorized) {
					return Frame{}, w.fail(err)
				}
				unauthorized = err
			}
		}
	}
	if w.virtual.internal != nil {
		if fr, err = w.virtual.write(&excludeUnauthorized, fr); err != nil {
			if !errors.Is(err, xcontrol.ErrUnauthorized) {
				return Frame{}, w.fail(err)
			}
			unauthorized = err
		}
	}
	fr = fr.ExcludeKeys(excludeUnauthorized)
	if w.Mode.Stream() {
		select {
		case w.relay.Inlet() <- relayResponse{frame: fr, group: w.ControlSubject.Group}:
		case <-w.relayStopped:
		}
	}
	return fr, unauthorized
}

// autoStamp injects a TimeStamp series for each idxWriter whose index channel is
// referenced by the writer but absent from fr. It runs in O(KeysInFrame + IndexGroups):
// a single pass over fr's keys uses keyToIdx to resolve each key's idxWriter group in
// O(1) and records whether that group's index is present and the length of any data
// channel for the group. A subsequent pass over idxWriters counts how many stamps are
// needed, grows fr by exactly that amount, and performs the appends.
func (w *Writer) autoStamp(fr Frame) Frame {
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
// each written index whose referencing data channels' max may have changed. Broadcast
// calls (cfg.Channels empty) are forwarded unchanged — the caller already applies the
// broadcast across every channel in the writer, including indexes — but the tracked
// state is refreshed so the next per-channel call computes correctly. Indexes the
// caller explicitly included in cfg.Channels are left untouched.
//
// propagateAuthority may append to cfg.Channels and cfg.Authorities, so a caller of
// SetAuthority must not retain or share the underlying arrays of cfg.Channels and
// cfg.Authorities.
func (w *Writer) propagateAuthority(cfg WriterConfig) WriterConfig {
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
		if !idx.writingToIdx || idx.setAuthExplicit || len(idx.dataAuth) == 0 {
			continue
		}
		cfg.Channels = append(cfg.Channels, idx.idx.ch.Key)
		cfg.Authorities = append(cfg.Authorities, idx.maxDataAuth())
	}
	return cfg
}

// Commit makes the written samples available to reads and returns the end of the
// commit. When the writer lacks control of a channel with samples to commit, Commit
// returns an error that wraps control.ErrUnauthorized and a zero end, and the writer
// stays open so that it can regain control.
func (w *Writer) Commit(ctx context.Context) (telem.TimeStamp, error) {
	if err := w.check(); err != nil {
		return 0, err
	}
	var (
		maxTS        = telem.TimeStampMin
		unauthorized error
	)
	for _, idxW := range w.internal {
		ts, err := idxW.Commit(ctx)
		if errors.Is(err, xcontrol.ErrUnauthorized) {
			unauthorized = err
			continue
		}
		if err != nil {
			return 0, w.fail(err)
		}
		if ts > maxTS {
			maxTS = ts
		}
	}
	if unauthorized != nil {
		return 0, unauthorized
	}
	return maxTS, nil
}

// EndDomain commits the writer, then ends the domains of the given indexes and of the
// data channels they index. The next write to an index starts a new domain at its first
// timestamp, which frees the time before that timestamp for another writer. Indexes the
// writer does not write are ignored, and an index with no data keeps its domains.
// EndDomain returns an error that wraps control.ErrUnauthorized if another writer
// controls an index or one of its data channels. The index then keeps its domain, and
// some of its data channels may end theirs.
func (w *Writer) EndDomain(ctx context.Context, indexes ...ChannelKey) error {
	if err := w.check(); err != nil {
		return err
	}
	var unauthorized error
	for _, idx := range w.internal {
		if !idx.writingToIdx || !slices.Contains(indexes, idx.idx.ch.Key) {
			continue
		}
		if err := idx.endDomain(ctx, w.authorizeEndDomain); err != nil {
			if !errors.Is(err, xcontrol.ErrUnauthorized) {
				return w.fail(err)
			}
			unauthorized = err
		}
	}
	return unauthorized
}

type unaryWriterState struct {
	unary.Writer
	lastWriteCall int
}

// idxWriter is a writer to a set of channels that all share the same index.
type idxWriter struct {
	internal map[ChannelKey]*unaryWriterState
	idx      struct {
		// Index is the index used to resolve timestamps for domains in the DB.
		*index.Domain
		// Key is the channel key of the index.
		ch channel.Channel
		// db is the unary database of the index channel. Consulted for the current
		// write cursor when this group does not write the index itself.
		db *unary.DB
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
	// numWriteCalls tracks the number of write calls made to the idxWriter.
	numWriteCalls int
	// sampleCount is the total number of samples written to the index as if it were a
	// single logical channel. i.e. N channels with M samples will result in a sample
	// count of M.
	sampleCount int64
	start       telem.TimeStamp
	// anchor is the index write cursor observed at this group's first write, valid only
	// when writingToIdx is false. Every subsequent write is stamped at anchor plus the
	// number of samples the group has already written, so a group that does not write
	// the index still lands in the index's sample space.
	anchor telem.Alignment
	// anchorSet reports whether anchor has been resolved.
	anchorSet bool
	// writingToIdx is true when the Write is writing to the index channel. This is
	// typically true, which allows us to avoid unnecessary lookups.
	writingToIdx bool
	// hasUncommittedData tracks whether new data has been written since the last
	// successful commit. This prevents stale commits when a control transfer
	// advances the domain writer's prevCommit beyond this writer's highWaterMark.
	hasUncommittedData bool
	// lastCommitEnd stores the end timestamp from the last successful commit,
	// returned when Commit is called with no new data to commit.
	lastCommitEnd telem.TimeStamp
	// group shows every channel of a commit to readers in one step.
	group domain.Group
	// dataAuth tracks the most recent control authority for each data channel in this
	// group (i.e. the keys of internal excluding the index itself). Populated when
	// writingToIdx is true; updated by SetAuthority calls so that maxDataAuth can
	// recompute the written index's authority as the max across its referencing data
	// channels.
	dataAuth map[ChannelKey]xcontrol.Authority
	// scanIdxPresent is a transient scratch field set during the single frame scan in
	// Writer.autoStamp. True when the caller's frame already contains this
	// idxWriter's index key (in which case no stamping is needed). Reset to false at
	// the start of every autoStamp call.
	scanIdxPresent bool
	// scanDataLen is a transient scratch field set during the single frame scan in
	// Writer.autoStamp. Holds the length of the first non-empty data channel
	// observed for this group (the size of the auto-stamped series). Reset to zero at
	// the start of every autoStamp call.
	scanDataLen int64
	// setAuthExplicit is a transient scratch field used during the per-channel scan in
	// Writer.propagateAuthority. True when the caller explicitly named this
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
	series := telem.MakeSeries(telem.TimestampT, int(w.scanDataLen))
	stamps := unsafe.CastSlice[byte, telem.TimeStamp](series.Data)
	for j := range stamps {
		stamps[j] = t0 + telem.TimeStamp(j)
	}
	w.idx.autoStampClock = stamps[len(stamps)-1]
	return fr.Append(w.idx.ch.Key, series)
}

// setDataAuth records auth as the authority for k if k is one of this group's data
// channels and returns true. Returns false if k is not in this group (the Writer is
// expected to try the next idxWriter in that case).
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
// Returns 0 when the group has no data channels; propagateAuthority skips such groups.
func (w *idxWriter) maxDataAuth() xcontrol.Authority {
	var maxAuth xcontrol.Authority
	for _, a := range w.dataAuth {
		if a > maxAuth {
			maxAuth = a
		}
	}
	return maxAuth
}

// resolveAlignment returns the alignment for the current write when no index series is
// available to establish it. A group that writes the index takes the index's cursor
// directly. A group that does not anchors on the cursor covering its start timestamp,
// resolved once, and offsets by the number of samples it has written since, so the
// index is consulted once per writer rather than once per write.
func (w *idxWriter) resolveAlignment() (telem.Alignment, error) {
	if w.writingToIdx {
		return w.idx.db.WriteCursor(w.start)
	}
	if !w.anchorSet {
		anchor, err := w.idx.db.WriteCursor(w.start)
		if err != nil {
			return 0, err
		}
		w.anchor = anchor
		w.anchorSet = true
	}
	return w.anchor.AddSamples(uint32(w.sampleCount)), nil
}

// indexWrite is what writing the index series of a frame establishes for the data
// series of the frame.
type indexWrite struct {
	alignment telem.Alignment
	timeRange telem.TimeRange
	written   bool
	// unauthorized is true when the writer does not control the index.
	unauthorized bool
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
	// Pass 1: the index channel establishes the alignment for the whole frame. Its
	// sample position is the space every channel it indexes is expressed in, so it must
	// be written before any of them.
	idx, err := w.writeIndex(excludeUnauthorized, fr)
	if err != nil && !idx.unauthorized {
		return fr, err
	}
	var (
		accumulatedErr         = err
		alignment              = idx.alignment
		alignmentSet           = idx.written
		incrementedSampleCount = idx.written
	)
	// Pass 2: every data channel is stamped with the index's alignment rather than its
	// own position in its own domain. The alignment is resolved lazily so a frame that
	// touches none of this group's data channels costs nothing.
	for i, key := range fr.RawKeys() {
		uWriter, series, ok := w.dataSeries(fr, i, key)
		if !ok {
			continue
		}
		// Losing control of the index means there is no position to express these
		// samples against. Writing them anyway would stamp them in a controlling
		// writer's space, so the whole group is held back instead.
		if idx.unauthorized {
			*excludeUnauthorized = append(*excludeUnauthorized, key)
			continue
		}
		if !alignmentSet {
			if alignment, err = w.resolveAlignment(); err != nil {
				return fr, err
			}
			alignmentSet = true
		}
		if idx.written {
			uWriter.SetDomainStart(idx.timeRange.Start)
		}
		if _, err := uWriter.WriteAt(series, alignment); err != nil {
			accumulatedErr = err
			if !errors.Is(accumulatedErr, xcontrol.ErrUnauthorized) {
				return fr, accumulatedErr
			}
			*excludeUnauthorized = append(*excludeUnauthorized, key)
			continue
		}
		if !incrementedSampleCount {
			w.countSamples(alignment, series.Len())
			incrementedSampleCount = true
		}
		series.Alignment = alignment
		if idx.written {
			series.TimeRange = idx.timeRange
		}
		fr.SetRawSeriesAt(i, series)
	}
	if errors.Is(accumulatedErr, xcontrol.ErrUnauthorized) {
		w.hasUncommittedData = false
	}
	return fr, accumulatedErr
}

// dataSeries returns the series at raw position i of fr and the writer of its data
// channel key. It returns false when the series is excluded or empty, or when the group
// does not write key as a data channel.
func (w *idxWriter) dataSeries(
	fr Frame,
	i int,
	key ChannelKey,
) (*unaryWriterState, telem.Series, bool) {
	if key == w.idx.ch.Key || fr.ShouldExcludeRaw(i) {
		return nil, telem.Series{}, false
	}
	series := fr.RawSeriesAt(i)
	if series.Len() == 0 {
		return nil, telem.Series{}, false
	}
	uWriter, ok := w.internal[key]
	return uWriter, series, ok
}

// writeIndex writes the index series of fr when the group writes its index. When the
// writer does not control the index, it adds the index to excludeUnauthorized and
// returns the error with unauthorized set.
func (w *idxWriter) writeIndex(
	excludeUnauthorized *[]ChannelKey,
	fr Frame,
) (idx indexWrite, err error) {
	if !w.writingToIdx {
		return idx, nil
	}
	for i, key := range fr.RawKeys() {
		if key != w.idx.ch.Key || fr.ShouldExcludeRaw(i) {
			continue
		}
		series := fr.RawSeriesAt(i)
		uWriter, ok := w.internal[key]
		if series.Len() == 0 || !ok {
			return idx, nil
		}
		if err = w.updateHighWater(series); err != nil {
			return idx, err
		}
		first := series.ValueAt[telem.TimeStamp](0)
		uWriter.SetDomainStart(first)
		if idx.alignment, err = uWriter.Write(series); err != nil {
			if !errors.Is(err, xcontrol.ErrUnauthorized) {
				return idx, err
			}
			*excludeUnauthorized = append(*excludeUnauthorized, key)
			idx.unauthorized = true
			return idx, err
		}
		idx.written = true
		w.sampleCount = int64(idx.alignment.SampleIndex()) + series.Len()
		w.hasUncommittedData = true
		series.Alignment = idx.alignment
		// The index series carries the timestamps for the whole group, so data series
		// can be stamped with a real time range without consulting the index on disk.
		idx.timeRange = telem.TimeRange{Start: first, End: w.idx.highWaterMark + 1}
		series.TimeRange = idx.timeRange
		fr.SetRawSeriesAt(i, series)
		return idx, nil
	}
	return idx, nil
}

// countSamples records the n samples of the first data series a write stamps with
// alignment. A data channel's alignment is expressed in its index's sample space, so it
// does not report a position measured from this writer's start. resolveCommitEnd needs
// that measurement, so the group counts its own.
func (w *idxWriter) countSamples(alignment telem.Alignment, n int64) {
	if w.writingToIdx {
		w.sampleCount = int64(alignment.SampleIndex()) + n
	} else {
		w.sampleCount += n
	}
	w.hasUncommittedData = true
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
	// No reader sees a channel of the commit before every channel has committed, so no
	// read finds an index sample without its data.
	defer w.group.Publish()
	// The index persists first: after a restart, a reader resolves a data domain
	// through the index.
	if w.writingToIdx {
		err = w.internal[w.idx.ch.Key].CommitWithEnd(ctx, end.Lower, &w.group)
	}
	// Every channel shares the same end, so their errors repeat. Keep the first.
	for key, chW := range w.internal {
		if key == w.idx.ch.Key {
			continue
		}
		if cErr := chW.CommitWithEnd(ctx, end.Lower, &w.group); err == nil {
			err = cErr
		}
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
	if err != nil {
		return 0, err
	}
	return end.Lower, nil
}

func (w *idxWriter) endDomain(
	ctx context.Context,
	authorize func(index ChannelKey, own map[ChannelKey]*unaryWriterState) error,
) error {
	if _, err := w.Commit(ctx); err != nil {
		return err
	}
	if err := authorize(w.idx.ch.Key, w.internal); err != nil {
		return err
	}
	// A data domain that outlives its index domain cannot resolve its samples, so the
	// index ends last, and only if every data domain ended.
	for key, chW := range w.internal {
		if key == w.idx.ch.Key {
			continue
		}
		if err := chW.EndDomain(ctx); err != nil {
			return err
		}
	}
	return w.internal[w.idx.ch.Key].EndDomain(ctx)
}

func (w *idxWriter) Close() (ControlUpdate, error) {
	var err error
	update := ControlUpdate{
		Transfers: make([]xcontrol.Transfer[ChannelKey], 0, len(w.internal)),
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
length %d, while series for channel %v has length %d. See https://docs.synnaxlabs.com/reference/client/write-data#rule-1
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
			See https://docs.synnaxlabs.com/reference/client/write-data#the-rules-of-writes
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
			if !ch.DataType.IsVariable() &&
				s.DataType.Density() == telem.UnknownDensity {
				return invalidDataTypeError(ch, s.DataType)
			}
			lengthOfFrame = s.Len()
		}
		if uWriter.lastWriteCall == w.numWriteCalls {
			return oneSeriesPerChannelError(ch)
		}
		uWriter.lastWriteCall = w.numWriteCalls

		if s.Len() != lengthOfFrame {
			return sameLengthForAllSeriesError(ch, lengthOfFrame, s)
		}
		numChannelsWrittenTo++
	}

	if numChannelsWrittenTo == 0 {
		return nil
	}

	if numChannelsWrittenTo != expectedChannels {
		return w.seriesCountError(fr, numChannelsWrittenTo)
	}

	return nil
}

// seriesCountError returns the error for a frame fr that holds numChannelsWrittenTo
// series for the group instead of one series per channel.
func (w *idxWriter) seriesCountError(fr Frame, numChannelsWrittenTo int) error {
	expectedChannels := len(w.internal)
	if numChannelsWrittenTo < expectedChannels {
		keys := set.New(fr.KeysSlice()...)
		for k, db := range w.internal {
			if !keys.Contains(k) {
				return missingChannelError(w.idx.ch, db.Channel, w.channelsExcept(k))
			}
		}
	}
	err := incorrectNumberOfSeriesError(expectedChannels, numChannelsWrittenTo)
	zap.S().DPanic(err.Error())
	return err
}

// channelsExcept returns the channels of the group other than key.
func (w *idxWriter) channelsExcept(key ChannelKey) []Channel {
	channels := make([]Channel, 0, len(w.internal))
	for k, other := range w.internal {
		if k != key {
			channels = append(channels, other.Channel)
		}
	}
	return channels
}

func (w *idxWriter) updateHighWater(s telem.Series) error {
	if s.DataType != telem.TimestampT && s.DataType != telem.Int64T {
		return invalidDataTypeError(w.idx.ch, s.DataType)
	}
	w.idx.highWaterMark = s.ValueAt[telem.TimeStamp](-1)
	return nil
}

// resolveCommitEnd returns the end timestamp for a commit. For an index channel, this
// returns the high watermark. For a non-index channel, this returns a stamp to the
// approximation of the end.
func (w *idxWriter) resolveCommitEnd(
	ctx context.Context,
) (index.TimeStampApproximation, error) {
	if w.writingToIdx {
		return index.Exactly(w.idx.highWaterMark), nil
	}
	approx, err := w.idx.Stamp(ctx, w.start, w.sampleCount-1, index.MustBeContinuous)
	if err != nil {
		return approx, err
	}
	// An inexact approximation means w.start is not an exact sample in the index, so
	// there is no defined timestamp for the samples written by this writer. Committing
	// anyway would use the approximation's zero-valued lower bound as the commit end.
	if !approx.Exact() {
		return approx, index.NewDiscontinuousStampError(w.start)
	}
	return approx, nil
}

type virtualWriter struct {
	internal map[ChannelKey]*virtual.Writer
}

func (w virtualWriter) write(
	filterUnauthorized *[]ChannelKey,
	fr Frame,
) (Frame, error) {
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
	var err error
	update := ControlUpdate{
		Transfers: make([]xcontrol.Transfer[ChannelKey], 0, len(w.internal)),
	}
	for _, chW := range w.internal {
		transfer, closeErr := chW.Close()
		if closeErr != nil {
			err = errors.Join(err, closeErr)
		} else if transfer.Occurred() {
			update.Transfers = append(update.Transfers, transfer)
		}
	}
	return update, err
}
