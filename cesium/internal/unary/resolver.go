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
	"container/list"
	"context"
	"io"
	"slices"
	"strconv"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// scanBufferSize is the buffer size used by buildOffsetTable when scanning
// length prefixes during a cache miss. Sized so that a single buffered read
// covers thousands of samples for typical event-rate workloads, amortizing
// syscall overhead without holding more memory than necessary while concurrent
// rebuilds run.
const scanBufferSize = 64 * telem.Kilobyte

// offsetTable stores byte offsets for each sample in a single variable-length
// domain. Offsets are uint32, matching domain.pointer.size; domains larger than
// ~4GB are unsupported at the storage layer. end is the End timestamp of the
// pointer at the time the table was built, used to invalidate the cache when a
// subsequent write appends more data and extends the pointer's End.
type offsetTable struct {
	end         telem.TimeStamp
	offsets     []uint32
	sampleCount int64
	// keyframes holds the ascending indices of the keyframes of a video domain. It is
	// nil for other channels.
	keyframes []uint32
}

// tableOverhead approximates the memory of a table beyond its offsets: the table, its
// cache entry, and its list element.
const tableOverhead = 128 * telem.Byte

func (t *offsetTable) byteOffsetAt(sampleIdx int64) telem.Size {
	return telem.Size(t.offsets[sampleIdx])
}

func (t *offsetTable) size() telem.Size {
	return telem.Size(4*(len(t.offsets)+len(t.keyframes))) + tableOverhead
}

// lastKeyframe returns the index of the last keyframe at or before sample.
func (t *offsetTable) lastKeyframe(sample int64) (int64, bool) {
	j, found := slices.BinarySearch(t.keyframes, uint32(sample))
	if found {
		return sample, true
	}
	if j == 0 {
		return 0, false
	}
	return int64(t.keyframes[j-1]), true
}

// DefaultOffsetCacheBudget is the budget of the cache that a DB opens when its config
// has none.
const DefaultOffsetCacheBudget = 128 * telem.Megabyte

// OffsetCache holds the offset tables of variable-density domains for every DB that
// shares it. When its tables pass the budget, it evicts the least recently used ones.
// A table larger than the whole budget is not cached. Safe for concurrent use.
type OffsetCache struct {
	budget telem.Size
	mu     struct {
		sync.Mutex
		size telem.Size
		// recent orders the cached tables from most to least recently used.
		recent list.List
		tables map[channel.Key]map[telem.TimeStamp]*list.Element
	}
}

type cachedTable struct {
	channel channel.Key
	// start is the Start of the domain pointer. Pointer Starts are unique within a DB
	// and immutable once set, so they are stable keys even when other pointers are
	// inserted or removed before this one.
	start telem.TimeStamp
	table *offsetTable
}

// NewOffsetCache opens a cache that holds at most budget bytes of tables.
func NewOffsetCache(budget telem.Size) *OffsetCache {
	c := &OffsetCache{budget: budget}
	c.mu.tables = make(map[channel.Key]map[telem.TimeStamp]*list.Element)
	return c
}

func (c *OffsetCache) get(ch channel.Key, start telem.TimeStamp) (*offsetTable, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.mu.tables[ch][start]
	if !ok {
		return nil, false
	}
	c.mu.recent.MoveToFront(el)
	return el.Value.(*cachedTable).table, true
}

func (c *OffsetCache) set(ch channel.Key, start telem.TimeStamp, t *offsetTable) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.mu.tables[ch][start]; ok {
		c.remove(el)
	}
	if t.size() > c.budget {
		return
	}
	tables, ok := c.mu.tables[ch]
	if !ok {
		tables = make(map[telem.TimeStamp]*list.Element)
		c.mu.tables[ch] = tables
	}
	tables[start] = c.mu.recent.PushFront(
		&cachedTable{channel: ch, start: start, table: t},
	)
	c.mu.size += t.size()
	for c.mu.size > c.budget {
		c.remove(c.mu.recent.Back())
	}
}

// invalidate removes the tables of the channel's domains that overlap tr. Bounds are
// inclusive, so a domain that only touches tr goes too.
func (c *OffsetCache) invalidate(ch channel.Key, tr telem.TimeRange) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for start, el := range c.mu.tables[ch] {
		if start <= tr.End && el.Value.(*cachedTable).table.end >= tr.Start {
			c.remove(el)
		}
	}
}

