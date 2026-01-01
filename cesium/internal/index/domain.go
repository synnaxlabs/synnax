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
	"fmt"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/core"
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
	Channel core.Channel
}

var sampleDensity = telem.TimeStampT.Density()

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
//
// The distance method also returns an alignment pair, which represents the
// alignment of the lower and upper bounds. The alignment pair is a 64-bit integer
// where the lower 32 bits represent the domain and the upper 32 bits represent the
// sample index within the domain.
func (i *Domain) Distance(
	ctx context.Context,
	tr telem.TimeRange,
	continuous ContinuousPolicy,
) (approx DistanceApproximation, alignment telem.Alignment, err error) {
	ctx, span := i.T.Bench(ctx, "distance")
	defer func() { _ = span.EndWith(err, ErrDiscontinuous) }()

	iter := i.DB.OpenIterator(domain.IteratorConfig{Bounds: tr})
	defer func() { err = errors.Combine(err, iter.Close()) }()

	if !iter.SeekFirst(ctx) {
		// If the domain with the given time range doesn't exist in the database, then
		// no data exists for that time range, so it's impossible to approximate a
		// distance.
		err = NewErrDiscontinuousTR(tr)
		return
	}

	effectiveDomainTR, _ := resolveForwardEffectiveDomainTR(iter)
	if !iter.SeekFirst(ctx) {
		// Reset the iterator position after using it to determine effective bound.
		// A result of false in this case should be impossible, as we already validated
		// the iterator by calling SeekFirst in a previous call.
		i.L.DPanic("iterator seekFirst failed in stamp")
		return
	}

	// If the time range is not contained within the effective domain, then it's
	// discontinuous, and we return early if the user doesn't want discontinuous
	// results.
	if !effectiveDomainTR.ContainsRange(tr) && continuous {
		err = NewErrDiscontinuousTR(tr)
		return
	}

	// If the time range is zero, then the distance is zero.
	if tr.Span().IsZero() {
		alignment = telem.NewAlignment(iter.Position(), 0)
		return
	}

	// Open a new reader on the domain at the start of the range.
	r, err := iter.OpenReader(ctx)
	if err != nil {
		return
	}
	defer func() { err = errors.Combine(err, r.Close()) }()

	var startApprox, endApprox Approximation[int64]

	startApprox, err = i.search(tr.Start, r)
	if err != nil {
		return
	}
	approx.StartExact = startApprox.Exact()

	if iter.TimeRange().ContainsStamp(tr.End) || tr.End == iter.TimeRange().End {
		// If the current domain contains the end of the time range, then everything
		// is continuous and within the current domain.
		endApprox, err = i.search(tr.End, r)
		approx.Approximation = Between(
			endApprox.Lower-startApprox.Upper,
			endApprox.Upper-startApprox.Lower,
		)
		approx.EndExact = endApprox.Exact()

		alignment = telem.NewAlignment(iter.Position(), uint32(endApprox.Upper))
		return
	} else if continuous &&
		!effectiveDomainTR.ContainsStamp(tr.End) &&
		effectiveDomainTR.End != tr.End {
		// Otherwise, unless the effective domain contains the end of the time range
		// the distance is discontinuous
		err = NewErrDiscontinuousTR(tr)
		return
	}

	var (
		// Length of the current domain
		domainLen = sampleCount(r.Size())
		// the total number of samples traversed as we move through domains
		totalTraversed int64 = 0
		// Distance from the end of the domain to the start approximation.
		startToFirstEnd = Between(domainLen-startApprox.Upper, domainLen-startApprox.Lower)
	)

	for {
		if !iter.Next() || (continuous && !effectiveDomainTR.ContainsRange(iter.TimeRange())) {
			if continuous {
				err = NewErrDiscontinuousTR(tr)
				return
			}
			approx.Approximation = Between(
				startToFirstEnd.Lower+totalTraversed,
				startToFirstEnd.Upper+totalTraversed,
			)
			alignment = telem.NewAlignment(iter.Position(), uint32(sampleCount(iter.Size())))
			return
		}
		if iter.TimeRange().ContainsStamp(tr.End) {
			if err = r.Close(); err != nil {
				return
			}
			if r, err = iter.OpenReader(ctx); err != nil {
				return
			}
			if endApprox, err = i.search(tr.End, r); err != nil {
				return
			}
			approx.EndExact = endApprox.Exact()
			alignment = telem.NewAlignment(iter.Position(), uint32(endApprox.Lower))
			approx.Approximation = Between(
				startToFirstEnd.Lower+totalTraversed+endApprox.Lower,
				startToFirstEnd.Upper+totalTraversed+endApprox.Upper,
			)
			return
		}
		totalTraversed += sampleCount(iter.Size())
	}
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
	defer func() { _ = span.EndWith(err, ErrDiscontinuous) }()
	if offset == 0 {
		approx, err = i.zeroStamp(ctx, ref)
	} else if offset < 0 {
		approx, err = i.backwardStamp(ctx, ref, offset, continuous)
	} else {
		approx, err = i.forwardStamp(ctx, ref, offset, continuous)
	}
	return approx, err
}

