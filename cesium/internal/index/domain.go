// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
	"go.uber.org/zap"
)

// Domain is an implementation of Index backed by a domain-based database that stores
// the underlying timestamp values.
type Domain struct {
	alamos.Instrumentation
	// DB is the database to query for timestamp values.
	DB *domain.DB
	// Channel is the channel definition for the index.
	Channel channel.Channel
}

// sampleDensity is the density of telem.TimestampT.
const sampleDensity = telem.Bit64

func sampleCount(size telem.Size) int64 {
	return sampleDensity.SampleCount(size)
}

func byteSize(sampleCount int64) telem.Size {
	return sampleDensity.Size(sampleCount)
}

// Distance calculates an approximate distance (arithmetic difference in offset)
// between the start and end timestamps of the given time range. If continuous is
// true, the index will return an error if the underlying telemetry has
// discontinuities across the time range.
//
// The distance is approximated using a lower and upper bound. The underlying time
// series can be viewed as a contiguous slice of timestamps, where each timestamp
// exists at a specific index (i.e. slice[x]). The lower bound of the distance is
// the index of the timestamp less than or equal to the end timestamp and
// the index of the timestamp greater than or equal to the start timestamp. The upper
// bound is calculated using the opposite approach (i.e. finding the index of the
// timestamp greater than or equal to the end timestamp and the index of the
// timestamp less than or equal to the start timestamp). Naturally, a time range
// whose start timestamp and end timestamps are both known will have an equal lower
// and upper bound.
func (i *Domain) Distance(
	ctx context.Context,
	tr telem.TimeRange,
	continuous continuousPolicy,
) (approx DistanceApproximation, err error) {
	ctx, span := i.T.Bench(ctx, "distance")
	defer func() {
		if err != nil {
			approx = DistanceApproximation{}
		}
		err = span.EndWith(err, ErrDiscontinuous)
	}()

	iter := i.DB.OpenIterator(domain.IteratorConfig{Bounds: tr})
	defer func() { err = errors.Combine(err, iter.Close()) }()

	effectiveDomainTR, ok, err := i.seekDistanceStart(ctx, iter, tr, continuous)
	if err != nil || !ok {
		return approx, err
	}

	// Open a new reader on the domain at the start of the range.
	r, err := iter.OpenReader(ctx)
	if err != nil {
		return approx, err
	}
	defer func() { err = errors.Combine(err, r.Close()) }()

	startApprox, err := i.search(tr.Start, r)
	if err != nil {
		return approx, err
	}
	approx.StartExact = startApprox.Exact()

	if iter.TimeRange().ContainsStamp(tr.End) || tr.End == iter.TimeRange().End {
		// If the current domain contains the end of the time range, then everything
		// is continuous and within the current domain.
		approx.Approximation, approx.EndExact, err = i.searchEnd(
			tr.End,
			r,
			Between(-startApprox.Upper, -startApprox.Lower),
		)
		return approx, err
	}

	startToDomain, found, err := seekEnd(
		iter,
		tr,
		startApprox,
		sampleCount(r.Size()),
		effectiveDomainTR,
		continuous,
	)
	if err != nil || !found {
		approx.Approximation = startToDomain
		return approx, err
	}
	if r, err = reopenReader(ctx, iter, r); err != nil {
		return approx, err
	}
	approx.Approximation, approx.EndExact, err = i.searchEnd(tr.End, r, startToDomain)
	return approx, err
}

// searchEnd searches r for end and returns the distance to it from a start that lies
// startToDomain samples before the start of r's domain. It also returns whether end is
// in the index.
func (i *Domain) searchEnd(
	end telem.TimeStamp,
	r *domain.Reader,
	startToDomain Approximation[int64],
) (Approximation[int64], bool, error) {
	endApprox, err := i.search(end, r)
	if err != nil {
		return Approximation[int64]{}, false, err
	}
	return Between(
		startToDomain.Lower+endApprox.Lower,
		startToDomain.Upper+endApprox.Upper,
	), endApprox.Exact(), nil
}

