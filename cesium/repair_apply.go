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

// apply puts p into the state of e at the damage d: limits above the values of l and
// of the files, the placed segments, a commit with a fresh alignment for each clock
// write, and the end of the gap of each segment. It moves the ranges of a clock write
// whose commit breaks a rule from rebuilt to unplaced, with the rule.
func (e *engine) apply(d Damage, l *later, placed []placement, p *plan) error {
	files, err := e.fileLimit()
	if err != nil {
		return err
	}
	rb := state.Rebuild{
		Sequence: d.LastSequence,
		Limits: state.Limits{
			Segments:   max(l.limits.Segments, files),
			Alignments: l.limits.Alignments,
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
// time, and returns those that break a rule.
func (e *engine) replayEach(sequence uint64, records []log.Record) ([]Record, error) {
	pre, err := e.preloads(records)
	if err != nil {
		return nil, err
	}
	skipped, err := e.state.ReplayEach(sequence, records, pre...)
	if err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(skipped))
	for _, s := range skipped {
		r := recordOf(sequence, s.Record)
		r.Reason = s.Err.Error()
		out = append(out, r)
	}
	return out, nil
}
