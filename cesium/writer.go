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
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/stringer"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Writer writes frames to one clock and the channels it times, and commits them. A
// Writer is not safe for concurrent use.
//
// The first error breaks the writer: every later call returns that error, and Close
// returns it after it releases the writer.
type Writer struct {
	WriterConfig
	idx *idxWriter
	// err is the error that broke the writer.
	err    error
	closed bool
	// excluded holds the keys that a Write leaves out, kept between calls so that a
	// write that skips allocates nothing.
	excluded []ChannelKey
	// log is the writer of the pointer log layout, and nil on the old layout.
	log *logWriter
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

// fail breaks the writer with err and returns err.
func (w *Writer) fail(err error) error {
	if err != nil && w.err == nil {
		w.err = err
	}
	return err
}

// Sync makes the commits of the writer durable. On the old layout, it persists the
// index of each channel that the writer stores, and syncs no file.
func (w *Writer) Sync() error {
	if err := w.check(); err != nil {
		return err
	}
	if w.log != nil {
		return w.fail(w.log.sync())
	}
	return w.fail(w.persist())
}

// persist persists the clock before the data channels it times, as Commit commits
// them, so a reopen never sees data that its clock cannot resolve.
func (w *Writer) persist() error {
	idx := w.idx
	if idx.writingToIdx {
		if err := idx.internal[idx.idx.ch.Key].Persist(); err != nil {
			return err
		}
	}
	for key, u := range idx.internal {
		if key == idx.idx.ch.Key {
			continue
		}
		if err := u.Persist(); err != nil {
			return err
		}
	}
	return nil
}

// Close releases the writer and makes its commits durable, as Sync does. It does not
// commit. It returns the error that broke the writer, if any, joined with any error
// from the release. A second Close returns the same error.
func (w *Writer) Close() error {
	if w.closed {
		return w.err
	}
	w.closed = true
	if w.log != nil {
		w.err = errors.Join(w.err, w.log.close())
		return w.err
	}
	err := errors.Join(w.persist(), w.idx.Close())
	w.err = errors.Join(w.err, err)
	return w.err
}

// Write writes fr except the channels for which skip returns true, and returns it
// without them, with the alignment and time range of each written series set. A skipped
// index also skips the data channels it indexes in the writer. The returned bool is
// true when Write skipped a series of fr, and the next commit of that index then
// commits nothing. Write validates every series of fr. When skip returns true with no
// lock held, Write skips at once. If not, it calls skip again under the lock of each
// channel's claim. A nil skip skips nothing.
func (w *Writer) Write(
	ctx context.Context,
	fr Frame,
	skip func(ChannelKey) bool,
) (Frame, bool, error) {
	if err := w.check(); err != nil {
		return Frame{}, false, err
	}
	if w.log != nil {
		fr, skipped, err := w.log.write(fr, skip)
		if err != nil {
			return Frame{}, false, w.fail(err)
		}
		return fr, skipped, nil
	}
	var err error
	w.excluded = w.excluded[:0]
	if fr, err = w.idx.write(skip, &w.excluded, fr); err != nil {
		return Frame{}, false, w.fail(err)
	}
	return fr.ExcludeKeys(w.excluded), len(w.excluded) > 0, nil
}

// Commit makes the written samples of every channel except the channels for which skip
// returns true available to reads, and returns the end of the commit. The returned bool
// is true when it skipped a channel of an index with samples to commit, and the end is
// then zero. The samples of a skipped channel stay uncommitted. Commit calls skip as
// Write does. A nil skip skips nothing.
func (w *Writer) Commit(
	ctx context.Context,
	skip func(ChannelKey) bool,
) (telem.TimeStamp, bool, error) {
	if err := w.check(); err != nil {
		return 0, false, err
	}
	if w.log != nil {
		end, skipped, err := w.log.commit(skip)
		if err != nil {
			return 0, false, w.fail(err)
		}
		return end, skipped, nil
	}
	end, skipped, err := w.idx.Commit(ctx, skip)
	if err != nil {
		return 0, false, w.fail(err)
	}
	return end, skipped, nil
}

// EndStretch commits, then ends the stretch of the writer's clock and of the channels
// it times. The next write starts a new stretch at its first stamp, which frees the
// time before that stamp for another writer. A writer that does not write its clock
// only commits, and a clock with no data keeps its stretches.
func (w *Writer) EndStretch(ctx context.Context) error {
	if err := w.check(); err != nil {
		return err
	}
	if w.log != nil {
		return w.fail(w.log.endStretch())
	}
	if !w.idx.writingToIdx {
		_, _, err := w.Commit(ctx, nil)
		return err
	}
	return w.fail(w.idx.endDomain(ctx))
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
		// highWaterMark is the last timestamp written to the index. Drives commit
		// resolution (resolveCommitEnd). Only relevant when writingToIdx is true.
		highWaterMark telem.TimeStamp
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
	// successful commit. This prevents stale commits when another writer of the claim
	// advances the domain writer's prevCommit beyond this writer's highWaterMark.
	hasUncommittedData bool
	// lastCommitEnd stores the end timestamp from the last successful commit,
	// returned when Commit is called with no new data to commit.
	lastCommitEnd telem.TimeStamp
	// group shows every channel of a commit to readers in one step.
	group domain.Group
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
	// skipped is true when the write skips the index.
	skipped bool
}

func (w *idxWriter) write(
	skip func(ChannelKey) bool,
	excluded *[]ChannelKey,
	fr Frame,
) (Frame, error) {
	w.numWriteCalls++
	if err := w.validateWrite(fr); err != nil {
		return fr, err
	}
	// Pass 1: the index channel establishes the alignment for the whole frame. Its
	// sample position is the space every channel it indexes is expressed in, so it must
	// be written before any of them.
	idx, err := w.writeIndex(skip, excluded, fr)
	if err != nil {
		return fr, err
	}
	var (
		skipped                = idx.skipped
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
		// A skipped index leaves no position for these samples, so the group waits.
		if idx.skipped {
			*excluded = append(*excluded, key)
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
		_, keySkipped, wErr := uWriter.WriteAtUnless(series, alignment, skip)
		if wErr != nil {
			return fr, wErr
		}
		if keySkipped {
			*excluded = append(*excluded, key)
			skipped = true
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
	if skipped {
		w.hasUncommittedData = false
	}
	return fr, nil
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

// writeIndex writes the index series of fr when the group writes its index. When skip
// holds the index, it adds the index to excluded and returns with skipped set.
func (w *idxWriter) writeIndex(
	skip func(ChannelKey) bool,
	excluded *[]ChannelKey,
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
		if idx.alignment, idx.skipped, err = uWriter.WriteUnless(
			series,
			skip,
		); err != nil {
			return idx, err
		}
		if idx.skipped {
			*excluded = append(*excluded, key)
			return idx, nil
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

// Commit commits every channel of the group except the channels in skip. It returns
// true when it skipped a channel and the group had samples to commit.
func (w *idxWriter) Commit(
	ctx context.Context,
	skip func(ChannelKey) bool,
) (telem.TimeStamp, bool, error) {
	if w.sampleCount == 0 || !w.hasUncommittedData {
		return w.lastCommitEnd, false, nil
	}
	end, err := w.resolveCommitEnd(ctx)
	if err != nil {
		return 0, false, err
	}
	// because the range is exclusive, we need to add 1 nanosecond to the end
	end.Lower++
	// No reader sees a channel of the commit before every channel has committed, so no
	// read finds an index sample without its data.
	defer w.group.Publish()
	skipped := false
	commit := func(chW *unaryWriterState) bool {
		keySkipped, cErr := chW.CommitWithEndUnless(ctx, end.Lower, &w.group, skip)
		skipped = skipped || keySkipped
		// Every channel shares the same end, so their errors repeat. Keep the first.
		if err == nil {
			err = cErr
		}
		return keySkipped
	}
	// The index persists first: after a restart, a reader resolves a data domain
	// through the index.
	indexSkipped := w.writingToIdx && commit(w.internal[w.idx.ch.Key])
	// A skipped index leaves its data channels with no stamps of this writer, so the
	// group waits.
	for key, chW := range w.internal {
		if key != w.idx.ch.Key && !indexSkipped {
			commit(chW)
		}
	}
	if err != nil {
		return 0, false, err
	}
	// A skipped commit also ends the uncommitted data: the writer that commits the
	// skipped channels may advance their domains past this writer's high water mark.
	w.hasUncommittedData = false
	if skipped {
		return 0, true, nil
	}
	w.lastCommitEnd = end.Lower
	return end.Lower, false, nil
}

func (w *idxWriter) endDomain(ctx context.Context) error {
	if _, _, err := w.Commit(ctx, nil); err != nil {
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

func (w *idxWriter) Close() error {
	var err error
	for _, uWriter := range w.internal {
		err = errors.Join(err, uWriter.Close())
	}
	return err
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