// seekDistanceStart moves iter to the first domain in tr and returns the effective
// domain that holds it. It returns false if tr is empty or iter cannot move back to
// that domain.
func (i *Domain) seekDistanceStart(
	ctx context.Context,
	iter *domain.Iterator,
	tr telem.TimeRange,
	continuous continuousPolicy,
) (telem.TimeRange, bool, error) {
	if !iter.SeekFirst(ctx) {
		// If the domain with the given time range doesn't exist in the database, then
		// no data exists for that time range, so it's impossible to approximate a
		// distance.
		return telem.TimeRange{}, false, newDiscontinuousTRError(tr)
	}

	effectiveDomainTR, _ := resolveForwardEffectiveDomainTR(iter)
	if !iter.SeekFirst(ctx) {
		// Reset the iterator position after using it to determine effective bound.
		// A result of false in this case should be impossible, as we already validated
		// the iterator by calling SeekFirst in a previous call.
		i.L.DPanic("iterator seekFirst failed in stamp")
		return effectiveDomainTR, false, nil
	}

	// If the time range is not contained within the effective domain, then it's
	// discontinuous, and we return early if the user doesn't want discontinuous
	// results.
	if !effectiveDomainTR.ContainsRange(tr) && continuous {
		return telem.TimeRange{}, false, newDiscontinuousTRError(tr)
	}
	// If the time range is zero, then the distance is zero.
	return effectiveDomainTR, !tr.Span().IsZero(), nil
}

// seekEnd moves iter forward onto the domain that holds tr.End, where startApprox is
// the position of tr.Start in the current domain of domainLen samples. It returns the
// distance from tr.Start to the start of the domain it stops on, and false if no
// domain holds tr.End.
func seekEnd(
	iter *domain.Iterator,
	tr telem.TimeRange,
	startApprox Approximation[int64],
	domainLen int64,
	effectiveDomainTR telem.TimeRange,
	continuous continuousPolicy,
) (Approximation[int64], bool, error) {
	// Unless the effective domain contains the end of the time range, the distance is
	// discontinuous.
	if continuous &&
		!effectiveDomainTR.ContainsStamp(tr.End) &&
		effectiveDomainTR.End != tr.End {
		return Approximation[int64]{}, false, newDiscontinuousTRError(tr)
	}
	traversed := Between(domainLen-startApprox.Upper, domainLen-startApprox.Lower)
	for {
		if !iter.Next() ||
			(continuous && !effectiveDomainTR.ContainsRange(iter.TimeRange())) {
			if continuous {
				return Approximation[int64]{}, false, newDiscontinuousTRError(tr)
			}
			return traversed, false, nil
		}
		if iter.TimeRange().ContainsStamp(tr.End) {
			return traversed, true, nil
		}
		size := sampleCount(iter.Size())
		traversed.Lower += size
		traversed.Upper += size
	}
}

// reopenReader closes r and opens a reader on the domain at iter. It returns r when
// closing it fails.
func reopenReader(
	ctx context.Context,
	iter *domain.Iterator,
	r *domain.Reader,
) (*domain.Reader, error) {
	if err := r.Close(); err != nil {
		return r, err
	}
	return iter.OpenReader(ctx)
}

// Segment is a run of consecutive samples that lie in one domain of an index.
type Segment struct {
	// TimeRange is the time range of the domain that holds the segment.
	TimeRange telem.TimeRange
	// Alignment is the alignment of the segment's first sample.
	Alignment telem.Alignment
	// Len is the number of samples in the segment.
	Len int64
}

