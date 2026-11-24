// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v14

import (
	"maps"
	"slices"
	"strconv"

	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	enumeration "github.com/synnaxlabs/synnax/pkg/service/enumeration/versions/v0"
	symbolv4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	"github.com/synnaxlabs/x/errors"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
)

// paintPair is the stroke and fill that a part shows.
type paintPair struct{ stroke, fill *xcolor.Color }

func (p paintPair) equal(o paintPair) bool {
	same := func(a, b *xcolor.Color) bool {
		return (a == nil) == (b == nil) && (a == nil || *a == *b)
	}
	return same(p.stroke, o.stroke) && same(p.fill, o.fill)
}

// drawing is what a symbol shows: the paints of each part, for each value.
type drawing map[float64]map[string]paintPair

func (d drawing) equal(o drawing) bool {
	return maps.EqualFunc(d, o, func(a, b map[string]paintPair) bool {
		return maps.EqualFunc(a, b, paintPair.equal)
	})
}

func (d drawing) clone() drawing {
	c := make(drawing, len(d))
	for v, parts := range d {
		c[v] = maps.Clone(parts)
	}
	return c
}

func entryColor(entries []enumeration.Entry, v float64) *xcolor.Color {
	for _, e := range entries {
		if e.Value != v || e.Color == nil {
			continue
		}
		if f, ok := e.Color.Variant.(servicecolor.FixedRef); ok {
			return &f.Value
		}
	}
	return nil
}

func resolve(p *symbol.Paint, entries []enumeration.Entry, v float64) *xcolor.Color {
	switch pv := p.Variant.(type) {
	case symbol.EntryPaint:
		return entryColor(entries, v)
	case symbol.ColorPaint:
		if f, ok := pv.Value.Variant.(servicecolor.FixedRef); ok {
			return &f.Value
		}
	}
	return nil
}

func applies(c *symbol.Condition, v float64) bool {
	if c == nil {
		return true
	}
	eq, ok := c.Variant.(symbol.EqualsCondition)
	return ok && slices.Contains(eq.Values, v)
}

func values(entries []enumeration.Entry) []float64 {
	vs := make([]float64, len(entries))
	for i, e := range entries {
		vs[i] = e.Value
	}
	return vs
}

// draw returns what spec shows with entries, after its looks apply in order.
func draw(spec symbol.Spec, entries []enumeration.Entry) drawing {
	d := drawing{}
	for _, v := range values(entries) {
		parts := map[string]paintPair{}
		for _, l := range spec.Looks {
			if l.Part == nil || !applies(l.When, v) {
				continue
			}
			p := parts[*l.Part]
			if l.Stroke != nil {
				p.stroke = resolve(l.Stroke, entries, v)
			}
			if l.Fill != nil {
				p.fill = resolve(l.Fill, entries, v)
			}
			parts[*l.Part] = p
		}
		maps.DeleteFunc(parts, func(_ string, p paintPair) bool {
			return p.stroke == nil && p.fill == nil
		})
		d[v] = parts
	}
	return d
}

// overridden returns what a 0.62 Console drew for a node with overrides states on a
// symbol that shows has: state i showed the regions of override i.
func overridden(
	spec symbol.Spec,
	has drawing,
	vs []float64,
	states []symbolv4.State,
) (drawing, error) {
	want := has.clone()
	for i, st := range states[:min(len(states), len(vs))] {
		parts := map[string]paintPair{}
		for _, r := range st.Regions {
			if !slices.ContainsFunc(spec.Parts, func(p symbol.Part) bool {
				return p.Key == r.Key
			}) {
				return nil, errors.Newf("no part for region %s", r.Key)
			}
			if r.StrokeColor != nil || r.FillColor != nil {
				parts[r.Key] = paintPair{stroke: r.StrokeColor, fill: r.FillColor}
			}
		}
		want[vs[i]] = parts
	}
	return want, nil
}

