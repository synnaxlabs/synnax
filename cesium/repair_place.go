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
	"math"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// unnamedReason is the reason of each segment file that no record names.
const unnamedReason = "no record names it"

// later is what a repair reads from the valid frames after the damage.
type later struct {
	// created maps each segment that a Create makes to its clock.
	created map[uint64]uint32
	// named maps each other segment that a record names to the first Commit that names
	// it, or to nil when none does.
	named map[uint64]*log.Commit
	// sealed maps each segment that a Seal seals to the first Seal of it.
	sealed map[uint64]*log.Seal
	// dropped holds the keys that a Drop drops.
	dropped set.Set[ChannelKey]
}

func newLater() *later {
	return &later{
		created: make(map[uint64]uint32),
		named:   make(map[uint64]*log.Commit),
		sealed:  make(map[uint64]*log.Seal),
		dropped: set.New[ChannelKey](),
	}
}

// add reads the records of an entry after the damage.
func (l *later) add(_ uint64, records []log.Record) error {
	for _, rec := range records {
		switch rec := rec.(type) {
		case *log.Create:
			l.created[rec.Segment] = rec.Clock
		case *log.Commit:
			for _, s := range rec.Slots {
				if l.named[s.Segment] == nil {
					l.named[s.Segment] = rec
				}
			}
		case *log.Seal:
			l.name(rec.Segment)
			if l.sealed[rec.Segment] == nil {
				l.sealed[rec.Segment] = rec
			}
		case *log.Tombstone:
			for _, c := range []*log.Cut{rec.Left, rec.Right} {
				if c != nil {
					l.name(c.Segment)
				}
			}
		case *log.Replace:
			l.name(rec.Old)
			l.name(rec.New)
		case *log.Check:
			l.name(rec.Segment)
		case *log.Drop:
			l.dropped.Add(rec.Key)
		}
	}
	return nil
}

func (l *later) name(segment uint64) {
	if _, ok := l.named[segment]; !ok {
		l.named[segment] = nil
	}
}

// orphan is a segment file whose segment the state lacks.
type orphan struct {
	key  ChannelKey
	id   uint64
	size int64
}

func (o orphan) name() string { return segmentName(o.key, o.id, 0) }

// placement is an orphan that the place search places.
type placement struct {
	orphan
	cfg segment.Config
	// checked is the number of full blocks from the start of the file that pass.
	checked int64
}

// place searches the type and clock of each orphan that a record after the damage
// names, and returns those that it places. It lists as unrecovered each one that it
// cannot place, and each other orphan, unless the state left the segment or a Drop
// after the damage drops its key: the next open removes those files.
func (e *engine) place(r *recovery, l *later) ([]placement, []File, error) {
	named, unrecovered, err := e.unheld(r, l)
	if err != nil {
		return nil, nil, err
	}
	keys := set.New[ChannelKey]()
	for _, o := range named {
		keys.Add(o.key)
	}
	var placed []placement
	for _, o := range named {
		p, reason, err := e.search(r, l, o, keys)
		if err != nil {
			return nil, nil, err
		}
		if reason != "" {
			unrecovered = append(unrecovered, File{
				Key:     o.key,
				Segment: o.id,
				Bytes:   telem.Size(o.size),
				Reason:  reason,
			})
			continue
		}
		placed = append(placed, p)
	}
	slices.SortFunc(unrecovered, func(a, b File) int {
		return cmp.Or(cmp.Compare(a.Key, b.Key), cmp.Compare(a.Segment, b.Segment))
	})
	return placed, unrecovered, nil
}

// unheld returns, in order of key and identity, the segment files whose segments the
// state lacks, no part of the head holds, and no Create after the damage makes, that a
// record after the damage names. It lists the rest that the state never left as
// unrecovered.
func (e *engine) unheld(r *recovery, l *later) ([]orphan, []File, error) {
	dead := set.New[uint64]()
	for _, d := range e.state.Dead() {
		dead.Add(d.Segment)
	}
	keys, err := e.keyDirs()
	if err != nil {
		return nil, nil, err
	}
	var (
		named       []orphan
		unrecovered []File
	)
	for _, key := range keys {
		infos, err := e.fs.List(strconv.FormatUint(uint64(key), 10))
		if err != nil {
			return nil, nil, err
		}
		live := r.parts.Live(key)
		for _, info := range infos {
			hex, ok := strings.CutSuffix(info.Name(), ".seg")
			id, err := strconv.ParseUint(hex, 16, 64)
			if !ok || err != nil {
				continue
			}
			_, held := e.state.KeyOf(id)
			_, parted := live[id]
			_, created := l.created[id]
			if held || parted || created || dead.Contains(id) {
				continue
			}
			o := orphan{key: key, id: id, size: info.Size()}
			if _, ok := l.named[id]; ok {
				named = append(named, o)
			} else if !l.dropped.Contains(o.key) {
				unrecovered = append(unrecovered, File{
					Key:     o.key,
					Segment: id,
					Bytes:   telem.Size(o.size),
					Reason:  unnamedReason,
				})
			}
		}
	}
	slices.SortFunc(named, func(a, b orphan) int {
		return cmp.Or(cmp.Compare(a.key, b.key), cmp.Compare(a.id, b.id))
	})
	return named, unrecovered, nil
}

// search searches the type and clock of o among every type code and the clocks of
// clocks: the clocks of the segments of its key, each key of keys, and the clock of
// the first Commit after the damage that names o. It returns why it places nothing, or
// an empty reason.
func (e *engine) search(
	r *recovery,
	l *later,
	o orphan,
	keys set.Set[ChannelKey],
) (placement, string, error) {
	clocks := set.New[uint32]()
	for k := range keys {
		clocks.Add(k)
	}
	for _, seg := range e.state.ReplaySegments(o.key) {
		clocks.Add(seg.Clock)
	}
	for _, seg := range r.parts.Live(o.key) {
		clocks.Add(seg.Clock)
	}
	if c := l.named[o.id]; c != nil {
		for _, s := range c.Slots {
			if clock, ok := e.clockOf(l, s.Segment); ok {
				clocks.Add(clock)
			}
		}
	}
	var candidates []segment.Config
	for code := range telem.DataTypeCode(math.MaxUint8) {
		dt := code.DataType()
		if dt == telem.UnknownT {
			continue
		}
		for clock := range clocks {
			candidates = append(candidates, segment.Config{
				DB:       r.db,
				ID:       o.id,
				DataType: uint8(code),
				Clock:    clock,
				Variable: dt.IsVariable(),
			})
		}
	}
	f, err := e.fs.Open(o.name(), os.O_RDONLY)
	if err != nil {
		return placement{}, "", err
	}
	cfg, checked, reason, err := segment.Search(f, o.size, candidates)
	if err = errors.Join(err, f.Close()); err != nil {
		return placement{}, "", err
	}
	return placement{orphan: o, cfg: cfg, checked: checked}, reason, nil
}

// clockOf returns the clock of segment, which the state holds or a Create after the
// damage makes, and false when neither.
func (e *engine) clockOf(l *later, segment uint64) (uint32, bool) {
	if clock, ok := l.created[segment]; ok {
		return clock, true
	}
	key, ok := e.state.KeyOf(segment)
	if !ok {
		return 0, false
	}
	for _, seg := range e.state.ReplaySegments(key) {
		if seg.ID == segment {
			return seg.Clock, true
		}
	}
	return 0, false
}
