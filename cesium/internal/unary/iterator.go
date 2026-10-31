// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary

import (
	"bufio"
	"context"
	"io"
	"slices"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type IteratorConfig struct {
	Bounds telem.TimeRange
	// AutoChunkSize sets the maximum size of a chunk that will be returned by the
	// iterator when using AutoSpan in calls ot Next or Prev.
	AutoChunkSize int64
	// Reduction reduces the samples the iterator reads. A stride keeps the samples
	// whose alignment is a multiple of its factor. A limit reduces numeric channels to
	// about PointLimit points over groups anchored at sample 0 of each index domain.
	// Under a limit, AutoChunkSize counts points, auto spans end on group boundaries,
	// and a fixed span clips groups at the view edges.
	Reduction telem.Reduction
}

func (i IteratorConfig) domainIteratorConfig() domain.IteratorConfig {
	return domain.IteratorConfig{Bounds: i.Bounds}
}

// Override implements config.Config.
func (i IteratorConfig) Override(other IteratorConfig) IteratorConfig {
	i.Bounds = override.Zero(i.Bounds, other.Bounds)
	i.AutoChunkSize = override.Numeric(i.AutoChunkSize, other.AutoChunkSize)
	i.Reduction.Variant = override.Nil(i.Reduction.Variant, other.Reduction.Variant)
	return i
}

// Validate implements config.Config.
func (i IteratorConfig) Validate() error {
	v := validate.New("unary.iterator_config")
	v.Positive("auto_chunk_size", i.AutoChunkSize)
	v.Exec(i.Reduction.Validate)
	return v.Error()
}

var (
	_                     config.Config[IteratorConfig] = IteratorConfig{}
	defaultIteratorConfig                               = IteratorConfig{
		AutoChunkSize: 1e5,
	}
)

func IterRange(tr telem.TimeRange) IteratorConfig {
	return IteratorConfig{Bounds: domain.IterRange(tr).Bounds, AutoChunkSize: 0}
}

var ErrIteratorClosed = resource.NewClosedError("unary.iterator")

type Iterator struct {
	alamos.Instrumentation
	err      error
	internal *domain.Iterator
	idx      *index.Domain
	resolver *offsetResolver
	Channel  channel.Channel
	frame    channel.Frame
	// segments is scratch space for the index segments of a read.
	segments []index.Segment
	IteratorConfig
	view   telem.TimeRange
	bounds telem.TimeRange
	closed bool
	// stride keeps the samples whose alignment is a multiple of stride. Values below 2
	// keep every sample.
	stride uint32
	// aggregation and pointLimit are the limit reduction of the read. A zero pointLimit
	// means the read has no limit.
	aggregation telem.Aggregation
	pointLimit  uint32
	// groupSize is the number of samples per aggregation group over bounds. Zero means
	// it has not been computed since bounds last changed.
	groupSize uint32
	// head and tail summarize the cut groups at the start and end of the frame, so a
	// read of the rest of a group can complete it. Each is nil when its group is whole.
	// A forward read keeps only tail current, and a backward read only head.
	head, tail *telem.Group
}

func (db *DB) OpenIterator(cfgs ...IteratorConfig) (*Iterator, error) {
	if db.closed.Load() {
		return nil, db.wrapError(ErrDBClosed)
	}
	cfg, err := config.New(defaultIteratorConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	iter := db.domain.OpenIterator(cfg.domainIteratorConfig())
	i := &Iterator{
		Instrumentation: db.cfg.Instrumentation,
		idx:             db.index(),
		Channel:         db.cfg.Channel,
		resolver:        db.resolver,
		internal:        iter,
		IteratorConfig:  cfg,
	}
	switch r := cfg.Reduction.Variant.(type) {
	case telem.StrideReduction:
		i.stride = r.Factor
	case telem.LimitReduction:
		i.aggregation, i.pointLimit = r.Aggregation, r.PointLimit
	}
	i.SetBounds(cfg.Bounds)
	return i, nil
}

const AutoSpan telem.TimeSpan = -1

// SetBounds sets the iterator's bounds. The iterator is invalidated, and will not be
// valid until a seeking call is made.
func (i *Iterator) SetBounds(tr telem.TimeRange) {
	i.bounds = tr
	i.groupSize = 0
	i.internal.SetBounds(tr)
	i.seekReset(i.bounds.End)
}

func (i *Iterator) Bounds() telem.TimeRange { return i.bounds }

// MoveTo empties the frame and moves the view to the empty range at ts without reading,
// so that Continue reads forward from ts. Unlike a seek, MoveTo keeps the error that
// stopped the iterator.
func (i *Iterator) MoveTo(ctx context.Context, ts telem.TimeStamp) {
	i.reset(ts.SpanRange(0))
	// A domain at or before ts keeps the iterator on a domain when none follows ts, so
	// a later Prev still finds the earlier ones.
	if !i.internal.SeekLE(ctx, ts) {
		i.internal.SeekGE(ctx, ts)
	}
}

// Continue replaces the frame with the data between the end of the view and end, and
// extends the view to end. It splits one read into consecutive chunks.
func (i *Iterator) Continue(ctx context.Context, end telem.TimeStamp) bool {
	start := i.view.Start
	ok := i.Next(ctx, i.view.End.Span(end))
	i.view.Start = start
	return ok
}

// EndAfter returns the end of the time range that starts at the end of the view and
// holds the next n samples of the channel. See index.Domain.EndAfter.
func (i *Iterator) EndAfter(ctx context.Context, n int64) (telem.TimeStamp, error) {
	return i.idx.EndAfter(ctx, i.view.End, n)
}

func (i *Iterator) Value() channel.Frame { return i.frame }

func (i *Iterator) View() telem.TimeRange { return i.view }

func (i *Iterator) SeekFirst(ctx context.Context) bool {
	if i.closed {
		i.err = ErrIteratorClosed
		return false
	}
	ok := i.internal.SeekFirst(ctx)
	i.seekReset(i.internal.TimeRange().BoundBy(i.bounds).Start)
	return ok
}

// SeekLast moves the iterator to the end of the last domain in its bounds.
func (i *Iterator) SeekLast(ctx context.Context) bool {
	if i.closed {
		i.err = ErrIteratorClosed
		return false
	}
	ok := i.internal.SeekLast(ctx)
	i.seekReset(i.internal.TimeRange().BoundBy(i.bounds).End)
	return ok
}

func (i *Iterator) SeekLE(ctx context.Context, ts telem.TimeStamp) bool {
	if i.closed {
		i.err = ErrIteratorClosed
		return false
	}

	ok := i.internal.SeekLE(ctx, ts)

	if i.internal.TimeRange().OverlapsWith(ts.SpanRange(0)) {
		// If the provided ts is in the seeked domain, set the view to be ts.
		i.seekReset(ts)
	} else {
		// Otherwise, set the view to the end of the seeked domain or bounds, whichever
		// one is earlier.
		i.seekReset(i.internal.TimeRange().BoundBy(i.bounds).End)
	}
	return ok
}

func (i *Iterator) SeekGE(ctx context.Context, ts telem.TimeStamp) bool {
	if i.closed {
		i.err = ErrIteratorClosed
		return false
	}

	ok := i.internal.SeekGE(ctx, ts)

	if i.internal.TimeRange().OverlapsWith(ts.SpanRange(0)) {
		// If the provided ts is in the seeked domain, set the view to be ts.
		i.seekReset(ts)
	} else {
		// Otherwise, set the view to the start of the seeked domain or bounds,
		// whichever one is later.
		i.seekReset(i.internal.TimeRange().BoundBy(i.bounds).Start)
	}
	return ok
}

// Next moves the iterator forward by span. More specifically, if the current view is
// [start, end), after Next(span) is called, the view becomes [end, end + span). After
// the view changes, the internal iterator moves forward and accumulates data until the
// entire view is contained in the iterator's frame.
func (i *Iterator) Next(ctx context.Context, span telem.TimeSpan) (ok bool) {
	if i.closed {
		i.err = ErrIteratorClosed
		return false
	}
	ctx, spn := i.T.Bench(ctx, "Next")
	defer func() {
		ok = i.Valid()
		spn.End()
	}()
	if i.atEnd() {
		i.reset(i.bounds.End.SpanRange(0))
		return ok
	}
	if err := i.resolveGroupSize(ctx); err != nil {
		i.err = err
		return false
	}

	if span == AutoSpan {
		return i.autoNext(ctx)
	}

	i.reset(i.view.End.SpanRange(span).BoundBy(i.bounds))

	if i.view.Span().IsZero() || i.view.End.BeforeEq(i.internal.TimeRange().Start) {
		return ok
	}

	// A domain that ends after the view can hold samples the next span needs, so the
	// iterator stays on it.
	i.accumulate(ctx)
	for i.err == nil &&
		i.internal.TimeRange().End.Before(i.view.End) &&
		i.internal.Next() &&
		i.internal.TimeRange().Start.Before(i.view.End) {
		i.accumulate(ctx)
	}
	return ok
}

// autoNext reads chunks forward from the current view end until one holds data. A
// downsampled chunk can keep no samples, so an empty chunk does not end the read.
func (i *Iterator) autoNext(ctx context.Context) bool {
	for {
		if more := i.nextChunk(ctx); !more || i.partiallySatisfied() {
			return i.partiallySatisfied()
		}
	}
}

// nextChunk reads up to AutoChunkSize samples, or points under a limit, forward from
// the current view end. Each domain resolves the chunk to a pair of sample positions,
// and both the samples read and the time range reported come from that pair, so the two
// cannot disagree. nextChunk returns true if samples can follow the chunk.
func (i *Iterator) nextChunk(ctx context.Context) bool {
	// The chunk provisionally covers everything left in bounds. The sample that closes
	// it replaces the end once the read reaches it.
	i.reset(i.view.End.Range(i.bounds.End))
	var (
		nRemaining = i.AutoChunkSize
		end        = i.view.Start
		closed     bool
		more       bool
	)
	for {
		domainTR := i.internal.TimeRange()
		if domainTR.IsZero() || !domainTR.OverlapsWith(i.view) {
			if !i.internal.Next() {
				break
			}
			continue
		}
		startApprox, err := i.approximateStart(ctx)
		if err != nil {
			i.err = err
			return false
		}
		endApprox, err := i.approximateEnd(ctx)
		if err != nil {
			i.err = err
			return false
		}
		var (
			startSample = pickSampleOffset(startApprox)
			// limit is the sample after the last one this domain can give the chunk.
			limit     = pickSampleOffset(endApprox)
			endSample = min(startSample+nRemaining, limit)
		)
		if i.groupSize > 1 {
			if nRemaining > 0 {
				endSample, err = i.groupEnd(ctx, startSample, limit, nRemaining)
			} else {
				endSample, err = i.cutGroupEnd(ctx, startSample, limit)
			}
			if err != nil {
				i.err = err
				return false
			}
		}
		end = domainTR.End
		if closed = endSample < limit; closed {
			// The chunk fills inside this domain, so it ends at the first sample it
			// leaves behind.
			end, err = i.stampSample(ctx, domainTR.Start, endSample)
			if err != nil {
				i.err = err
				return false
			}
			i.view.End = end
		}
		used, err := i.read(ctx, startSample, endSample)
		if err != nil && !errors.Is(err, io.EOF) {
			i.err = err
			return false
		}
		nRemaining -= used
		// A full chunk reads on into the next domain only to complete a cut group.
		more = closed || nRemaining <= 0
		if closed || (more && i.tail == nil) || !i.internal.Next() {
			break
		}
	}
	i.view.End = min(end, i.bounds.End)
	return more && !i.atEnd()
}

// autoPrev mirrors autoNext, reading chunks backward from the current view start.
func (i *Iterator) autoPrev(ctx context.Context) bool {
	for {
		if more := i.prevChunk(ctx); !more || i.partiallySatisfied() {
			return i.partiallySatisfied()
		}
	}
}

// prevChunk reads up to AutoChunkSize samples, or points under a limit, backward from
// the current view start. It mirrors nextChunk: the chunk resolves to sample positions
// first, and its time range follows from them.
func (i *Iterator) prevChunk(ctx context.Context) bool {
	i.reset(i.bounds.Start.Range(i.view.Start))
	var (
		nRemaining = i.AutoChunkSize
		start      = i.view.End
		closed     bool
		more       bool
	)
	for {
		domainTR := i.internal.TimeRange()
		if domainTR.IsZero() || !domainTR.OverlapsWith(i.view) {
			if !i.internal.Prev() {
				break
			}
			continue
		}
		startApprox, err := i.approximateStart(ctx)
		if err != nil {
			i.err = err
			return false
		}
		endApprox, err := i.approximateEnd(ctx)
		if err != nil {
			i.err = err
			return false
		}
		var (
			// first is the earliest sample of this domain the chunk can reach.
			first       = pickSampleOffset(startApprox)
			endSample   = pickSampleOffset(endApprox)
			startSample = max(endSample-nRemaining, first)
		)
		if i.groupSize > 1 {
			if nRemaining > 0 {
				startSample, err = i.groupStart(ctx, first, endSample, nRemaining)
			} else {
				startSample, err = i.cutGroupStart(ctx, first, endSample)
			}
			if err != nil {
				i.err = err
				return false
			}
		}
		start = domainTR.Start
		if closed = startSample > first; closed {
			// The chunk fills inside this domain, so it starts at the earliest sample
			// it holds.
			start, err = i.stampSample(ctx, domainTR.Start, startSample)
			if err != nil {
				i.err = err
				return false
			}
			i.view.Start = start
		}
		used, err := i.read(ctx, startSample, endSample)
		if err != nil && !errors.Is(err, io.EOF) {
			i.err = err
			return false
		}
		nRemaining -= used
		more = closed || nRemaining <= 0
		if closed || (more && i.head == nil) || !i.internal.Prev() {
			break
		}
	}
	i.view.Start = max(start, i.bounds.Start)
	return more && !i.atStart()
}

// stampSample returns the timestamp of the sample at position offset within the domain
// starting at start.
func (i *Iterator) stampSample(
	ctx context.Context,
	start telem.TimeStamp,
	offset int64,
) (telem.TimeStamp, error) {
	approx, err := i.idx.Stamp(ctx, start, offset, index.AllowDiscontinuous)
	return approx.Upper, err
}

// Prev moves the iterator backward by span. More specifically, if the current view is
// [start, end), after Next(span) is called, the view becomes [start - span, start).
// After the view changes, the internal iterator moves backward and accumulates data
// until the entire view is contained in the iterator's frame.
func (i *Iterator) Prev(ctx context.Context, span telem.TimeSpan) (ok bool) {
	if i.closed {
		i.err = ErrIteratorClosed
		return false
	}
	ctx, spn := i.T.Bench(ctx, "Prev")
	defer func() { ok = i.Valid(); spn.End() }()

	if i.atStart() {
		i.reset(i.bounds.Start.SpanRange(0))
		return ok
	}
	if err := i.resolveGroupSize(ctx); err != nil {
		i.err = err
		return false
	}

	if span == AutoSpan {
		return i.autoPrev(ctx)
	}

	i.reset(i.view.Start.SpanRange(-1 * span).BoundBy(i.bounds))

	if i.view.Span().IsZero() || i.view.Start.AfterEq(i.internal.TimeRange().End) {
		return ok
	}

	// A domain that starts before the view can hold samples the next span needs, so the
	// iterator stays on it.
	i.accumulate(ctx)
	for i.err == nil &&
		i.internal.TimeRange().Start.After(i.view.Start) &&
		i.internal.Prev() &&
		i.internal.TimeRange().End.After(i.view.Start) {
		i.accumulate(ctx)
	}
	return ok
}

// Error returns the error that caused the iterator to stop moving. If the iterator is
// still moving, Error returns nil.
func (i *Iterator) Error() error {
	wrap := channel.NewErrorWrapper(i.Channel)
	return wrap(i.err)
}

// Valid checks if an iterator has accumulated no errors and has at least one series in
// its current frame.
func (i *Iterator) Valid() bool { return i.partiallySatisfied() && i.err == nil }

// Close closes the iterator and releases any resources it holds. It returns the error
// that stopped the iterator, unless a seek or SetBounds has cleared it since. Closing
// a closed iterator returns nil. As with all other iterator methods, Close is not safe
// to call concurrently with any other database method.
//
// After close is called, the iterator should no longer be used.
func (i *Iterator) Close() error {
	if i.closed {
		return nil
	}
	i.closed = true
	wrap := channel.NewErrorWrapper(i.Channel)
	return wrap(errors.Combine(i.err, i.internal.Close()))
}

// accumulate reads the samples of the current domain that lie in the view into the
// frame.
func (i *Iterator) accumulate(ctx context.Context) {
	domainTR := i.internal.TimeRange()
	if domainTR.IsZero() || !domainTR.OverlapsWith(i.view) {
		return
	}
	start, end, err := i.sliceDomain(ctx)
	if err != nil {
		i.err = err
		return
	}
	if _, err = i.read(ctx, start, end); err != nil && !errors.Is(err, io.EOF) {
		i.err = err
	}
}

// loadSegments loads the index segments of samples [start, end) of the current domain
// into i.segments.
func (i *Iterator) loadSegments(ctx context.Context, start, end int64) (err error) {
	if i.Channel.IsIndex {
		// An index channel timestamps itself, so the domain it reads is its segment.
		i.segments = append(i.segments[:0], index.Segment{
			TimeRange: i.internal.TimeRange(),
			Alignment: telem.NewAlignment(i.internal.Position(), uint32(start)),
			Len:       end - start,
		})
		return nil
	}
	i.segments, err = i.idx.AppendSegments(
		ctx,
		i.segments[:0],
		i.internal.TimeRange().Start,
		start,
		end,
	)
	return err
}

// read reads samples [start, end) of the current domain into the frame in time order.
// The samples become one series per index domain that holds their timestamps, so each
// series carries the alignment of its own index samples. read returns how much of
// AutoChunkSize the read used: points when the read is reduced, and source samples
// otherwise.
func (i *Iterator) read(
	ctx context.Context,
	start, end int64,
) (used int64, err error) {
	if start == end {
		return 0, nil
	}
	if err = i.loadSegments(ctx, start, end); err != nil {
		return 0, err
	}
	domainTR := i.internal.TimeRange()
	r, err := i.internal.OpenReader(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Combine(err, r.Close()) }()
	tr := domainTR.BoundBy(i.view)
	// The frame holds either earlier domains or later ones, so a read that does not
	// follow the frame goes in front of it, last segment first.
	if i.frame.Empty() || i.frame.SeriesAt(-1).TimeRange.End.BeforeEq(tr.Start) {
		for _, seg := range i.segments {
			res, err := i.readSegment(ctx, r, seg, start, tr)
			if err != nil {
				return 0, err
			}
			used += i.consumed(i.append(res), res.srcLen)
			start += seg.Len
		}
		return used, nil
	}
	for _, seg := range slices.Backward(i.segments) {
		end -= seg.Len
		res, err := i.readSegment(ctx, r, seg, end, tr)
		if err != nil {
			return 0, err
		}
		used += i.consumed(i.prepend(res), res.srcLen)
	}
	return used, nil
}

// segmentRead is the result of reading one index segment.
type segmentRead struct {
	series telem.Series
	// srcLen is the number of source samples read.
	srcLen int64
	// head and tail summarize the first and last groups of a reduced read when the
	// segment holds only part of them.
	head, tail *telem.Group
}

// append adds res to the end of the frame and returns the series it added. When the
// frame's last series and res hold parts of the same group, the last series takes the
// merged group and res drops its part.
func (i *Iterator) append(res segmentRead) telem.Series {
	s := res.series
	if s.Len() == 0 {
		return s
	}
	if i.tail != nil && res.head != nil && i.tail.Start() == res.head.Start() {
		last, rest, merged := i.frame.SeriesAt(-1).AbsorbNext(s, *i.tail, *res.head)
		i.frame.SetSeriesAt(-1, last)
		if s = rest; s.Len() == 0 && res.tail != nil {
			res.tail = &merged
		}
	}
	if s.Len() > 0 {
		if i.frame.Empty() {
			i.head = res.head
		}
		i.frame = i.frame.Append(i.Channel.Key, s)
	}
	i.tail = res.tail
	return s
}

// prepend mirrors append for a read that goes in front of the frame. The frame's first
// series takes the merged group.
func (i *Iterator) prepend(res segmentRead) telem.Series {
	s := res.series
	if s.Len() == 0 {
		return s
	}
	if i.head != nil && res.tail != nil && res.tail.Start() == i.head.Start() {
		first, rest, merged := i.frame.SeriesAt(0).AbsorbPrev(s, *res.tail, *i.head)
		i.frame.SetSeriesAt(0, first)
		if s = rest; s.Len() == 0 && res.head != nil {
			res.head = &merged
		}
	}
	if s.Len() > 0 {
		i.frame = i.frame.Prepend(i.Channel.Key, s)
	}
	i.head = res.head
	return s
}

// readSegment reads the samples of seg, the first of which is sample start of the
// current domain. tr is the time range of the whole read. A strided read keeps the
// samples whose alignment is a multiple of the stride, so every channel that shares an
// index keeps the same samples.
func (i *Iterator) readSegment(
	ctx context.Context,
	r *domain.Reader,
	seg index.Segment,
	start int64,
	tr telem.TimeRange,
) (segmentRead, error) {
	offset, err := i.resolver.byteOffset(ctx, i.internal, start)
	if err != nil {
		return segmentRead{}, err
	}
	endOffset, err := i.resolver.byteOffset(ctx, i.internal, start+seg.Len)
	if err != nil {
		return segmentRead{}, err
	}
	series := telem.Series{
		DataType:  i.Channel.DataType,
		TimeRange: tr.BoundBy(seg.TimeRange),
		Alignment: seg.Alignment,
	}
	size := endOffset - offset
	if i.groupSize > 1 {
		if i.aggregation == telem.AggregationDecimate {
			return i.readDecimated(r, series, offset, size)
		}
		return i.readReduced(r, series, offset, size)
	}
	if factor := i.stride; factor > 1 {
		skip := (factor - seg.Alignment.SampleIndex()%factor) % factor
		series.Alignment = seg.Alignment.AddSamples(skip)
		series.AlignmentMultiple = factor
		var srcLen int64
		if series.Data, srcLen, err = i.readStrided(
			r,
			offset,
			size,
			int64(skip),
		); err != nil {
			return segmentRead{}, err
		}
		return segmentRead{series: series, srcLen: srcLen}, nil
	}
	series.Data = make([]byte, size)
	n, err := r.ReadAt(series.Data, int64(offset))
	if err != nil && !errors.Is(err, io.EOF) {
		return segmentRead{}, err
	}
	series.Data = series.Data[:n]
	return segmentRead{series: series, srcLen: series.Len()}, nil
}

// strideBufferSize bounds the scratch buffer a strided read holds, so the buffer never
// scales with the size of the slice being read.
const strideBufferSize = 64 * telem.Kilobyte

// reduceBufferSize bounds the scratch buffer a reduced read holds, so the buffer never
// scales with the size of a group.
const reduceBufferSize = telem.Megabyte

// resolveGroupSize computes the aggregation group size over the iterator's bounds once
// per bounds. The count comes from the index, so every channel on it agrees.
func (i *Iterator) resolveGroupSize(ctx context.Context) error {
	if i.groupSize != 0 {
		return nil
	}
	if i.pointLimit == 0 || !telem.Reducible(i.Channel.DataType) {
		i.groupSize = 1
		return nil
	}
	approx, err := i.idx.Distance(ctx, i.bounds, index.AllowDiscontinuous)
	if errors.Is(err, index.ErrDiscontinuous) {
		i.groupSize = 1
		return nil
	}
	if err != nil {
		return err
	}
	i.groupSize = telem.GroupSize(
		pickSampleOffset(approx),
		i.pointLimit,
		i.aggregation,
	)
	return nil
}

// consumed returns how much of AutoChunkSize a read used: points when the read is
// reduced, and source samples otherwise.
func (i *Iterator) consumed(series telem.Series, srcLen int64) int64 {
	if i.groupSize > 1 {
		return series.Len()
	}
	return srcLen
}

// groupEnd returns the end of a forward chunk of samples [start, limit) of the current
// domain that holds whole groups giving up to points points. The chunk holds at least
// the group of its first sample.
func (i *Iterator) groupEnd(
	ctx context.Context,
	start, limit, points int64,
) (int64, error) {
	if err := i.loadSegments(ctx, start, limit); err != nil {
		return 0, err
	}
	for _, seg := range i.segments {
		if span := i.forwardGroupSpan(seg.Alignment, points); span < seg.Len {
			return start + span, nil
		}
		start += seg.Len
		if points -= i.points(seg); points <= 0 {
			return start, nil
		}
	}
	return limit, nil
}

// groupStart mirrors groupEnd for a backward chunk of samples [first, end).
func (i *Iterator) groupStart(
	ctx context.Context,
	first, end, points int64,
) (int64, error) {
	if err := i.loadSegments(ctx, first, end); err != nil {
		return 0, err
	}
	for _, seg := range slices.Backward(i.segments) {
		upper := seg.Alignment.AddSamples(uint32(seg.Len))
		if span := i.backwardGroupSpan(upper, points); span < seg.Len {
			return end - span, nil
		}
		end -= seg.Len
		if points -= i.points(seg); points <= 0 {
			return end, nil
		}
	}
	return first, nil
}

// cutGroupEnd returns the end of the samples of the current domain from start that
// complete the frame's cut last group. It returns start when the domain does not
// continue the group.
func (i *Iterator) cutGroupEnd(ctx context.Context, start, limit int64) (int64, error) {
	if err := i.loadSegments(ctx, start, limit); err != nil || len(i.segments) == 0 {
		return start, err
	}
	seg := i.segments[0]
	if telem.GroupStart(seg.Alignment, i.groupSize) != i.tail.Start() {
		return start, nil
	}
	rest := int64(i.groupSize - seg.Alignment.SampleIndex()%i.groupSize)
	return start + min(rest, seg.Len), nil
}

// cutGroupStart mirrors cutGroupEnd for a backward chunk that ends at end, completing
// the frame's cut first group.
func (i *Iterator) cutGroupStart(ctx context.Context, first, end int64) (int64, error) {
	if err := i.loadSegments(ctx, first, end); err != nil || len(i.segments) == 0 {
		return end, err
	}
	seg := i.segments[len(i.segments)-1]
	last := seg.Alignment.AddSamples(uint32(seg.Len - 1))
	if telem.GroupStart(last, i.groupSize) != i.head.Start() {
		return end, nil
	}
	held := int64(last.SampleIndex()-i.head.Start().SampleIndex()) + 1
	return end - min(held, seg.Len), nil
}

// points returns the number of points a reduced read of seg yields.
func (i *Iterator) points(seg index.Segment) int64 {
	var (
		size  = int64(i.groupSize)
		first = int64(seg.Alignment.SampleIndex()) % size
	)
	return (first + seg.Len + size - 1) / size * i.aggregation.PointsPerGroup()
}

// forwardGroupSpan returns the number of samples from start that covers enough whole
// groups to give up to points points. The span ends on a group boundary and covers at
// least the group that holds start.
func (i *Iterator) forwardGroupSpan(start telem.Alignment, points int64) int64 {
	var (
		size   = int64(i.groupSize)
		groups = (points + i.aggregation.PointsPerGroup() - 1) /
			i.aggregation.PointsPerGroup()
		pos = int64(start.SampleIndex())
		end = pos + groups*size
	)
	end -= end % size
	if end <= pos {
		end = pos - pos%size + size
	}
	return end - pos
}

// backwardGroupSpan mirrors forwardGroupSpan for a chunk that ends before end.
func (i *Iterator) backwardGroupSpan(end telem.Alignment, points int64) int64 {
	var (
		size   = int64(i.groupSize)
		groups = (points + i.aggregation.PointsPerGroup() - 1) /
			i.aggregation.PointsPerGroup()
		pos   = int64(end.SampleIndex())
		start = max(pos-groups*size, 0)
	)
	if rem := start % size; rem != 0 {
		start += size - rem
	}
	if start >= pos {
		start = max(pos-1, 0)
		start -= start % size
	}
	return pos - start
}

// readReduced reads the slice [offset, offset+size) of r in chunks and reduces each
// chunk into series. A chunk holds whole groups when a group fits in the read buffer,
// so only a larger group spans two chunks.
func (i *Iterator) readReduced(
	r io.ReaderAt,
	series telem.Series,
	offset telem.Size,
	size telem.Size,
) (segmentRead, error) {
	var (
		density   = i.Channel.DataType.Density()
		groupSize = int64(i.groupSize)
		src       = density.SampleCount(size)
		chunk     = density.SampleCount(reduceBufferSize)
		// next is the sample count of the next chunk.
		next = chunk
		res  = segmentRead{series: series}
	)
	if groupSize <= chunk {
		chunk -= chunk % groupSize
		// The first chunk stops at the end of the group that holds the first sample.
		next = chunk - int64(series.Alignment.SampleIndex())%groupSize
	}
	buf := make([]byte, density.Size(min(chunk, src)))
	for res.srcLen < src {
		n := min(next, src-res.srcLen)
		b := buf[:density.Size(n)]
		count, err := r.ReadAt(b, int64(offset+density.Size(res.srcLen)))
		if err != nil && !errors.Is(err, io.EOF) {
			return segmentRead{}, err
		}
		avail := density.SampleCount(telem.Size(count))
		if avail == 0 {
			break
		}
		raw := series
		raw.Data = b[:density.Size(avail)]
		raw.Alignment = series.Alignment.AddSamples(uint32(res.srcLen))
		res.extend(raw.ReduceCut(i.aggregation, i.groupSize))
		res.srcLen += avail
		if avail < n {
			break
		}
		next = chunk
	}
	return res, nil
}

// extend appends part, a chunk reduced after the chunks in res, to the series of res.
// head and tail summarize the groups part holds only part of.
func (res *segmentRead) extend(part telem.Series, head, tail *telem.Group) {
	if res.series.Len() == 0 {
		res.series, res.head, res.tail = part, head, tail
		return
	}
	if res.tail != nil && head != nil && res.tail.Start() == head.Start() {
		var merged telem.Group
		res.series, part, merged = res.series.AbsorbNext(part, *res.tail, *head)
		if res.head != nil && res.head.Start() == merged.Start() {
			res.head = &merged
		}
		if part.Len() == 0 && tail != nil {
			tail = &merged
		}
	}
	res.series.Data = append(res.series.Data, part.Data...)
	res.tail = tail
}

// readDecimated keeps the first present sample of each group in the slice [offset,
// offset+size) of r, and reads only the samples it keeps. It returns the series and the
// number of source samples the slice held.
func (i *Iterator) readDecimated(
	r io.ReaderAt,
	series telem.Series,
	offset telem.Size,
	size telem.Size,
) (segmentRead, error) {
	var (
		groupSize = int64(i.groupSize)
		first     = series.Alignment
		lead      = int64(first.SampleIndex()) % groupSize
		// skip is the position in the slice of the first group start.
		skip = (groupSize - lead) % groupSize
		head []byte
		res  segmentRead
	)
	series.Alignment = telem.GroupStart(first, i.groupSize)
	series.AlignmentMultiple = i.groupSize
	if skip > 0 {
		// A partial first group keeps its first sample, which is not a group start.
		head = make([]byte, i.Channel.DataType.Density())
		n, err := r.ReadAt(head, int64(offset))
		if err != nil && !errors.Is(err, io.EOF) {
			return segmentRead{}, err
		}
		if n < len(head) {
			return segmentRead{series: series}, nil
		}
		_, res.head, _ = telem.Series{
			DataType:  series.DataType,
			Data:      head,
			Alignment: first,
		}.ReduceCut(i.aggregation, i.groupSize)
	}
	data, srcLen, err := i.readStridedFixed(r, offset, size, skip, groupSize)
	if err != nil {
		return segmentRead{}, err
	}
	series.Data = append(head, data...)
	if (lead+srcLen)%groupSize > 0 && series.Len() > 0 {
		// The last sample kept is the first sample of the cut last group.
		at := max(
			telem.GroupStart(first.AddSamples(uint32(srcLen-1)), i.groupSize),
			first,
		)
		density := int(series.DataType.Density())
		_, _, res.tail = telem.Series{
			DataType:  series.DataType,
			Data:      series.Data[len(series.Data)-density:],
			Alignment: at,
		}.ReduceCut(i.aggregation, i.groupSize)
	}
	res.series, res.srcLen = series, srcLen
	return res, nil
}

var errPrefixOverrunsSlice = errors.New("length prefix exceeds domain slice")

// readStrided reads every stride-th sample of the slice [offset, offset+size) in r,
// starting skip samples into the slice, and packs the kept samples into a buffer sized
// to them alone. It returns the packed data and the number of source samples the slice
// held. A slice shorter than size yields the samples that were available.
func (i *Iterator) readStrided(
	r io.ReaderAt,
	offset telem.Size,
	size telem.Size,
	skip int64,
) ([]byte, int64, error) {
	if i.Channel.DataType.IsVariable() {
		return i.readStridedVariable(r, offset, size, skip)
	}
	return i.readStridedFixed(r, offset, size, skip, int64(i.stride))
}

// readStridedFixed is readStrided for a fixed-density data type, keeping every
// factor-th sample.
func (i *Iterator) readStridedFixed(
	r io.ReaderAt,
	offset telem.Size,
	size telem.Size,
	skip int64,
	factor int64,
) ([]byte, int64, error) {
	var (
		density    = i.Channel.DataType.Density()
		srcSamples = density.SampleCount(size)
		strided    = max(srcSamples-skip, 0)
		kept       = strided/factor + min(strided%factor, 1)
		stride     = int64(density.Size(factor))
		// batch is how many kept samples a single ReadAt covers. A stride wider than
		// the buffer drops it to one, so the read skips the discarded samples instead
		// of pulling them through the buffer.
		batch = max(int64(strideBufferSize)/stride, 1)
		buf   = make([]byte, stride*(batch-1)+int64(density))
		out   = make([]byte, 0, density.Size(kept))
	)
	for read := int64(0); read < kept; {
		n := min(batch, kept-read)
		b := buf[:stride*(n-1)+int64(density)]
		count, err := r.ReadAt(b, int64(offset+density.Size(skip+read*factor)))
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, 0, err
		}
		avail := density.SampleCount(telem.Size(count))
		for j := int64(0); j < n && j*factor < avail; j++ {
			out = append(out, b[j*stride:j*stride+int64(density)]...)
		}
		if count < len(b) {
			return out, skip + read*factor + avail, nil
		}
		read += n
	}
	return out, srcSamples, nil
}

func (i *Iterator) readStridedVariable(
	r io.ReaderAt,
	offset telem.Size,
	size telem.Size,
	skip int64,
) ([]byte, int64, error) {
	var (
		factor = int64(i.stride)
		br     = bufio.NewReaderSize(
			io.NewSectionReader(r, int64(offset), int64(size)),
			int(min(strideBufferSize, size)),
		)
		lenBuf = make([]byte, 4)
		out    []byte
		src    int64
	)
	for pos := int64(0); pos+4 <= int64(size); src++ {
		start := pos
		if _, err := io.ReadFull(br, lenBuf); err != nil {
			if errors.IsAny(err, io.EOF, io.ErrUnexpectedEOF) {
				i.logShortStride(offset, start, err)
				return out, src, nil
			}
			return nil, 0, err
		}
		length := int64(telem.ByteOrder.Uint32(lenBuf))
		// A length prefix is stored data. Without this bound a corrupt one would drive
		// an allocation of up to 4GiB before the short read caught it.
		if pos+4+length > int64(size) {
			i.logShortStride(offset, start, errPrefixOverrunsSlice)
			return out, src, nil
		}
		pos += 4 + length
		if src < skip || (src-skip)%factor != 0 {
			if _, err := br.Discard(int(length)); err != nil {
				if errors.Is(err, io.EOF) {
					i.logShortStride(offset, start, err)
					return out, src, nil
				}
				return nil, 0, err
			}
			continue
		}
		out = append(out, lenBuf...)
		payload := len(out)
		out = append(out, make([]byte, length)...)
		if _, err := io.ReadFull(br, out[payload:]); err != nil {
			if errors.IsAny(err, io.EOF, io.ErrUnexpectedEOF) {
				i.logShortStride(offset, start, err)
				return out[:payload-4], src, nil
			}
			return nil, 0, err
		}
	}
	return out, src, nil
}

// logShortStride reports a strided read that stopped before the end of its slice,
// because the domain file ended early or because a length prefix claimed more bytes
// than the slice holds. The offset cache scans the same prefixes and usually reports
// the domain first, but its table survives a truncation that leaves End untouched.
func (i *Iterator) logShortStride(offset telem.Size, pos int64, cause error) {
	i.L.Error(
		"strided read stopped short of the domain slice",
		zap.Stringer("range", i.internal.TimeRange()),
		zap.Int64("domain_size", int64(i.internal.Size())),
		zap.Int64("stopped_at", int64(offset)+pos),
		zap.Error(cause),
	)
}

// sliceDomain returns the positions of the first sample of the current domain in the
// view and of the sample after its last one.
func (i *Iterator) sliceDomain(ctx context.Context) (start, end int64, err error) {
	startApprox, err := i.approximateStart(ctx)
	if err != nil {
		return 0, 0, err
	}
	endApprox, err := i.approximateEnd(ctx)
	if err != nil {
		return 0, 0, err
	}
	return pickSampleOffset(startApprox), pickSampleOffset(endApprox), nil
}

// pickSampleOffset selects the appropriate sample index from an approximation
// based on which bounds are exact. Matches the case analysis used by both the
// iterator and the delete code path.
//
// See unary/delete.go's calculateStartOffset / calculateEndOffset for the full
// case tree.
func pickSampleOffset(approx index.DistanceApproximation) int64 {
	if approx.Exact() || approx.StartExact {
		return approx.Upper
	}
	if approx.EndExact {
		return approx.Lower
	}
	// Distance widens its bounds by one sample for each end it could not place, so with
	// neither end exact the count sits one below the upper bound.
	return approx.Upper - 1
}

// approximateStart approximates the number of samples between the start of the current
// domain and the start of the current iterator view. If the start of the current view
// is before the start of the range, the returned value will be zero.
func (i *Iterator) approximateStart(
	ctx context.Context,
) (index.DistanceApproximation, error) {
	target := i.internal.TimeRange().Start.SpanRange(0)
	if i.internal.TimeRange().Start.Before(i.view.Start) {
		target.End = i.view.Start
	}
	return i.idx.Distance(ctx, target, index.MustBeContinuous)
}

// approximateEnd approximates the number of samples between the start of the current
// range and the end of the current iterator view. If the end of the current view is
// after the end of the range, the returned value will be the number of samples in the
// range.
func (i *Iterator) approximateEnd(
	ctx context.Context,
) (endApprox index.DistanceApproximation, err error) {
	total, err := i.resolver.domainSampleCount(ctx, i.internal)
	if err != nil {
		return index.DistanceApproximation{}, err
	}
	endApprox.Approximation = index.Exactly(total)
	if i.internal.TimeRange().End.After(i.view.End) {
		target := i.internal.TimeRange().Start.Range(i.view.End)
		endApprox, err = i.idx.Distance(ctx, target, index.MustBeContinuous)
	}
	return endApprox, err
}

func (i *Iterator) partiallySatisfied() bool { return i.frame.HasData() }

func (i *Iterator) reset(nextView telem.TimeRange) {
	i.frame = channel.Frame{}
	i.head, i.tail = nil, nil
	i.view = nextView
}

func (i *Iterator) seekReset(ts telem.TimeStamp) {
	i.reset(ts.SpanRange(0))
	i.err = nil
}

func (i *Iterator) atStart() bool { return i.view.Start == i.bounds.Start }

func (i *Iterator) atEnd() bool { return i.view.End == i.bounds.End }