// AppendSegments splits samples [start, end) of a run of consecutive index samples into
// one segment per domain that holds them, and appends the segments to dst in time
// order. The first sample of the run is the first sample at or after ts. It returns
// ErrDiscontinuous if no domain holds a sample at or after ts, or if the run extends
// past the effective domain that holds its first sample.
func (i *Domain) AppendSegments(
	ctx context.Context,
	dst []Segment,
	ts telem.TimeStamp,
	start, end int64,
) (segments []Segment, err error) {
	if start >= end {
		return dst, nil
	}
	defer func() {
		if err != nil {
			segments = nil
		}
	}()
	iter := i.DB.OpenIterator(domain.IterRange(ts.SpanRange(telem.TimeSpanMax)))
	defer func() { err = errors.Combine(err, iter.Close()) }()
	if !iter.SeekFirst(ctx) {
		return nil, NewDiscontinuousStampError(ts)
	}
	// base is the position in the run of the first sample of the current domain.
	var base int64
	if ts > iter.TimeRange().Start {
		if base, err = i.position(ctx, iter, ts); err != nil {
			return nil, err
		}
		base = -base
	}
	for {
		n := sampleCount(iter.Size())
		if lo, hi := max(start, base), min(end, base+n); lo < hi {
			dst = append(dst, Segment{
				TimeRange: iter.TimeRange(),
				Alignment: telem.NewAlignment(iter.Position(), uint32(lo-base)),
				Len:       hi - lo,
			})
		}
		if base += n; base >= end {
			return dst, nil
		}
		prevEnd := iter.TimeRange().End
		if !iter.Next() || iter.TimeRange().Start != prevEnd {
			return nil, newDiscontinuousOffsetError(end, base)
		}
	}
}

// Coverage walks the run of continuous domains that holds tr.Start. When the run
// reaches tr.End, Coverage stops there. It returns the end of the domain that reaches
// tr.End and a count of zero, and it does not read the index from disk. Otherwise, it
// returns the end of the run and the number of samples in the run at or after tr.Start.
// It returns zero values when no domain holds tr.Start.
func (i *Domain) Coverage(
	ctx context.Context,
	tr telem.TimeRange,
) (end telem.TimeStamp, count int64, err error) {
	iter := i.DB.OpenIterator(domain.IterRange(tr.Start.SpanRange(telem.TimeSpanMax)))
	defer func() {
		if err = errors.Combine(err, iter.Close()); err != nil {
			end, count = 0, 0
		}
	}()
	if !iter.SeekFirst(ctx) || !iter.TimeRange().ContainsStamp(tr.Start) {
		return 0, 0, nil
	}
	first := iter.TimeRange().Start
	for {
		if end = iter.TimeRange().End; tr.End <= end {
			return end, 0, nil
		}
		count += sampleCount(iter.Size())
		if !iter.Next() || iter.TimeRange().Start != end {
			break
		}
	}
	if tr.Start > first && iter.SeekFirst(ctx) {
		var pos int64
		if pos, err = i.position(ctx, iter, tr.Start); err != nil {
			return 0, 0, err
		}
		count -= pos
	}
	return end, count, nil
}

// position returns the position in the current domain of iter of the first sample at or
// after ts.
func (i *Domain) position(
	ctx context.Context,
	iter *domain.Iterator,
	ts telem.TimeStamp,
) (pos int64, err error) {
	r, err := iter.OpenReader(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Combine(err, r.Close()) }()
	approx, err := i.search(ts, r)
	return approx.Upper, err
}

// Stamp calculates an approximate ending timestamp for a range given a known distance
// in the number of samples. This operation may be understood as the
// opposite of Distance.
// Stamp assumes the caller is aware of discontinuities in the underlying time
// series, and will calculate the ending timestamp even across discontinuous ranges.
func (i *Domain) Stamp(
	ctx context.Context,
	ref telem.TimeStamp,
	offset int64,
	continuous bool,
) (approx TimeStampApproximation, err error) {
	ctx, span := i.T.Bench(ctx, "stamp")
	defer func() {
		if err != nil {
			approx = TimeStampApproximation{}
		}
		err = span.EndWith(err, ErrDiscontinuous)
	}()
	switch {
	case offset == 0:
		approx, err = i.zeroStamp(ctx, ref)
	case offset < 0:
		approx, err = i.backwardStamp(ctx, ref, offset, continuous)
	default:
		approx, err = i.forwardStamp(ctx, ref, offset, continuous)
	}
	return approx, err
}