func (i *Domain) zeroStamp(
	ctx context.Context,
	ref telem.TimeStamp,
) (approx TimeStampApproximation, err error) {
	iter := i.DB.OpenIterator(domain.IterRange(ref.Range(ref + 1)))
	defer func() { err = errors.Combine(err, iter.Close()) }()
	if !iter.SeekFirst(ctx) {
		err = errors.Wrapf(domain.ErrRangeNotFound, "cannot find stamp start timestamp %s", ref)
		return
	}
	var r *domain.Reader
	r, err = iter.OpenReader(ctx)
	if err != nil {
		return
	}
	defer func() { err = errors.Combine(err, r.Close()) }()
	startApprox, err := i.search(ref, r)
	readStamp := newStampReader()
	if !startApprox.Exact() {
		approx.Upper, err = readStamp(r, byteSize(startApprox.Upper))
		return
	}
	s, err := readStamp(r, byteSize(startApprox.Upper))
	approx = Exactly[telem.TimeStamp](s)
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

	if !iter.SeekFirst(ctx) {
		err = NewErrDiscontinuousStamp(ref)
		return
	}

	effectiveDomainBounds, effectiveDomainLen := resolveForwardEffectiveDomainTR(iter)

	if !effectiveDomainBounds.ContainsStamp(ref) ||
		(continuous && offset >= effectiveDomainLen) {
		err = NewErrDiscontinuousOffset(offset, effectiveDomainLen)
		return
	}

	if !iter.SeekFirst(ctx) {
		// Reset the iterator position after using it to determine effective bound.
		i.L.DPanic("iterator seekFirst failed in stamp")
	}

	r, err := iter.OpenReader(ctx)
	if err != nil {
		return
	}
	defer func() { err = errors.Combine(err, r.Close()) }()

	startApprox, err := i.search(ref, r)
	if err != nil {
		return
	}

	// endOffset is the upper-bound distance of the desired sample from the start of the
	// domain.
	domainLen := sampleCount(iter.Size())
	endOffset := startApprox.Upper + offset

	// If the upper and lower bounds are exact of the startOffset are exact, then if the
	// lower is out of the file, the stamp is discontinuous.
	// If they are not exact, and the lower bound is the last sample, then the upper
	// bound must be discontinuous as well.
	if continuous {
		if (startApprox.Exact() && startApprox.Lower+offset >= effectiveDomainLen) ||
			(!startApprox.Exact() && startApprox.Lower+offset >= effectiveDomainLen-1) {
			err = NewErrDiscontinuousOffset(startApprox.Upper+offset, effectiveDomainLen)
			return
		}
	}

	totalTraversed := domainLen
	if endOffset >= domainLen {
		for {
			if !iter.Next() {
				if continuous {
					err = NewErrDiscontinuousOffset(endOffset, effectiveDomainLen)
					return
				}
				approx = Between(iter.TimeRange().End, telem.TimeStampMax)
				return
			}
			domainLen = sampleCount(iter.Size())
			totalTraversed += domainLen
			if endOffset < totalTraversed {
				if err = r.Close(); err != nil {
					return
				}
				r, err = iter.OpenReader(ctx)
				if err != nil {
					return
				}
				endOffset -= totalTraversed - domainLen
				break
			}
		}
	}

	upperTSByteOffset := byteSize(endOffset)
	lowerTSOffset := endOffset - startApprox.Span()
	lowerTSByteOffset := byteSize(lowerTSOffset)
	return i.approximateStamp(
		ctx,
		r,
		iter,
		endOffset,
		effectiveDomainLen,
		upperTSByteOffset,
		lowerTSByteOffset,
	)
}

func (i *Domain) approximateStamp(
	ctx context.Context,
	r *domain.Reader,
	iter *domain.Iterator,
	endOffset int64,
	effectiveDomainLen int64,
	upperTSByteOffset,
	lowerTSByteOffset telem.Size,
) (TimeStampApproximation, error) {
	readStamp := newStampReader()
	upperTS, err := readStamp(r, upperTSByteOffset)
	if err != nil {
		return TimeStampApproximation{}, err
	}
	if lowerTSByteOffset >= 0 {
		lowerTs, err := readStamp(r, lowerTSByteOffset)
		return Between(lowerTs, upperTS), err
	}
	// Edge case: end timestamps are split between two different files, so we must go
	// back to read the lower bound.
	if !iter.Prev() {
		i.L.DPanic("iterator prev failed in stamp")
		return TimeStampApproximation{}, NewErrDiscontinuousOffset(endOffset, effectiveDomainLen)
	}
	if err = r.Close(); err != nil {
		return TimeStampApproximation{}, err
	}
	if r, err = iter.OpenReader(ctx); err != nil {
		return TimeStampApproximation{}, err
	}
	lowerTS, err := readStamp(r, iter.Size()+lowerTSByteOffset)
	return Between(lowerTS, upperTS), err
}

