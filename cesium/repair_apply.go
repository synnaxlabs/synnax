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
	"cmp"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// apply puts p into the state of e at the damage d: limits above the values of all,
// the entries after the damage, and of the files, the placed segments, a commit with a
// fresh alignment for each clock write, and the end of the gap of each segment, at the
// records of l, the entries up to the next bad frame. It moves the ranges of a clock
// write whose commit breaks a rule from rebuilt to unplaced, with the rule.
func (e *engine) apply(d Damage, l, all *later, placed []placement, p *plan) error {
	files, err := e.fileLimit()
	if err != nil {
		return err
	}
	rb := state.Rebuild{
		Sequence: d.LastSequence,
		Limits: state.Limits{
			Segments:   max(all.limits.Segments, files),
			Alignments: all.limits.Alignments,
		},
	}
	created := make(map[uint64]ChannelKey, len(placed))
	for _, pl := range placed {
		rb.Creates = append(rb.Creates, log.Create{
			Segment:  pl.id,
			Key:      pl.key,
			Clock:    pl.cfg.Clock,
			Version:  segmentVersion,
			DataType: pl.cfg.DataType,
		})
		created[pl.id] = pl.key
	}
	var pre []state.Preload
	for _, g := range p.groups {
		c, err := e.commitOf(g)
		if err != nil {
			return err
		}
		rb.Commits = append(rb.Commits, c)
		if pre, err = e.commitPreloads(pre, &c, created); err != nil {
			return err
		}
	}
	slices.SortFunc(
		pre,
		func(a, b state.Preload) int { return cmp.Compare(a.Key, b.Key) },
	)
	for _, s := range p.spans {
		end, ok, err := e.gapEnd(l, s)
		if err != nil {
			return err
		}
		if ok {
			rb.Ends = append(rb.Ends, end)
		}
	}
	errs, err := e.state.Rebuild(rb, pre...)
	if err != nil {
		return err
	}
	for i, err := range errs {
		if err != nil {
			p.refuse(p.groups[i], err)
		}
	}
	return nil
}

// refuse moves the ranges of g from the rebuilt ranges of p to its unplaced ranges,
// with the rule that err names.
func (p *plan) refuse(g group, err error) {
	for _, s := range append([]span{g.clock}, g.data...) {
		i := slices.IndexFunc(p.rebuilt, func(r Range) bool {
			return r.Key == s.key && r.Segment == s.cfg.ID
		})
		r := p.rebuilt[i]
		r.TimeRange, r.Reason = telem.TimeRange{}, err.Error()
		p.rebuilt = slices.Delete(p.rebuilt, i, i+1)
		p.unplaced = append(p.unplaced, r)
	}
	slices.SortFunc(p.unplaced, func(a, b Range) int {
		return cmp.Or(cmp.Compare(a.Key, b.Key), cmp.Compare(a.Segment, b.Segment))
	})
}

// commitOf returns the commit of the clock write g, whose slots end at the ends of g.
func (e *engine) commitOf(g group) (log.Commit, error) {
	c := log.Commit{
		Samples: uint32(g.samples),
		First:   int64(g.timeRange.Start),
		Last:    int64(g.timeRange.End - 1),
	}
	for i, s := range append([]span{g.clock}, g.data...) {
		crc, err := e.tailCRC(s, g.ends[i])
		if err != nil {
			return log.Commit{}, err
		}
		c.Slots = append(c.Slots, log.Slot{
			Segment: s.cfg.ID,
			Length:  uint32(g.ends[i]),
			TailCRC: crc,
		})
	}
	slices.SortFunc(c.Slots, func(a, b log.Slot) int {
		return cmp.Compare(a.Segment, b.Segment)
	})
	return c, nil
}

