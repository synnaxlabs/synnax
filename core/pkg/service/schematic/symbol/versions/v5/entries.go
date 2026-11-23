// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"maps"
	"slices"
	"strconv"

	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	enumeration "github.com/synnaxlabs/synnax/pkg/service/enumeration/versions/v0"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/set"
)

// paint is the stroke or the fill of one part, with its color at each value.
type paint struct {
	part   string
	stroke bool
	colors map[float64]xcolor.Color
	// looks are the indexes of the looks that set the paint.
	looks []int
}

func (p paint) changes() bool {
	distinct := make(set.Set[xcolor.Color], len(p.colors))
	for _, c := range p.colors {
		distinct.Add(c)
	}
	return len(distinct) > 1
}

func (p paint) carried(entries map[float64]xcolor.Color) bool {
	for v, c := range p.colors {
		if e, ok := entries[v]; !ok || e != c {
			return false
		}
	}
	return true
}

// entryLooks takes the entry colors from the first stroke or fill whose color changes
// between values. It turns each stroke or fill whose colors equal the entries into one
// entry look, in the place of the first look that it replaces. It returns the entry
// colors, or nil when no paint changes.
func (m *specMigration) entryLooks() map[float64]xcolor.Color {
	paints := m.paints()
	first := slices.IndexFunc(paints, func(p *paint) bool { return p.changes() })
	if first < 0 {
		return nil
	}
	entries := paints[first].colors
	var (
		inserted = map[int][]Look{}
		strokes  = set.Set[int]{}
		fills    = set.Set[int]{}
	)
	for _, p := range paints {
		if !p.carried(entries) {
			continue
		}
		look := Look{
			Part: new(p.part),
			When: equals(slices.Sorted(maps.Keys(p.colors))...),
		}
		entry := &Paint{Variant: EntryPaint{State: activeState}}
		if p.stroke {
			look.Key, look.Stroke = "entry-"+p.part+"-stroke", entry
			strokes.Add(p.looks...)
		} else {
			look.Key, look.Fill = "entry-"+p.part+"-fill", entry
			fills.Add(p.looks...)
		}
		inserted[p.looks[0]] = append(inserted[p.looks[0]], look)
	}
	looks := make([]Look, 0, len(m.spec.Looks))
	for i, l := range m.spec.Looks {
		looks = append(looks, inserted[i]...)
		if strokes.Contains(i) {
			l.Stroke = nil
		}
		if fills.Contains(i) {
			l.Fill = nil
		}
		if l.Stroke != nil || l.Fill != nil {
			looks = append(looks, l)
		}
	}
	m.spec.Looks = looks
	return entries
}

// paints returns one paint for each part and field, in look order with the stroke
// before the fill.
func (m *specMigration) paints() []*paint {
	var paints []*paint
	add := func(look int, rp regionPaint, stroke bool, c *xcolor.Color) {
		if c == nil {
			return
		}
		idx := slices.IndexFunc(paints, func(p *paint) bool {
			return p.part == rp.part && p.stroke == stroke
		})
		if idx < 0 {
			paints = append(paints, &paint{
				part:   rp.part,
				stroke: stroke,
				colors: map[float64]xcolor.Color{},
			})
			idx = len(paints) - 1
		}
		paints[idx].colors[rp.value] = *c
		paints[idx].looks = append(paints[idx].looks, look)
	}
	for i, rp := range m.painted {
		add(i, rp, true, rp.stroke)
		add(i, rp, false, rp.fill)
	}
	return paints
}

// defaultEntries returns an entry for each old state: value i and the state's name.
func defaultEntries(
	states []v4.State,
	colors map[float64]xcolor.Color,
) []enumeration.Entry {
	entries := make([]enumeration.Entry, len(states))
	keys := set.Set[string]{}
	for i, st := range states {
		key := st.Key
		for n := i; key == "" || keys.Contains(key); n++ {
			key = strconv.Itoa(n)
		}
		keys.Add(key)
		entries[i] = enumeration.Entry{Key: key, Value: float64(i), Name: st.Name}
		if c, ok := colors[float64(i)]; ok {
			entries[i].Color = &color.Ref{Variant: color.FixedRef{Value: c}}
		}
	}
	return entries
}