// purge removes every table of the channel.
func (c *OffsetCache) purge(ch channel.Key) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, el := range c.mu.tables[ch] {
		c.remove(el)
	}
}

// remove requires c.mu to be held.
func (c *OffsetCache) remove(el *list.Element) {
	entry := c.mu.recent.Remove(el).(*cachedTable)
	tables := c.mu.tables[entry.channel]
	delete(tables, entry.start)
	if len(tables) == 0 {
		delete(c.mu.tables, entry.channel)
	}
	c.mu.size -= entry.table.size()
}

func buildOffsetTable(
	ins alamos.Instrumentation,
	r *domain.Reader,
	domainSize telem.Size,
	tr telem.TimeRange,
) (*offsetTable, error) {
	var (
		t       = &offsetTable{end: tr.End}
		bufSize = min(scanBufferSize, domainSize)
		br      = bufio.NewReaderSize(
			io.NewSectionReader(r, 0, int64(domainSize)),
			int(bufSize),
		)
		lenBuf = make([]byte, 4)
		pos    int64
	)
	for pos+4 <= int64(domainSize) {
		// The loop guard guarantees the SectionReader still has 4 bytes of
		// advertised range left, so a short read here means the on-disk file is
		// shorter than the domain pointer's recorded size — i.e. truncation or
		// corruption. Surface it as a warning, then return the recoverable
		// prefix so reads of the intact head still serve correct offsets;
		// queries past the truncation point fall through to iter.Size() and a
		// short sampleCount, which is preferable to failing the whole table
		// rebuild and breaking every read on this domain.
		if _, err := io.ReadFull(br, lenBuf); err != nil {
			if errors.IsAny(err, io.EOF, io.ErrUnexpectedEOF) {
				logTruncation(ins, tr, domainSize, pos, err)
				break
			}
			return nil, err
		}
		t.offsets = append(t.offsets, uint32(pos))
		length := int64(telem.ByteOrder.Uint32(lenBuf))
		if length > 0 {
			// Discard signals "fewer bytes than requested" with io.EOF (not
			// ErrUnexpectedEOF, per bufio's contract), so the EOF-only check
			// here mirrors the ReadFull branch above for the same truncation
			// scenario.
			if _, err := br.Discard(int(length)); err != nil {
				if errors.Is(err, io.EOF) {
					logTruncation(ins, tr, domainSize, pos, err)
					break
				}
				return nil, err
			}
		}
		pos += 4 + length
		t.sampleCount++
	}
	return t, nil
}

// videoScanSize is how much of each frame buildVideoTable reads: the 4-byte length
// and the 2-byte header.
const videoScanSize = 6

// buildVideoTable builds the table of a video domain like buildOffsetTable, but it
// reads only the length and header of each frame and skips its data.
func buildVideoTable(
	ins alamos.Instrumentation,
	r *domain.Reader,
	domainSize telem.Size,
	tr telem.TimeRange,
) (*offsetTable, error) {
	var (
		t      = &offsetTable{end: tr.End}
		header = make([]byte, videoScanSize)
		pos    int64
	)
	for pos+videoScanSize <= int64(domainSize) {
		if _, err := r.ReadAt(header, pos); err != nil {
			if errors.IsAny(err, io.EOF, io.ErrUnexpectedEOF) {
				logTruncation(ins, tr, domainSize, pos, err)
				break
			}
			return nil, err
		}
		length := int64(telem.ByteOrder.Uint32(header))
		if telem.IsVideoKeyframe(header[4:]) {
			t.keyframes = append(t.keyframes, uint32(t.sampleCount))
		}
		t.offsets = append(t.offsets, uint32(pos))
		pos += 4 + length
		t.sampleCount++
	}
	return t, nil
}

// logTruncation records a warning when the on-disk file underlying a domain
// pointer is shorter than the pointer's recorded size, surfacing a class of
// corruption that buildOffsetTable would otherwise silently absorb by serving
// a partial offset table.
func logTruncation(
	ins alamos.Instrumentation,
	tr telem.TimeRange,
	domainSize telem.Size,
	pos int64,
	cause error,
) {
	ins.L.Error(
		"variable-density domain truncated below recorded size",
		zap.Stringer("range", tr),
		zap.Int64("recorded_size", int64(domainSize)),
		zap.Int64("recovered_bytes", pos),
		zap.Error(cause),
	)
}