// gapEnd returns the length and the tail checksum of the segment of s at the end of its
// gap, and false for a span that no later record ends, or that a later record ends
// below its start. A span whose samples do not end at its later Commit ends at the
// length of that Commit, so the Commit breaks a rule.
func (e *engine) gapEnd(l *later, s span) (log.Slot, bool, error) {
	switch {
	case s.reason == walkReason:
		length, crc, _ := l.end(s.cfg.ID)
		return log.Slot{Segment: s.cfg.ID, Length: uint32(length), TailCRC: crc}, true,
			nil
	case s.reason == lengthReason || !s.closed:
		return log.Slot{}, false, nil
	}
	crc, err := e.tailCRC(s, s.bound)
	return log.Slot{Segment: s.cfg.ID, Length: uint32(s.bound), TailCRC: crc}, true, err
}

// tailCRC returns the running checksum of the segment of s at stream length length.
func (e *engine) tailCRC(s span, length int64) (crc uint32, err error) {
	f, err := e.fs.Open(segmentName(s.key, s.cfg.ID, 0), os.O_RDONLY)
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	return s.cfg.TailCRC(f, length)
}

// fileLimit returns one above the identity in the name of each segment file of a
// channel directory, under any suffix.
func (e *engine) fileLimit() (uint64, error) {
	keys, err := e.keyDirs()
	if err != nil {
		return 0, err
	}
	var limit uint64
	for _, key := range keys {
		infos, err := e.fs.List(strconv.FormatUint(uint64(key), 10))
		if err != nil {
			return 0, err
		}
		for _, info := range infos {
			hex, _, ok := strings.Cut(info.Name(), ".seg")
			if id, err := strconv.ParseUint(hex, 16, 64); ok && err == nil {
				limit = max(limit, id+1)
			}
		}
	}
	return limit, nil
}

// replayEach applies the records of the entry of sequence after the damage one at a
// time, and lists in rep those that break a rule. A Replace whose old segment has no
// file takes that segment out of the state, and its copy takes the pointers of the
// Replace when its blocks pass, or its bytes are unplaced (D-370).
func (e *engine) replayEach(sequence uint64, records []log.Record, rep *Report) error {
	pre, err := e.preloads(records)
	if err != nil {
		return err
	}
	gone, pre, err := e.markGone(records, pre)
	if err != nil {
		return err
	}
	skipped, err := e.state.ReplayEach(sequence, records, pre...)
	if err != nil {
		return err
	}
	refused := make(map[log.Record]error, len(skipped))
	for _, s := range skipped {
		r := recordOf(sequence, s.Record)
		r.Reason = s.Err.Error()
		rep.Skipped = append(rep.Skipped, r)
		refused[s.Record] = s.Err
	}
	for _, rec := range records {
		r, ok := rec.(*log.Replace)
		if !ok {
			continue
		}
		key, ok := gone[r.Old]
		if !ok {
			continue
		}
		if err, ok := refused[rec]; ok {
			rep.Unplaced = append(rep.Unplaced, Range{
				Key:     key,
				Segment: r.New,
				Bytes:   telem.Size(r.Length),
				Samples: pointerSamples(r.Pointers),
				Reason:  "the copy of a Replace whose old file is gone: " + err.Error(),
			})
			continue
		}
		rep.Removed = append(rep.Removed, Segment{Key: key, Segment: r.Old})
	}
	return nil
}

// markGone marks gone in the state of e the old segment of each Replace of records
// whose file is gone and whose copy passes its checks, and adds the preload of its key
// to pre. It returns the key of the old segment of each Replace whose file is gone.
func (e *engine) markGone(
	records []log.Record,
	pre []state.Preload,
) (map[uint64]ChannelKey, []state.Preload, error) {
	gone := make(map[uint64]ChannelKey)
	for _, rec := range records {
		r, ok := rec.(*log.Replace)
		if !ok {
			continue
		}
		key, passes, missing, err := e.goneReplace(r)
		if err != nil {
			return nil, nil, err
		}
		if !missing {
			continue
		}
		gone[r.Old] = key
		if passes {
			e.state.Gone(r.Old)
			first, last := pointerSpan(r.Pointers)
			if pre, err = e.addPreload(pre, key, first, last, false); err != nil {
				return nil, nil, err
			}
		}
	}
	slices.SortFunc(
		pre,
		func(a, b state.Preload) int { return cmp.Compare(a.Key, b.Key) },
	)
	return gone, pre, nil
}