// redrawStored is redraw for a stored v5 symbol with the entries of its enumeration.
// It returns the entries of an enumeration that draws states on spec, or a copy of
// spec and its entries when an enumeration cannot.
func redrawStored(
	spec symbol.Spec,
	entries []enumeration.Entry,
	states []symbolv4.State,
) ([]enumeration.Entry, *symbol.Spec, error) {
	has := draw(spec, entries)
	want, err := overridden(spec, has, values(entries), states)
	if err != nil || want.equal(has) {
		return nil, nil, err
	}
	recolored := slices.Clone(entries)
	for i, e := range recolored {
		recolored[i].Color = nil
		for _, l := range spec.Looks {
			if l.Part == nil || !applies(l.When, e.Value) {
				continue
			}
			p := want[e.Value][*l.Part]
			if isEntry(l.Stroke) {
				recolored[i].Color = fixed(p.stroke)
				break
			}
			if isEntry(l.Fill) {
				recolored[i].Color = fixed(p.fill)
				break
			}
		}
	}
	if draw(spec, recolored).equal(want) {
		return recolored, nil, nil
	}
	c := spec
	c.Looks = slices.Clone(spec.Looks)
	got := draw(c, recolored)
	for _, v := range values(recolored) {
		parts := slices.Collect(maps.Keys(want[v]))
		parts = append(parts, slices.Collect(maps.Keys(got[v]))...)
		slices.Sort(parts)
		for _, part := range slices.Compact(parts) {
			w, g := want[v][part], got[v][part]
			if w.equal(g) {
				continue
			}
			if w.stroke == nil && g.stroke != nil {
				c.Looks = unset(c.Looks, part, v, true)
			}
			if w.fill == nil && g.fill != nil {
				c.Looks = unset(c.Looks, part, v, false)
			}
			if w.stroke == nil && w.fill == nil {
				continue
			}
			look := symbol.Look{
				Key:  "override-" + part + "-" + strconv.Itoa(len(c.Looks)),
				Part: new(part),
				When: &symbol.Condition{Variant: symbol.EqualsCondition{
					State: activeState, Values: []float64{v},
				}},
			}
			if w.stroke != nil {
				look.Stroke = colorPaint(*w.stroke)
			}
			if w.fill != nil {
				look.Fill = colorPaint(*w.fill)
			}
			c.Looks = append(c.Looks, look)
		}
	}
	if !draw(c, recolored).equal(want) {
		return nil, nil, errors.New("a copy cannot draw the overrides")
	}
	return recolored, &c, nil
}

// unset returns looks with no stroke, or no fill, on part at v. A look that also
// applies at other values keeps its paint there.
func unset(looks []symbol.Look, part string, v float64, stroke bool) []symbol.Look {
	out := make([]symbol.Look, 0, len(looks))
	for _, l := range looks {
		field := func(l *symbol.Look) **symbol.Paint {
			if stroke {
				return &l.Stroke
			}
			return &l.Fill
		}
		eq, ok := l.When.Variant.(symbol.EqualsCondition)
		if l.Part == nil || *l.Part != part || *field(&l) == nil || !ok ||
			!slices.Contains(eq.Values, v) {
			out = append(out, l)
			continue
		}
		if rest := slices.DeleteFunc(slices.Clone(eq.Values), func(x float64) bool {
			return x == v
		}); len(rest) > 0 {
			r := l
			r.When = &symbol.Condition{Variant: symbol.EqualsCondition{
				State: eq.State, Values: rest,
			}}
			out = append(out, r)
		}
		at := l
		at.Key = l.Key + "-" + strconv.FormatFloat(v, 'f', -1, 64)
		at.When = &symbol.Condition{Variant: symbol.EqualsCondition{
			State: eq.State, Values: []float64{v},
		}}
		*field(&at) = nil
		if at.Stroke != nil || at.Fill != nil {
			out = append(out, at)
		}
	}
	return out
}

func isEntry(p *symbol.Paint) bool {
	if p == nil {
		return false
	}
	_, ok := p.Variant.(symbol.EntryPaint)
	return ok
}

func colorPaint(c xcolor.Color) *symbol.Paint {
	return &symbol.Paint{Variant: symbol.ColorPaint{Value: *fixed(&c)}}
}
