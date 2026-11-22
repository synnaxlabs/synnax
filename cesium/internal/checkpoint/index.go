// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint

import (
	"bytes"
	"cmp"
	"os"
	"slices"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
)

// defaultPartSize is the size of the sealed segment records in a head at which a switch
// moves them to a part.
const defaultPartSize = 8 << 20

// Index holds the parts that a head names, with the segment records of each. A switch
// finds the dead and cut records of each part against it.
type Index struct {
	parts []indexed
	// ranges are the part ranges of the head that x holds the parts of, once its switch
	// sets them.
	ranges []state.Range
	// limit is the part identity limit.
	limit uint64
}

// indexed is a part with its segment records, in record order.
type indexed struct {
	Part
	keys []uint32
	segs []state.Segment
}

// LoadParts reads each part that h, a head of database db, names in the root fs, and
// returns their index. It returns an error that wraps place.ErrCorrupt when a part is
// missing or fails its checks.
func LoadParts(fs xfs.FS, db uint64, h Header) (*Index, error) {
	x := &Index{limit: h.Limits.Parts}
	for _, p := range h.Parts {
		channels, err := readPart(fs, db, p)
		if errors.Is(err, os.ErrNotExist) {
			err = errors.Wrapf(
				place.ErrCorrupt,
				"part %d that head %d names is missing",
				p.ID,
				h.Generation,
			)
		}
		if err != nil {
			return nil, err
		}
		x.parts = append(x.parts, index(p, channels))
	}
	return x, nil
}

func index(p Part, channels []*state.Channel) indexed {
	ip := indexed{Part: p}
	for _, c := range channels {
		for _, seg := range c.Segments {
			ip.keys = append(ip.keys, c.Key)
			ip.segs = append(ip.segs, seg)
		}
	}
	return ip
}

// rangesAfter returns the part range of each key that live records of the parts of x
// hold, in rising order of key. It returns the ranges of last when x has its parts with
// the same dead marks, since the ranges depend on those alone.
func (x *Index) rangesAfter(last *Index) []state.Range {
	if last.ranges != nil && len(x.parts) == len(last.parts) &&
		slices.EqualFunc(x.parts, last.parts, func(a, b indexed) bool {
			return a.ID == b.ID && bytes.Equal(a.Dead, b.Dead)
		}) {
		return last.ranges
	}
	return x.liveRanges()
}

// liveRanges returns the part range of each key that live records of the parts of x
// hold, in rising order of key.
func (x *Index) liveRanges() []state.Range {
	// held holds one entry per key of each part with a live record, as a range of that
	// part alone. The parts are in rising order of identity, and so are their keys.
	var n int
	for _, p := range x.parts {
		for i, key := range p.keys {
			if i == 0 || key != p.keys[i-1] {
				n++
			}
		}
	}
	held := make([]state.Range, 0, n)
	for _, p := range x.parts {
		for i, key := range p.keys {
			if p.dead(i) || (len(held) > 0 && held[len(held)-1].Key == key &&
				held[len(held)-1].Oldest == p.ID) {
				continue
			}
			held = append(held, state.Range{Key: key, Oldest: p.ID, Newest: p.ID})
		}
	}
	slices.SortFunc(held, func(a, b state.Range) int {
		return cmp.Or(cmp.Compare(a.Key, b.Key), cmp.Compare(a.Oldest, b.Oldest))
	})
	ranges := held[:0]
	for _, r := range held {
		if len(ranges) > 0 && ranges[len(ranges)-1].Key == r.Key {
			ranges[len(ranges)-1].Newest = r.Oldest
			continue
		}
		ranges = append(ranges, r)
	}
	return ranges
}

// Parted returns what the parts of x hold for the state: the parts, the tags of their
// records, and the part range of each key that their records hold.
func (x *Index) Parted() ([]Part, []state.Tag, []state.Range) {
	parts := make([]Part, len(x.parts))
	var tags []state.Tag
	for i, p := range x.parts {
		parts[i] = p.Part
		for _, seg := range p.segs {
			tags = append(tags, state.Tag{Part: p.ID, Segment: seg})
		}
	}
	return parts, tags, x.ranges
}

// Live returns the live records of key in the parts of x, by segment identity.
func (x *Index) Live(key uint32) map[uint64]state.Segment {
	live := make(map[uint64]state.Segment)
	for _, p := range x.parts {
		i, _ := slices.BinarySearch(p.keys, key)
		for ; i < len(p.keys) && p.keys[i] == key; i++ {
			if !p.dead(i) {
				live[p.segs[i].ID] = p.segs[i]
			}
		}
	}
	return live
}

// fold marks each record of p dead that a kill of the generation of p or later names.
func (p *indexed) fold(kills map[uint32][]state.Kill) {
	if len(kills) == 0 {
		return
	}
	for i, key := range p.keys {
		for _, k := range kills[key] {
			if k.Generation >= p.Generation && (k.All || k.Segment == p.segs[i].ID) {
				p.kill(i)
			}
		}
	}
}

// ids returns the identities of the parts of x.
func (x *Index) ids() set.Set[uint64] {
	ids := set.New[uint64]()
	for _, p := range x.parts {
		ids.Add(p.ID)
	}
	return ids
}

func (p *indexed) dead(i int) bool { return !p.Live(i) }

func (p *indexed) kill(i int) {
	if len(p.Dead) == 0 {
		p.Dead = make([]byte, (len(p.segs)+7)/8)
	}
	p.Dead[i/8] |= 1 << (i % 8)
}

// deadCount returns the number of dead segment records of p.
func (p *indexed) deadCount() int {
	n := 0
	for i := range p.segs {
		if p.dead(i) {
			n++
		}
	}
	return n
}