// EndAfter returns the exclusive end of the time range that starts at ref and holds the
// next n samples of the index, which is the timestamp of sample n at or after ref,
// counting from zero. The end is always after ref. A run of equal timestamps can make
// the range hold more or fewer than n samples. It returns telem.TimeStampMax when n or
// fewer samples follow ref. Gaps between domains do not count.
func (i *Domain) EndAfter(
	ctx context.Context,
	ref telem.TimeStamp,
	n int64,
) (end telem.TimeStamp, err error) {
	iter := i.DB.OpenIterator(domain.IterRange(ref.SpanRange(telem.TimeSpanMax)))
	defer func() {
		if err = errors.Combine(err, iter.Close()); err != nil {
			end = 0
		}
	}()
	for ok := iter.SeekFirst(ctx); ok; ok = iter.Next() {
		var found bool
		if end, found, n, err = i.endInDomain(ctx, iter, ref, n); err != nil || found {
			return end, err
		}
	}
	return telem.TimeStampMax, nil
}

// endInDomain looks for sample n at or after ref in the domain at iter. When the domain
// holds it, endInDomain returns the end that EndAfter returns. Otherwise it returns the
// count still owed after the domain's samples.
func (i *Domain) endInDomain(
	ctx context.Context,
	iter *domain.Iterator,
	ref telem.TimeStamp,
	n int64,
) (end telem.TimeStamp, found bool, remaining int64, err error) {
	r, err := iter.OpenReader(ctx)
	if err != nil {
		return 0, false, n, err
	}
	defer func() { err = errors.Combine(err, r.Close()) }()
	var start int64
	if iter.TimeRange().Start < ref {
		approx, err := i.search(ref, r)
		if err != nil {
			return 0, false, n, err
		}
		start = approx.Upper
	}
	count := sampleCount(r.Size())
	if start+n < count {
		end, err = newStampReader()(r, byteSize(start+n))
		return max(end, ref+1), true, 0, err
	}
	return 0, false, n - (count - start), nil
}

func (i *Domain) zeroStamp(
	ctx context.Context,
	ref telem.TimeStamp,
) (approx TimeStampApproximation, err error) {
	iter := i.DB.OpenIterator(domain.IterRange(ref.Range(ref + 1)))
	defer func() { err = errors.Combine(err, iter.Close()) }()
	if !iter.SeekFirst(ctx) {
		err = errors.Wrapf(
			domain.ErrRangeNotFound,
			"cannot find stamp start timestamp %s",
			ref,
		)
		return approx, err
	}
	var r *domain.Reader
	r, err = iter.OpenReader(ctx)
	if err != nil {
		return approx, err
	}
	defer func() { err = errors.Combine(err, r.Close()) }()
	startApprox, err := i.search(ref, r)
	if err != nil {
		return approx, err
	}
	readStamp := newStampReader()
	if !startApprox.Exact() {
		approx.Upper, err = readStamp(r, byteSize(startApprox.Upper))
		return approx, err
	}
	s, err := readStamp(r, byteSize(startApprox.Upper))
	approx = Exactly(s)
	return approx, err
}

func (i *Domain) forwardStamp(
	ctx context.Context,
	ref telem.TimeStamp,
	offset int64,
	continuous bool,
) (approx TimeStampApproximation, err error) {
	iter := i.DB.OpenIterator(domain.IterRange(ref.SpanRange(telem.TimeSpanMax)))
	defer func() { err = errors.Combine(err, iter.Close()) }()

	effectiveDomainLen, err := i.seekFirstDomain(ctx, iter, ref, offset, continuous)
	if err != nil {
		return approx, err
	}

	r, err := iter.OpenReader(ctx)
	if err != nil {
		return approx, err
	}
	defer func() { err = errors.Combine(err, r.Close()) }()

	startApprox, err := i.search(ref, r)
	if err != nil {
		return approx, err
	}

	// endOffset is the upper-bound distance of the desired sample from the start of the
	// domain.
	domainLen := sampleCount(iter.Size())
	endOffset := startApprox.Upper + offset

	// If the upper and lower bounds are exact of the startOffset are exact, then if the
	// lower is out of the file, the stamp is discontinuous.
	// If they are not exact, and the lower bound is the last sample, then the upper
	// bound must be discontinuous as well.
	if continuous &&
		((startApprox.Exact() && startApprox.Lower+offset >= effectiveDomainLen) ||
			(!startApprox.Exact() && startApprox.Lower+offset >= effectiveDomainLen-1)) {
		err = newDiscontinuousOffsetError(startApprox.Upper+offset, effectiveDomainLen)
		return approx, err
	}

	if endOffset >= domainLen {
		var found bool
		if endOffset, found = seekForward(iter, endOffset, domainLen); !found {
			if continuous {
				err = newDiscontinuousOffsetError(endOffset, effectiveDomainLen)
				return approx, err
			}
			approx = Between(iter.TimeRange().End, telem.TimeStampMax)
			return approx, err
		}
		if r, err = reopenReader(ctx, iter, r); err != nil {
			return approx, err
		}
	}

	upperTSByteOffset := byteSize(endOffset)
	lowerTSOffset := endOffset - startApprox.Span()
	lowerTSByteOffset := byteSize(lowerTSOffset)
	return i.approximateStamp(ctx, r, iter, upperTSByteOffset, lowerTSByteOffset)
}

