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
	"context"
	"encoding/binary"

	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/telem"
)

// offsetTracker is per-writer state that tracks sample counts and byte offsets
// as samples are appended to a single domain writer. Writer holds one tracker
// for its lifetime; iterators and deletes do not use trackers.
//
// Fixed-density channels derive their count directly from dw.Len(), so their
// tracker is effectively stateless. Variable-length channels maintain an
// incremental offset table populated by record so that count can answer in
// O(1) without rescanning the raw byte stream.
type offsetTracker interface {
	// count returns the number of samples written to dw so far.
	count(dw *domain.Writer) int64
	// record is called immediately after data is appended to the domain writer.
	// baseByteOffset is dw.Len() prior to the append. Stateless trackers may
	// ignore the hook.
	record(data []byte, baseByteOffset uint32)
}

// offsetResolver is DB-wide state that resolves sample-to-byte offsets in
// completed (read-path) domains. It is shared by Iterator and the Delete code
// path; it is not used on the write hot path. Use newTracker to obtain a
// per-writer offsetTracker from the same resolver instance.
type offsetResolver interface {
	// byteOffset returns the byte offset of the given sample index within the
	// domain that iter is currently positioned at. If sampleIdx is greater than
	// or equal to the domain's total sample count, the returned offset is the
	// end-of-domain offset (telem.Size(iter.Size())).
	byteOffset(
		ctx context.Context,
		iter *domain.Iterator,
		sampleIdx int64,
	) (telem.Size, error)
	// domainSampleCount returns the sample count of the domain that iter is
	// currently positioned at. Called by Iterator.approximateEnd when the view
	// spans the whole domain.
	domainSampleCount(ctx context.Context, iter *domain.Iterator) (int64, error)
	// invalidate drops any cached per-domain state. Called after Delete and
	// GarbageCollect mutate the underlying domain files.
	invalidate()
	// newTracker returns a fresh offsetTracker for a new writer session.
	newTracker() offsetTracker
}

// newResolver returns the resolver appropriate for the given data type:
// densityResolver for fixed-density types, cacheResolver for variable-length
// types.
func newResolver(dt telem.DataType) offsetResolver {
	if dt.IsVariable() {
		return &cacheResolver{cache: newOffsetCache()}
	}
	return densityResolver{density: dt.Density()}
}

// densityResolver implements offsetResolver for fixed-density channels using
// direct density arithmetic. It carries no state; invalidate is a no-op and
// newTracker hands back a matching stateless tracker.
type densityResolver struct{ density telem.Density }

func (r densityResolver) byteOffset(
	_ context.Context,
	iter *domain.Iterator,
	sampleIdx int64,
) (telem.Size, error) {
	total := r.density.SampleCount(telem.Size(iter.Size()))
	if sampleIdx >= total {
		return telem.Size(iter.Size()), nil
	}
	return r.density.Size(sampleIdx), nil
}

func (r densityResolver) domainSampleCount(
	_ context.Context,
	iter *domain.Iterator,
) (int64, error) {
	return r.density.SampleCount(telem.Size(iter.Size())), nil
}

func (densityResolver) invalidate() {}

func (r densityResolver) newTracker() offsetTracker { return densityTracker{density: r.density} }

// densityTracker is the offsetTracker counterpart to densityResolver.
type densityTracker struct{ density telem.Density }

func (t densityTracker) count(dw *domain.Writer) int64 {
	return t.density.SampleCount(telem.Size(dw.Len()))
}

func (densityTracker) record([]byte, uint32) {}

// cacheResolver implements offsetResolver for variable-length channels. It
// maintains a per-domain offset table cache: on first access for a given
// domain, the reader scans the file to build the table; subsequent accesses
// hit the cache. Tables are dropped wholesale after Delete or GarbageCollect.
type cacheResolver struct{ cache *offsetCache }

func (r *cacheResolver) byteOffset(
	ctx context.Context,
	iter *domain.Iterator,
	sampleIdx int64,
) (telem.Size, error) {
	table, err := r.tableFor(ctx, iter)
	if err != nil {
		return 0, err
	}
	if sampleIdx >= table.sampleCount {
		return telem.Size(iter.Size()), nil
	}
	return table.byteOffsetAt(sampleIdx), nil
}

func (r *cacheResolver) domainSampleCount(
	ctx context.Context,
	iter *domain.Iterator,
) (int64, error) {
	table, err := r.tableFor(ctx, iter)
	if err != nil {
		return 0, err
	}
	return table.sampleCount, nil
}

func (r *cacheResolver) invalidate() { r.cache.invalidateAll() }

func (r *cacheResolver) newTracker() offsetTracker { return &cacheTracker{} }

// tableFor returns the offset table for the domain iter is currently at,
// building and caching it on first access.
func (r *cacheResolver) tableFor(ctx context.Context, iter *domain.Iterator) (*offsetTable, error) {
	domainIdx := iter.Position()
	if t, ok := r.cache.get(domainIdx); ok {
		return t, nil
	}
	rd, err := iter.OpenReader(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rd.Close() }()
	t, err := buildOffsetTable(rd, iter.Size())
	if err != nil {
		return nil, err
	}
	r.cache.set(domainIdx, t)
	return t, nil
}

// cacheTracker is the offsetTracker counterpart to cacheResolver. It mirrors
// the per-writer offset table from the variable writer: scan each appended
// byte slice for its length-prefixed records, track cumulative sample count.
type cacheTracker struct {
	offsets     []uint32
	sampleCount int64
}

func (t *cacheTracker) count(*domain.Writer) int64 { return t.sampleCount }

func (t *cacheTracker) record(data []byte, baseByteOffset uint32) {
	offset := 0
	for offset+4 <= len(data) {
		t.offsets = append(t.offsets, baseByteOffset+uint32(offset))
		length := int(binary.LittleEndian.Uint32(data[offset:]))
		offset += 4 + length
		t.sampleCount++
	}
}