// goneReplace returns the key of the old segment of r, whether the state holds it
// while its file is gone, and whether the blocks of the copy pass for the length and
// the tail checksum of r.
func (e *engine) goneReplace(
	r *log.Replace,
) (key ChannelKey, passes, missing bool, err error) {
	key, held := e.state.KeyOf(r.Old)
	if !held {
		return 0, false, false, nil
	}
	i := slices.IndexFunc(e.state.ReplaySegments(key), func(s state.Segment) bool {
		return s.ID == r.Old
	})
	old := e.state.ReplaySegments(key)[i]
	exists, err := e.segmentExists(segmentName(key, old.ID, old.File), old.File)
	if err != nil || exists {
		return key, false, false, err
	}
	cfg := e.segmentConfig(old)
	cfg.ID, cfg.Bare = r.New, false
	f, err := e.fs.Open(segmentName(key, r.New, 0), os.O_RDONLY)
	if errors.Is(err, os.ErrNotExist) {
		return key, false, true, nil
	}
	if err != nil {
		return key, false, true, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	end, err := checkedEnd(f, cfg, 0, int64(r.Length), r.TailCRC, true)
	return key, end == int64(r.Length), true, err
}

// pointerSpan returns the first and the last stamp of ptrs.
func pointerSpan(ptrs []log.Pointer) (first, last telem.TimeStamp) {
	first, last = telem.TimeStampMax, telem.TimeStampMin
	for _, p := range ptrs {
		first = min(first, telem.TimeStamp(p.First))
		last = max(last, telem.TimeStamp(p.Last))
	}
	return first, last
}

// pointerSamples returns the number of samples of ptrs.
func pointerSamples(ptrs []log.Pointer) int64 {
	var n int64
	for _, p := range ptrs {
		n += int64(p.Samples)
	}
	return n
}

// removeMissing takes out of the state of e each segment whose file is gone when a
// lost record explains it: a sealed segment, one that a later Replace of l retires
// after a lost Seal, or any segment of a key that has no directory. It returns them by
// key and segment. An open segment whose file is gone while its key has a directory
// fails it with an error that wraps os.ErrNotExist.
func (e *engine) removeMissing(l *later) ([]Segment, error) {
	var out []Segment
	for _, key := range e.state.ReplayKeys() {
		kept, err := e.fs.Exists(strconv.FormatUint(uint64(key), 10))
		if err != nil {
			return nil, err
		}
		held := e.state.ReplaySegments(key)
		pre, err := e.records(key, telem.TimeStampMin, telem.TimeStampMax)
		if err != nil {
			return nil, err
		}
		check := func(seg state.Segment, parted bool) error {
			name := segmentName(key, seg.ID, seg.File)
			ok, err := e.segmentExists(name, seg.File)
			switch {
			case err != nil || ok:
				return err
			case kept && !seg.Sealed && !l.replaced.Contains(seg.ID):
				return errors.Wrapf(os.ErrNotExist, "file %s of open segment %d", name,
					seg.ID)
			}
			out = append(out, Segment{Key: key, Segment: seg.ID})
			return e.state.Remove(key, seg.ID, parted)
		}
		for _, seg := range held {
			if err = check(seg, false); err != nil {
				return nil, err
			}
		}
		for _, seg := range pre.Segments {
			if slices.ContainsFunc(held, func(s state.Segment) bool {
				return s.ID == seg.ID
			}) {
				continue
			}
			if err = check(seg, true); err != nil {
				return nil, err
			}
		}
	}
	slices.SortFunc(out, func(a, b Segment) int {
		return cmp.Or(cmp.Compare(a.Key, b.Key), cmp.Compare(a.Segment, b.Segment))
	})
	return out, nil
}

// segmentExists reports whether the file name of a segment whose old file is n exists,
// under the name of an old compaction too for a version 0 segment.
func (e *engine) segmentExists(name string, n uint16) (bool, error) {
	ok, err := e.fs.Exists(name)
	if err != nil || ok || n == 0 {
		return ok, err
	}
	return e.fs.Exists(name + tempSuffix)
}