// seekFirstDomain moves iter to the first domain at or after ref and returns the
// length of the effective domain that holds ref.
func (i *Domain) seekFirstDomain(
	ctx context.Context,
	iter *domain.Iterator,
	ref telem.TimeStamp,
	offset int64,
	continuous bool,
) (int64, error) {
	if !iter.SeekFirst(ctx) {
		return 0, NewDiscontinuousStampError(ref)
	}

	effectiveDomainBounds, effectiveDomainLen := resolveForwardEffectiveDomainTR(iter)

	if !effectiveDomainBounds.ContainsStamp(ref) ||
		(continuous && offset >= effectiveDomainLen) {
		return 0, newDiscontinuousOffsetError(offset, effectiveDomainLen)
	}

	if !iter.SeekFirst(ctx) {
		// Reset the iterator position after using it to determine effective bound.
		i.L.DPanic("iterator seekFirst failed in stamp")
	}
	return effectiveDomainLen, nil
}

// seekForward moves iter forward from a domain of domainLen samples until a domain
// holds the sample offset samples after the start of that first domain. It returns the
// offset of the sample within the domain it stops on, and false when no domain holds
// it.
func seekForward(iter *domain.Iterator, offset, domainLen int64) (int64, bool) {
	totalTraversed := domainLen
	for {
		if !iter.Next() {
			return offset, false
		}
		domainLen = sampleCount(iter.Size())
		totalTraversed += domainLen
		if offset < totalTraversed {
			return offset - (totalTraversed - domainLen), true
		}
	}
}

// approximateStamp brackets the stamp between the timestamps at upperTSByteOffset and
// lowerTSByteOffset within the current domain. A negative lowerTSByteOffset places the
// lower timestamp that many bytes before the domain, i.e. at the tail of the previous
// one. approximateStamp leaves iter on that previous domain.
func (i *Domain) approximateStamp(
	ctx context.Context,
	r *domain.Reader,
	iter *domain.Iterator,
	upperTSByteOffset,
	lowerTSByteOffset telem.Size,
) (TimeStampApproximation, error) {
	readStamp := newStampReader()
	upperTS, err := readStamp(r, upperTSByteOffset)
	if err != nil {
		return TimeStampApproximation{}, err
	}
	if lowerTSByteOffset >= 0 {
		lowerTS, err := readStamp(r, lowerTSByteOffset)
		return Between(lowerTS, upperTS), err
	}
	lowerTS, ok, err := i.previousDomainStamp(ctx, iter, -lowerTSByteOffset)
	if err != nil {
		return TimeStampApproximation{}, err
	}
	// Nothing precedes the first domain, so no sample bounds the stamp from below.
	if !ok {
		return Between(telem.TimeStampMin, upperTS), nil
	}
	return Between(lowerTS, upperTS), nil
}

