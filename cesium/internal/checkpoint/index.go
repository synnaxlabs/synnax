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

// Merge adds the segments of the parts of x that are not dead to s, the snapshot of the
// head that names them.
func (x *Index) Merge(s *state.Snapshot) {
	if len(x.parts) == 0 {
		return
	}
	byKey := make(map[uint32]*state.Channel, len(s.Channels))
	for _, c := range s.Channels {
		byKey[c.Key] = c
	}
	for _, p := range x.parts {
		for i, seg := range p.segs {
			if p.dead(i) {
				continue
			}
			c, ok := byKey[p.keys[i]]
			if !ok {
				c = &state.Channel{Key: p.keys[i]}
				byKey[c.Key] = c
				s.Channels = append(s.Channels, c)
			}
			c.Segments = append(c.Segments, seg)
		}
	}
	slices.SortFunc(s.Channels, func(a, b *state.Channel) int {
		return cmp.Compare(a.Key, b.Key)
	})
	for _, c := range s.Channels {
		slices.SortFunc(c.Segments, func(a, b state.Segment) int {
			return cmp.Compare(a.ID, b.ID)
		})
	}
}

// Tags returns the tags of the records of the parts of x that are not dead.
func (x *Index) Tags() []state.Tag {
	var tags []state.Tag
	for _, p := range x.parts {
		for i, seg := range p.segs {
			if !p.dead(i) {
				tags = append(tags, state.Tag{Part: p.ID, Segment: seg})
			}
		}
	}
	return tags
}

// ids returns the identities of the parts of x.
func (x *Index) ids() set.Set[uint64] {
	ids := set.New[uint64]()
	for _, p := range x.parts {
		ids.Add(p.ID)
	}
	return ids
}

func (p *indexed) dead(i int) bool {
	return i/8 < len(p.Dead) && p.Dead[i/8]&(1<<(i%8)) != 0
}

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