// offsetResolver translates sample indices to byte offsets within domain files.
// Fixed-density channels have no cache and rely on density arithmetic;
// variable-length channels keep per-domain offset tables in a cache, built on
// first access by scanning the length-prefixed records.
type offsetResolver struct {
	// ins is the resolver's instrumentation handle. Used by tableFor's scan
	// path to surface on-disk truncation warnings; otherwise unused.
	ins     alamos.Instrumentation
	density telem.Density
	channel channel.Key
	cache   *OffsetCache // nil for fixed-density channels
	// video is true when the channel's tables hold keyframes.
	video bool
	// rebuilds collapses concurrent rebuilds of the same domain into a single
	// scan. Callers that miss on the same start timestamp wait for the leader
	// to finish and share its result rather than each opening their own reader
	// and walking the same length prefixes. Unused for fixed-density channels.
	rebuilds singleflight.Group
}

func newOffsetResolver(
	ch channel.Channel,
	cache *OffsetCache,
	ins alamos.Instrumentation,
) *offsetResolver {
	if ch.DataType.IsVariable() {
		return &offsetResolver{
			ins:     ins,
			channel: ch.Key,
			cache:   cache,
			video:   ch.DataType == telem.VideoT,
		}
	}
	return &offsetResolver{ins: ins, density: ch.DataType.Density()}
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
		total := r.density.SampleCount(iter.Size())
		if sampleIdx >= total {
			return iter.Size(), nil
		}
		return r.density.Size(sampleIdx), nil
	}
	t, err := r.tableFor(ctx, iter)
	if err != nil {
		return 0, err
	}
	if sampleIdx >= t.sampleCount {
		return iter.Size(), nil
	}
	return t.byteOffsetAt(sampleIdx), nil
}

func (r *offsetResolver) domainSampleCount(
	ctx context.Context,
	iter *domain.Iterator,
) (int64, error) {
	if r.cache == nil {
		return r.density.SampleCount(iter.Size()), nil
	}
	t, err := r.tableFor(ctx, iter)
	if err != nil {
		return 0, err
	}
	return t.sampleCount, nil
}

// invalidate drops the tables of the domains that a delete of tr changed.
func (r *offsetResolver) invalidate(tr telem.TimeRange) {
	if r.cache != nil {
		r.cache.invalidate(r.channel, tr)
	}
}

// purge drops every table of the channel.
func (r *offsetResolver) purge() {
	if r.cache != nil {
		r.cache.purge(r.channel)
	}
}