// previousDomainStamp reads the timestamp sitting offset bytes before the end of the
// domain preceding iter's current position, moving iter onto that domain. ok is false
// when iter is already on the first domain.
func (i *Domain) previousDomainStamp(
	ctx context.Context,
	iter *domain.Iterator,
	offset telem.Size,
) (ts telem.TimeStamp, ok bool, err error) {
	if !iter.Prev() {
		return 0, false, nil
	}
	r, err := iter.OpenReader(ctx)
	if err != nil {
		return 0, false, err
	}
	defer func() { err = errors.Combine(err, r.Close()) }()
	ts, err = newStampReader()(r, iter.Size()-offset)
	return ts, true, err
}

// BackwardStamp calculates an approximate starting timestamp for a range given a known
// distance in the number of samples, working backwards from the reference timestamp.
// This operation is similar to Stamp but works in the reverse direction.
func (i *Domain) backwardStamp(
	ctx context.Context,
	ref telem.TimeStamp,
	offset int64,
	continuous bool,
) (approx TimeStampApproximation, err error) {
	ctx, span := i.T.Bench(ctx, "backward_stamp")
	defer func() { err = span.EndWith(err, ErrDiscontinuous) }()

	iter := i.DB.OpenIterator(domain.IterRange(telem.TimeStamp(0).Range(ref + 1)))
	defer func() { err = errors.Combine(err, iter.Close()) }()

	ref, effectiveDomainLen, err := i.seekLastDomain(ctx, iter, ref, offset, continuous)
	if err != nil {
		return approx, err
	}

	r, err := iter.OpenReader(ctx)
	if err != nil {
		return approx, err
	}
	defer func() { err = errors.Combine(err, r.Close()) }()

	startApprox, err := i.search(ref, r)
	if err != nil {
		return approx, err
	}

	// endOffset is the lower-bound distance of the desired sample from the end of the
	// domain.
	domainLen := sampleCount(iter.Size())
	endOffset := domainLen - startApprox.Upper - offset

	// If the upper and lower bounds are exact of the startOffset are exact, then if the
	// lower is out of the file, the stamp is discontinuous.
	// If they are not exact, and the lower bound is the first sample, then the upper
	// bound must be discontinuous as well.
	if continuous && endOffset+startApprox.Span() > effectiveDomainLen {
		err = newDiscontinuousOffsetError(endOffset, 0)
		return approx, err
	}

	// endOffset counts backwards from the end of the domain, so an offset of exactly
	// domainLen still lands on the domain's first sample.
	if endOffset > domainLen {
		var found bool
		if endOffset, domainLen, found = seekBackward(
			iter,
			endOffset,
			domainLen,
		); !found {
			if continuous {
				err = newDiscontinuousOffsetError(endOffset, domainLen)
				return approx, err
			}
			approx = Between(telem.TimeStampMin, iter.TimeRange().Start)
			return approx, err
		}
		if r, err = reopenReader(ctx, iter, r); err != nil {
			return approx, err
		}
	}

	upperTSByteOffset := iter.Size() - byteSize(endOffset)
	lowerTSByteOffset := iter.Size() - byteSize(endOffset+startApprox.Span())
	return i.approximateStamp(ctx, r, iter, upperTSByteOffset, lowerTSByteOffset)
}

// seekLastDomain moves iter to the last domain at or before ref and returns the
// length of the effective domain that holds ref. It moves ref off the end of that
// domain onto its last possible sample.
func (i *Domain) seekLastDomain(
	ctx context.Context,
	iter *domain.Iterator,
	ref telem.TimeStamp,
	offset int64,
	continuous bool,
) (telem.TimeStamp, int64, error) {
	absOffset := -offset
	if !iter.SeekLast(ctx) {
		return 0, 0, NewDiscontinuousStampError(ref)
	}

	effectiveDomainBounds, effectiveDomainLen := resolveBackwardEffectiveDomainTR(iter)

	if ref == effectiveDomainBounds.End {
		ref -= 1
	}

	if (!effectiveDomainBounds.ContainsStamp(ref)) ||
		(continuous && absOffset >= effectiveDomainLen) {
		return 0, 0, newDiscontinuousOffsetError(offset, effectiveDomainLen)
	}

	if !iter.SeekLast(ctx) {
		// Reset the iterator position after using it to determine effective bound.
		i.L.DPanic("iterator seekFirst failed in stamp")
	}
	return ref, effectiveDomainLen, nil
}