// BackwardStamp calculates an approximate starting timestamp for a range given a known distance
// in the number of samples, working backwards from the reference timestamp. This operation
// is similar to Stamp but works in the reverse direction.
func (i *Domain) backwardStamp(
	ctx context.Context,
	ref telem.TimeStamp,
	offset int64,
	continuous bool,
) (approx TimeStampApproximation, err error) {
	ctx, span := i.T.Bench(ctx, "backward_stamp")
	defer func() { _ = span.EndWith(err, ErrDiscontinuous) }()
	absOffset := -offset

	iter := i.DB.OpenIterator(domain.IterRange(telem.TimeStamp(0).Range(ref + 1)))
	defer func() { err = errors.Combine(err, iter.Close()) }()

	if !iter.SeekLast(ctx) {
		err = NewErrDiscontinuousStamp(ref)
		return
	}

	effectiveDomainBounds, effectiveDomainLen := resolveBackwardEffectiveDomainTR(iter)

	if ref == effectiveDomainBounds.End {
		ref -= 1
	}

	if (!effectiveDomainBounds.ContainsStamp(ref)) ||
		(continuous && absOffset >= effectiveDomainLen) {
		err = NewErrDiscontinuousOffset(offset, effectiveDomainLen)
		return
	}

	if !iter.SeekLast(ctx) {
		// Reset the iterator position after using it to determine effective bound.
		i.L.DPanic("iterator seekFirst failed in stamp")
	}

	r, err := iter.OpenReader(ctx)
	if err != nil {
		return
	}
	defer func() { err = errors.Combine(err, r.Close()) }()

	startApprox, err := i.search(ref, r)
	if err != nil {
		return
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
		err = NewErrDiscontinuousOffset(endOffset, 0)
		return
	}

	totalTraversed := domainLen
	if endOffset >= domainLen {
		for {
			if !iter.Prev() {
				if continuous {
					err = NewErrDiscontinuousOffset(endOffset, domainLen)
					return
				}
				approx = Between(telem.TimeStampMin, iter.TimeRange().Start)
				return
			}
			domainLen = sampleCount(iter.Size())
			totalTraversed += domainLen
			if endOffset <= totalTraversed {
				if err = r.Close(); err != nil {
					return
				}
				r, err = iter.OpenReader(ctx)
				if err != nil {
					return
				}
				endOffset -= totalTraversed - domainLen
				break
			}
		}
	}

	upperTSByteOffset := iter.Size() - byteSize(endOffset)
	lowerTSByteOffset := iter.Size() - byteSize(endOffset+startApprox.Span())
	return i.approximateStamp(
		ctx,
		r,
		iter,
		endOffset,
		effectiveDomainLen,
		upperTSByteOffset,
		lowerTSByteOffset,
	)
}

// resolveForwardEffectiveDomainTR returns the TimeRange and length of the underlying domain(s).
// The effective domain can be many continuous domains as long as they're immediately
// continuous, i.e., the end of one domain is the start of the other.
func resolveForwardEffectiveDomainTR(i *domain.Iterator) (effectiveDomainBounds telem.TimeRange, effectiveDomainLen int64) {
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

// resolveForwardEffectiveDomainTR returns the TimeRange and length of the underlying domain(s).
// The effective domain can be many continuous domains as long as they're immediately
// continuous, i.e., the end of one domain is the start of the other.
func resolveBackwardEffectiveDomainTR(i *domain.Iterator) (effectiveDomainBounds telem.TimeRange, effectiveDomainLen int64) {
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

// search returns an approximation for the number of samples before a given timestamp. If the
// timestamp exists in the underlying index, the approximation will be exact.
func (i *Domain) search(ts telem.TimeStamp, r *domain.Reader) (Approximation[int64], error) {
	var (
		start int64 = 0
		end         = sampleCount(r.Size()) - 1
		read        = newStampReader()
		midTs telem.TimeStamp
		err   error
	)
	for start <= end {
		mid := (start + end) / 2
		if midTs, err = read(r, byteSize(mid)); err != nil {
			return Exactly[int64](0), err
		}
		if ts == midTs {
			return Exactly(mid), nil
		} else if midTs < ts {
			start = mid + 1
		} else {
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
			err := errors.Newf("[domain.index] unexpected failure to read %d bytes for timestamp lookup, only received %d", n, len(buf))
			zap.S().DPanic(err)
			return 0, err
		}
		return unsafe.CastBytes[telem.TimeStamp](buf)
	}
}

// Info returns the key and name of the channel of the index. If the database is
// domain-indexed, the information of the domain channel is returned. If the database
// is rate-based (i.e. self-indexing), the channel itself is returned.
func (i *Domain) Info() string {
	return fmt.Sprintf("domain index: %v", i.Channel)
}