func (r *offsetResolver) tableFor(
	ctx context.Context,
	iter *domain.Iterator,
) (*offsetTable, error) {
	tr := iter.TimeRange()
	// A pointer's Start is immutable; its End advances on commit. Match End
	// exactly: a stale-behind cache (cached.end < tr.End) misses tail samples,
	// and a stale-ahead cache (cached.end > tr.End, published by a concurrent
	// commit) inflates sampleCount and yields offsets past iter.Size().
	if cached, ok := r.cache.get(r.channel, tr.Start); ok && cached.end == tr.End {
		return cached, nil
	}
	// Concurrent callers missing on the same (Start, End) collapse into one scan
	// via singleflight; the leader rebuilds and the waiters share its result. End
	// is part of the key because the leader's closure runs against the leader's
	// iter, which carries a frozen (Start, End, size) snapshot of the pointer at
	// reload time. A follower that joined with a newer End would otherwise receive
	// a table truncated to the leader's older view, and byteOffset / sampleCount
	// queries against the missing tail samples would silently fall through to the
	// iter.Size() / zero-sampleCount fallback paths and lose data.
	key := strconv.FormatInt(
		int64(tr.Start),
		10,
	) + ":" + strconv.FormatInt(
		int64(tr.End),
		10,
	)
	result, err, _ := r.rebuilds.Do(key, func() (any, error) {
		// Re-check the cache: the leader of a prior call may have populated it
		// while this goroutine was waiting on the singleflight entry.
		if cached, ok := r.cache.get(r.channel, tr.Start); ok && cached.end == tr.End {
			return cached, nil
		}
		rd, err := iter.OpenReader(ctx)
		if err != nil {
			return nil, err
		}
		defer func() { err = errors.Combine(err, rd.Close()) }()
		build := buildOffsetTable
		if r.video {
			build = buildVideoTable
		}
		t, err := build(r.ins, rd, iter.Size(), tr)
		if err != nil {
			return nil, err
		}
		r.cache.set(r.channel, tr.Start, t)
		return t, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*offsetTable), nil
}

// newTracker returns an offsetTracker bound to this resolver. Fixed-density trackers
// carry the resolver's density and use it to convert per-write byte counts to sample
// counts; cache-backed trackers also track per-sample offsets within the current domain
// so they can publish them to the cache on commit.
func (r *offsetResolver) newTracker() *offsetTracker {
	t := &offsetTracker{resolver: r}
	if r.cache == nil {
		t.density = r.density
	}
	return t
}

// offsetTracker tracks per-domain offset state for a unary writer chain. The tracker
// is owned by the controlledWriter, so all Writers that share that resource (i.e. that
// take control of the same region across handoffs) observe the same tracker state.
// Fixed-density and variable-length channels share the cumulative-count behavior at
// the alignment math site by deriving it from the underlying domain.Writer for
// fixed-density and from the tracker's running counter for variable-length, both of
// which advance monotonically across handoff. Variable-length channels additionally
// track per-domain byte offsets so they can publish them to the offset cache on
// commit.
type offsetTracker struct {
	// resolver is the offsetResolver this tracker was created from. Used by publish to
	// install committed offsets into the resolver's cache.
	resolver *offsetResolver
	// density is the sample density of the channel for fixed-density trackers; zero
	// for variable-length trackers.
	density telem.Density
	// currentStart is the start timestamp of the domain currently being tracked. Set
	// when the first domain starts, then updated to commitEnd on rollover so subsequent
	// publishes target the new domain. Unused for fixed-density.
	currentStart telem.TimeStamp
	// domainBytes is the byte count written to the current domain since the last
	// rollover. Used as the base offset for the next write so per-sample offsets are
	// relative to the start of the current file. Reset on rollover. Unused for
	// fixed-density.
	domainBytes int64
	// domainOffsets is the running list of per-sample byte offsets within the current
	// domain for variable-length trackers; nil for fixed-density. Reset on rollover.
	domainOffsets []uint32
	// domainKeyframes holds the indices of the keyframes in the current domain of a
	// video channel. Reset on rollover.
	domainKeyframes []uint32
	// sessionSamples is the cumulative sample count for variable-length trackers
	// across every domain in the controlled resource's lifetime. Used for alignment
	// math and the end-timestamp lookup at commit time. Survives control handoff
	// because the tracker is shared across Writers via controlledWriter. Unused for
	// fixed-density (count derives from dw.Len() instead).
	sessionSamples int64
}

// count returns the cumulative sample count visible to the current writer. For
// fixed-density trackers it derives from dw.Len(); for variable-length trackers it
// returns the tracker's running session count. Both transfer across control handoff
// because the underlying domain.Writer and the offsetTracker are shared by every
// Writer that controls the same region.
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
		length := int(telem.ByteOrder.Uint32(data[offset:]))
		if t.resolver.video && telem.IsVideoKeyframe(data[offset+4:]) {
			t.domainKeyframes = append(
				t.domainKeyframes,
				uint32(len(t.domainOffsets)),
			)
		}
		t.domainOffsets = append(t.domainOffsets, baseByteOffset+uint32(offset))
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
	t.resolver.cache.set(t.resolver.channel, t.currentStart, &offsetTable{
		end:         end,
		offsets:     slices.Clone(t.domainOffsets),
		sampleCount: int64(len(t.domainOffsets)),
		keyframes:   slices.Clone(t.domainKeyframes),
	})
}

// reset clears the tracker for a new session of samples.
func (t *offsetTracker) reset() {
	t.domainBytes = 0
	t.domainOffsets = nil
	t.domainKeyframes = nil
	t.sessionSamples = 0
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
	t.domainKeyframes = nil
}