// seekBackward moves iter backward from a domain of domainLen samples until a domain
// holds the sample offset samples before the end of that first domain. It returns the
// offset of the sample from the end of the domain it stops on, the length of that
// domain, and false when no domain holds it.
func seekBackward(
	iter *domain.Iterator,
	offset, domainLen int64,
) (int64, int64, bool) {
	totalTraversed := domainLen
	for {
		if !iter.Prev() {
			return offset, domainLen, false
		}
		domainLen = sampleCount(iter.Size())
		totalTraversed += domainLen
		if offset <= totalTraversed {
			return offset - (totalTraversed - domainLen), domainLen, true
		}
	}
}

// resolveForwardEffectiveDomainTR returns the TimeRange and length of the underlying
// domain(s). The effective domain can be many continuous domains as long as they're
// immediately continuous, i.e., the end of one domain is the start of the other.
func resolveForwardEffectiveDomainTR(
	i *domain.Iterator,
) (effectiveDomainBounds telem.TimeRange, effectiveDomainLen int64) {
	effectiveDomainBounds = i.TimeRange()
	effectiveDomainLen = sampleCount(i.Size())
	for {
		currentDomainEnd := i.TimeRange().End
		if !i.Next() {
			return effectiveDomainBounds, effectiveDomainLen
		}
		nextDomainStart := i.TimeRange().Start

		if currentDomainEnd != nextDomainStart {
			return effectiveDomainBounds, effectiveDomainLen
		}
		effectiveDomainBounds.End = i.TimeRange().End
		effectiveDomainLen += sampleCount(i.Size())
	}
}

// resolveForwardEffectiveDomainTR returns the TimeRange and length of the underlying
// domain(s). The effective domain can be many continuous domains as long as they're
// immediately continuous, i.e., the end of one domain is the start of the other.
func resolveBackwardEffectiveDomainTR(
	i *domain.Iterator,
) (effectiveDomainBounds telem.TimeRange, effectiveDomainLen int64) {
	effectiveDomainBounds = i.TimeRange()
	effectiveDomainLen = sampleCount(i.Size())

	for {
		currentDomainStart := i.TimeRange().Start
		if !i.Prev() {
			return effectiveDomainBounds, effectiveDomainLen
		}
		previousDomainEnd := i.TimeRange().End

		if currentDomainStart != previousDomainEnd {
			return effectiveDomainBounds, effectiveDomainLen
		}
		effectiveDomainBounds.Start = i.TimeRange().Start
		effectiveDomainLen += sampleCount(i.Size())
	}
}

// search returns an approximation for the number of samples before a given timestamp.
// If the timestamp exists in the underlying index, the approximation will be exact.
func (i *Domain) search(
	ts telem.TimeStamp,
	r *domain.Reader,
) (Approximation[int64], error) {
	var (
		start int64
		end   = sampleCount(r.Size()) - 1
		read  = newStampReader()
		midTs telem.TimeStamp
		err   error
	)
	for start <= end {
		mid := (start + end) / 2
		if midTs, err = read(r, byteSize(mid)); err != nil {
			return Exactly[int64](0), err
		}
		switch {
		case ts == midTs:
			return Exactly(mid), nil
		case midTs < ts:
			start = mid + 1
		default:
			end = mid - 1
		}
	}
	return Between(end, end+1), nil
}

func newStampReader() func(r io.ReaderAt, offset telem.Size) (telem.TimeStamp, error) {
	buf := make([]byte, sampleDensity)
	return func(r io.ReaderAt, offset telem.Size) (telem.TimeStamp, error) {
		n, err := r.ReadAt(buf, int64(offset))
		if err != nil {
			return 0, err
		}
		if len(buf) != n {
			err := errors.Newf(
				"[domain.index] unexpected failure to read %d bytes for timestamp lookup, only received %d",
				n,
				len(buf),
			)
			zap.S().DPanic(err)
			return 0, err
		}
		return unsafe.CastBytes[telem.TimeStamp](buf)
	}
}
