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
	"io"
	"sync"

	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// offsetTable stores byte offsets for each sample in a single variable-length
// domain. Offsets are uint32, matching domain.pointer.size; domains larger than
// ~4GB are unsupported at the storage layer. domainSize is the byte length of
// the domain at the time the table was built, used to invalidate the cache when
// a subsequent write appends more data to the same domain index.
type offsetTable struct {
	domainSize  telem.Size
	offsets     []uint32
	sampleCount int64
}

func (t *offsetTable) byteOffsetAt(sampleIdx int64) telem.Size {
	return telem.Size(t.offsets[sampleIdx])
}

// offsetCache memoizes per-domain offset tables for variable-length channels.
type offsetCache struct {
	mu     sync.RWMutex
	tables map[uint32]*offsetTable
}

func newOffsetCache() *offsetCache {
	return &offsetCache{tables: make(map[uint32]*offsetTable)}
}

func (c *offsetCache) get(domainIdx uint32) (*offsetTable, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t, ok := c.tables[domainIdx]
	return t, ok
}

func (c *offsetCache) set(domainIdx uint32, t *offsetTable) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tables[domainIdx] = t
}

func (c *offsetCache) invalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tables = make(map[uint32]*offsetTable)
}

func buildOffsetTable(r *domain.Reader, domainSize telem.Size) (*offsetTable, error) {
	t := &offsetTable{}
	buf := make([]byte, 4)
	var pos int64
	for pos+4 <= int64(domainSize) {
		t.offsets = append(t.offsets, uint32(pos))
		n, err := r.ReadAt(buf, pos)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n < 4 {
			break
		}
		length := int64(binary.LittleEndian.Uint32(buf))
		pos += 4 + length
		t.sampleCount++
	}
	return t, nil
}

// offsetResolver translates sample indices to byte offsets within domain files.
// Fixed-density channels have a zero cache and rely on density arithmetic;
// variable-length channels carry a per-domain offset cache that is built on
// first access by scanning the length-prefixed records.
type offsetResolver struct {
	density telem.Density
	cache   *offsetCache // nil for fixed-density channels
}

func newOffsetResolver(dt telem.DataType) *offsetResolver {
	if dt.IsVariable() {
		return &offsetResolver{cache: newOffsetCache()}
	}
	return &offsetResolver{density: dt.Density()}
}

// byteOffset returns the byte offset of sampleIdx within iter's current domain.
// If sampleIdx is past the domain's total sample count, returns the end-of-domain
// byte offset.
func (r *offsetResolver) byteOffset(
	ctx context.Context,
	iter *domain.Iterator,
	sampleIdx int64,
) (telem.Size, error) {
	if r.cache == nil {
		total := r.density.SampleCount(telem.Size(iter.Size()))
		if sampleIdx >= total {
			return telem.Size(iter.Size()), nil
		}
		return r.density.Size(sampleIdx), nil
	}
	t, err := r.tableFor(ctx, iter)
	if err != nil {
		return 0, err
	}
	if sampleIdx >= t.sampleCount {
		return telem.Size(iter.Size()), nil
	}
	return t.byteOffsetAt(sampleIdx), nil
}

func (r *offsetResolver) domainSampleCount(
	ctx context.Context,
	iter *domain.Iterator,
) (int64, error) {
	if r.cache == nil {
		return r.density.SampleCount(telem.Size(iter.Size())), nil
	}
	t, err := r.tableFor(ctx, iter)
	if err != nil {
		return 0, err
	}
	return t.sampleCount, nil
}

func (r *offsetResolver) invalidate() {
	if r.cache != nil {
		r.cache.invalidateAll()
	}
}

func (r *offsetResolver) tableFor(ctx context.Context, iter *domain.Iterator) (t *offsetTable, err error) {
	domainIdx := iter.Position()
	size := telem.Size(iter.Size())
	// A domain index is stable across an entire writer session, so a table
	// cached after commit N will have a stale sampleCount if the writer
	// appends more data in commit N+1 against the same domain index. Gate the
	// cache hit on the domain size matching what the table was built from.
	if cached, ok := r.cache.get(domainIdx); ok && cached.domainSize == size {
		return cached, nil
	}
	rd, err := iter.OpenReader(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, rd.Close()) }()
	t, err = buildOffsetTable(rd, size)
	if err != nil {
		return nil, err
	}
	t.domainSize = size
	r.cache.set(domainIdx, t)
	return t, nil
}

// newTracker returns a per-writer offsetTracker bound to this resolver's
// density or cache mode. Fixed-density trackers are stateless; cache-backed
// trackers accumulate sample counts and offsets as writes are recorded.
func (r *offsetResolver) newTracker() *offsetTracker {
	if r.cache == nil {
		return &offsetTracker{density: r.density}
	}
	return &offsetTracker{}
}

// offsetTracker tracks sample counts (and, for variable-length channels, byte
// offsets) during a single writer session. The zero value is a variable-length
// tracker; setting density makes it a stateless fixed-density tracker.
type offsetTracker struct {
	density     telem.Density
	offsets     []uint32
	sampleCount int64
}

func (t *offsetTracker) count(dw *domain.Writer) int64 {
	if t.density != 0 {
		return t.density.SampleCount(telem.Size(dw.Len()))
	}
	return t.sampleCount
}

func (t *offsetTracker) record(data []byte, baseByteOffset uint32) {
	if t.density != 0 {
		return
	}
	offset := 0
	for offset+4 <= len(data) {
		t.offsets = append(t.offsets, baseByteOffset+uint32(offset))
		length := int(binary.LittleEndian.Uint32(data[offset:]))
		offset += 4 + length
		t.sampleCount++
	}
}
