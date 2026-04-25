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
	"slices"
	"sync"

	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// offsetTable stores byte offsets for each sample in a single variable-length
// domain. Offsets are uint32, matching domain.pointer.size; domains larger than
// ~4GB are unsupported at the storage layer. end is the End timestamp of the
// pointer at the time the table was built, used to invalidate the cache when a
// subsequent write appends more data and extends the pointer's End.
type offsetTable struct {
	end         telem.TimeStamp
	offsets     []uint32
	sampleCount int64
}

func (t *offsetTable) byteOffsetAt(sampleIdx int64) telem.Size {
	return telem.Size(t.offsets[sampleIdx])
}

// offsetCache memoizes per-domain offset tables for variable-length channels.
// Tables are keyed by the Start timestamp of the domain pointer they describe.
// Pointer Starts are unique within a DB (non-overlapping ranges) and immutable
// once set, so they are stable cache keys even when other pointers are
// inserted or removed before this one.
type offsetCache struct {
	mu     sync.RWMutex
	tables map[telem.TimeStamp]*offsetTable
}

func newOffsetCache() *offsetCache {
	return &offsetCache{tables: make(map[telem.TimeStamp]*offsetTable)}
}

func (c *offsetCache) get(start telem.TimeStamp) (*offsetTable, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t, ok := c.tables[start]
	return t, ok
}

func (c *offsetCache) set(start telem.TimeStamp, t *offsetTable) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tables[start] = t
}

func (c *offsetCache) invalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.tables)
}

func buildOffsetTable(r *domain.Reader, domainSize telem.Size, end telem.TimeStamp) (*offsetTable, error) {
	t := &offsetTable{end: end}
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
	tr := iter.TimeRange()
	// A pointer's Start is immutable but its End advances as the writer commits more
	// samples to the same domain. Gate the cache hit on the cached End matching the
	// pointer's current End so a cached table from before a later commit cannot
	// answer queries against the extended range.
	if cached, ok := r.cache.get(tr.Start); ok && cached.end == tr.End {
		return cached, nil
	}
	rd, err := iter.OpenReader(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, rd.Close()) }()
	t, err = buildOffsetTable(rd, telem.Size(iter.Size()), tr.End)
	if err != nil {
		return nil, err
	}
	r.cache.set(tr.Start, t)
	return t, nil
}

// newTracker returns a per-writer offsetTracker bound to this resolver, anchored to
// the start timestamp of the writer's first domain. Fixed-density trackers carry the
// resolver's density and use it to convert per-write byte counts to sample counts;
// cache-backed trackers also track per-sample offsets within the current domain so
// they can publish them to the cache on commit.
func (r *offsetResolver) newTracker(start telem.TimeStamp) *offsetTracker {
	t := &offsetTracker{resolver: r, currentStart: start}
	if r.cache == nil {
		t.density = r.density
	}
	return t
}

// offsetTracker tracks state for a single unary writer session. Fixed-density and
// variable-length channels share the cumulative-count behavior at the alignment math
// site by deriving it from the underlying domain.Writer, which is safe across control
// handoffs that reuse the same domain.Writer. Variable-length channels additionally
// track per-domain offsets and sample counts so they can publish them to the offset
// cache on commit.
type offsetTracker struct {
	// resolver is the offsetResolver this tracker was created from. Used by publish to
	// install committed offsets into the resolver's cache.
	resolver *offsetResolver
	// density is the sample density of the channel for fixed-density trackers; zero
	// for variable-length trackers.
	density telem.Density
	// currentStart is the start timestamp of the domain currently being tracked.
	// Updated to commitEnd on rollover so subsequent publishes target the new domain.
	// Unused for fixed-density.
	currentStart telem.TimeStamp
	// domainBytes is the byte count written to the current domain since the last
	// rollover. Used as the base offset for the next write so per-sample offsets are
	// relative to the start of the current file. Reset on rollover. Unused for
	// fixed-density.
	domainBytes int64
	// domainOffsets is the running list of per-sample byte offsets within the current
	// domain for variable-length trackers; nil for fixed-density. Reset on rollover.
	domainOffsets []uint32
	// sessionSamples is the cumulative sample count for variable-length trackers
	// across every domain in this writer's session. Used for alignment math and the
	// end-timestamp lookup at commit time. Does not transfer on control handoff.
	// Unused for fixed-density (count derives from dw.Len() instead).
	sessionSamples int64
}

// count returns the cumulative sample count for the current writer. For fixed-density
// trackers it derives from dw.Len(), which is shared across control handoff that
// reuses the same domain.Writer. For variable-length trackers it returns the
// tracker's internal session count, which does not transfer on control handoff.
func (t *offsetTracker) count(dw *domain.Writer) int64 {
	if t.density != 0 {
		return t.density.SampleCount(telem.Size(dw.Len()))
	}
	return t.sessionSamples
}

// record advances the tracker to reflect a Write of data starting at baseByteOffset
// within the current domain. No-op for fixed-density trackers (they derive their
// state from the domain.Writer). For variable-length trackers, it walks length
// prefixes and appends per-sample offsets, advances domainBytes, and increments
// sessionSamples.
func (t *offsetTracker) record(data []byte, baseByteOffset uint32) {
	if t.density != 0 {
		return
	}
	t.domainBytes += int64(len(data))
	offset := 0
	for offset+4 <= len(data) {
		t.domainOffsets = append(t.domainOffsets, baseByteOffset+uint32(offset))
		length := int(binary.LittleEndian.Uint32(data[offset:]))
		offset += 4 + length
		t.sessionSamples++
	}
}

// publish snaps the tracker's per-domain offsets into the resolver's cache under
// currentStart, gated on the just-committed domain's end timestamp. It is a no-op
// for fixed-density trackers and for empty trackers (where there is nothing new to
// cache). The published offsets are cloned so subsequent appends to the tracker
// cannot mutate the cache entry.
func (t *offsetTracker) publish(end telem.TimeStamp) {
	if t.density != 0 || t.resolver.cache == nil || len(t.domainOffsets) == 0 {
		return
	}
	t.resolver.cache.set(t.currentStart, &offsetTable{
		end:         end,
		offsets:     slices.Clone(t.domainOffsets),
		sampleCount: int64(len(t.domainOffsets)),
	})
}

// rollover is the OnRollover hook the tracker installs on its underlying
// domain.Writer. It publishes the just-finished domain's accumulated offsets to the
// cache, resets the per-domain fields, and advances currentStart to the new
// domain's start. Cumulative session state (sessionSamples) is preserved.
func (t *offsetTracker) rollover(commitEnd telem.TimeStamp) {
	t.publish(commitEnd)
	t.currentStart = commitEnd
	t.domainBytes = 0
	t.domainOffsets = nil
}
