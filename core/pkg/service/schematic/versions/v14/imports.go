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
	"context"
	"maps"
	"slices"
	"strconv"
	"uuid"

	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	enumeration "github.com/synnaxlabs/synnax/pkg/service/enumeration/versions/v0"
	symbolv4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// ImportOverrides gives each custom node of old with state overrides, in an import of
// an old file on tx, the enumeration or the copy of its symbol that NewMigration gives
// a stored node. s must be old lifted to v14. A symbol that retrievable rejects counts
// as missing. Each enumeration comes from intern, and create creates each copy.
func ImportOverrides(
	ctx context.Context,
	tx gorp.Tx,
	s *Schematic,
	old v13.Schematic,
	retrievable func(symbol.Key) bool,
	intern symbol.Intern,
	create func(ctx context.Context, sym *symbol.Symbol) error,
) error {
	o := &overrides{intern: intern, copies: map[string]symbol.Key{}}
	o.redraw = func(
		ctx context.Context,
		key symbol.Key,
		states []symbolv4.State,
	) (redrawn, bool, error) {
		if !retrievable(key) {
			return redrawn{}, false, nil
		}
		var src symbol.Symbol
		if err := gorp.NewRetrieve[symbol.Key, symbol.Symbol]().
			Where(gorp.MatchKeys[symbol.Key, symbol.Symbol](key)).
			Entry(&src).
			Exec(ctx, tx); err != nil {
			return redrawn{}, false, errors.Skip(err, query.ErrNotFound)
		}
		var state symbol.EnumerationState
		if len(src.Data.States) > 0 {
			state, _ = src.Data.States[0].Variant.(symbol.EnumerationState)
		}
		entries, err := entriesOf(ctx, tx, state)
		if err != nil {
			return redrawn{}, false, err
		}
		recolored, drawn, err := redrawStored(src.Data, state.Key, entries, states)
		r := redrawn{name: src.Name}
		if drawn == nil {
			r.entries = recolored
			return r, true, err
		}
		r.copy = func(ctx context.Context, name string) (symbol.Key, error) {
			e, err := intern(ctx, name, recolored)
			if err != nil {
				return uuid.Nil(), err
			}
			state.Enumeration = &e
			c := symbol.Symbol{Name: name, Data: *drawn, Group: src.Group}
			c.Data.States = []symbol.State{{Variant: state}}
			err = create(ctx, &c)
			return c.Key, err
		}
		return r, true, err
	}
	return o.apply(ctx, s, old)
}

// entriesOf returns the entries of the stored enumeration of state, or none when state
// has none.
func entriesOf(
	ctx context.Context,
	tx gorp.Tx,
	state symbol.EnumerationState,
) ([]enumeration.Entry, error) {
	if state.Enumeration == nil {
		return nil, nil
	}
	var e enumeration.Enumeration
	err := gorp.NewRetrieve[enumeration.Key, enumeration.Enumeration]().
		Where(gorp.MatchKeys[enumeration.Key, enumeration.Enumeration](*state.Enumeration)).
		Entry(&e).
		Exec(ctx, tx)
	return e.Entries, err
}

// paints is the stroke and fill that a part shows.
type paints struct{ stroke, fill *xcolor.Color }

func (p paints) equal(o paints) bool {
	same := func(a, b *xcolor.Color) bool {
		return (a == nil) == (b == nil) && (a == nil || *a == *b)
	}
	return same(p.stroke, o.stroke) && same(p.fill, o.fill)
}

// drawing is what a symbol shows: the paints of each part, for each value.
type drawing map[float64]map[string]paints

func (d drawing) equal(o drawing) bool {
	return maps.EqualFunc(d, o, func(a, b map[string]paints) bool {
		return maps.EqualFunc(a, b, paints.equal)
	})
}

// draw returns what spec shows with entries, after its looks apply in order.
func draw(spec symbol.Spec, entries []enumeration.Entry) drawing {
	d := drawing{}
	for _, e := range entries {
		parts := map[string]paints{}
		for _, l := range spec.Looks {
			if l.Part == nil || !applies(l.When, e.Value) {
				continue
			}
			p := parts[*l.Part]
			if l.Stroke != nil {
				p.stroke = paintColor(l.Stroke, e)
			}
			if l.Fill != nil {
				p.fill = paintColor(l.Fill, e)
			}
			parts[*l.Part] = p
		}
		maps.DeleteFunc(parts, func(_ string, p paints) bool {
			return p.stroke == nil && p.fill == nil
		})
		d[e.Value] = parts
	}
	return d
}

func applies(c *symbol.Condition, v float64) bool {
	if c == nil {
		return true
	}
	eq, ok := c.Variant.(symbol.EqualsCondition)
	return ok && slices.Contains(eq.Values, v)
}

// paintColor returns the color that p paints at the value of e, or nil when the node or
// the background sets the color.
func paintColor(p *symbol.Paint, e enumeration.Entry) *xcolor.Color {
	switch pv := p.Variant.(type) {
	case symbol.EntryPaint:
		return entryColor(e)
	case symbol.ColorPaint:
		if f, ok := pv.Value.Variant.(servicecolor.FixedRef); ok {
			return &f.Value
		}
	}
	return nil
}

// overridden returns what a 0.62 Console drew for a node with overrides states on spec,
// where state i showed the regions of override i, and the spec with the parts and the
// markup that those regions need.
func overridden(
	spec symbol.Spec,
	entries []enumeration.Entry,
	states []symbolv4.State,
) (symbol.Spec, drawing, error) {
	want := maps.Clone(draw(spec, entries))
	svg, ok := spec.Drawing.Variant.(symbol.SvgDrawing)
	if !ok {
		return spec, nil, errors.New("only an svg drawing has regions")
	}
	spec.Parts = slices.Clone(spec.Parts)
	for i, st := range states[:min(len(states), len(entries))] {
		parts := map[string]paints{}
		for _, r := range st.Regions {
			if r.StrokeColor == nil && r.FillColor == nil {
				continue
			}
			var sels []string
			sels, svg.Markup = symbol.ResolveRegion(svg.Markup, r.Selectors)
			if len(sels) == 0 {
				continue
			}
			key := partKey(&spec, r, sels)
			parts[key] = paints{stroke: r.StrokeColor, fill: r.FillColor}
		}
		want[entries[i].Value] = parts
	}
	spec.Drawing = symbol.Drawing{Variant: svg}
	return spec, want, nil
}

// partKey returns the key of the part of spec with selectors sels, and adds a part for
// r when spec has none.
func partKey(spec *symbol.Spec, r symbolv4.Region, sels []string) string {
	for _, p := range spec.Parts {
		if len(p.Selectors) == len(sels) &&
			!slices.ContainsFunc(sels, func(s string) bool {
				return !slices.Contains(p.Selectors, s)
			}) {
			return p.Key
		}
	}
	key := r.Key
	for n := 2; slices.ContainsFunc(spec.Parts, func(p symbol.Part) bool {
		return p.Key == key
	}); n++ {
		key = r.Key + "-" + strconv.Itoa(n)
	}
	spec.Parts = append(
		spec.Parts,
		symbol.Part{Key: key, Name: r.Name, Selectors: sels},
	)
	return key
}

// redrawStored is redraw for a stored v5 symbol with the enumeration state state. It
// returns the entries of an enumeration that draws states on spec, or a copy of spec
// and its entries when an enumeration cannot. It returns neither when states equal
// spec.
func redrawStored(
	spec symbol.Spec,
	state string,
	entries []enumeration.Entry,
	states []symbolv4.State,
) ([]enumeration.Entry, *symbol.Spec, error) {
	c, want, err := overridden(spec, entries, states)
	if err != nil || want.equal(draw(spec, entries)) {
		return nil, nil, err
	}
	recolored := recolor(spec.Looks, entries, want)
	if len(c.Parts) == len(spec.Parts) && draw(spec, recolored).equal(want) {
		return recolored, nil, nil
	}
	c.Looks = repaint(spec.Looks, state, recolored, want)
	return recolored, &c, nil
}

// recolor returns entries in the color that want paints, at each value, on the first
// part that a look paints with the entry.
func recolor(
	looks []symbol.Look,
	entries []enumeration.Entry,
	want drawing,
) []enumeration.Entry {
	out := slices.Clone(entries)
	for i, e := range out {
		out[i].Color = nil
		for _, l := range looks {
			if l.Part == nil || !applies(l.When, e.Value) {
				continue
			}
			p := want[e.Value][*l.Part]
			if isEntry(l.Stroke) {
				out[i].Color = fixed(p.stroke)
				break
			}
			if isEntry(l.Fill) {
				out[i].Color = fixed(p.fill)
				break
			}
		}
	}
	return out
}

// repaint returns looks with no paint on parts but the paints that paintColor cannot
// resolve, then a look for each part at each value of state that paints it as want
// does. A paint in the entry's color paints with the entry.
func repaint(
	looks []symbol.Look,
	state string,
	entries []enumeration.Entry,
	want drawing,
) []symbol.Look {
	out := make([]symbol.Look, 0, len(looks))
	taken := set.Set[string]{}
	for _, l := range looks {
		if l.Part != nil {
			l.Stroke, l.Fill = unresolved(l.Stroke), unresolved(l.Fill)
			if l.Stroke == nil && l.Fill == nil && l.Hidden == nil && l.Motion == nil &&
				l.Blink == nil {
				continue
			}
		}
		taken.Add(l.Key)
		out = append(out, l)
	}
	for _, e := range entries {
		paint := func(c *xcolor.Color) *symbol.Paint {
			if c == nil {
				return nil
			}
			if ec := entryColor(e); ec != nil && *ec == *c {
				return &symbol.Paint{Variant: symbol.EntryPaint{State: state}}
			}
			return colorPaint(*c)
		}
		for _, part := range slices.Sorted(maps.Keys(want[e.Value])) {
			key := "look-" + strconv.Itoa(len(out)+1)
			for n := len(out) + 2; taken.Contains(key); n++ {
				key = "look-" + strconv.Itoa(n)
			}
			taken.Add(key)
			out = append(out, symbol.Look{
				Key:  key,
				Part: new(part),
				When: &symbol.Condition{Variant: symbol.EqualsCondition{
					State:  state,
					Values: []float64{e.Value},
				}},
				Stroke: paint(want[e.Value][part].stroke),
				Fill:   paint(want[e.Value][part].fill),
			})
		}
	}
	return out
}

// unresolved returns p when the node or the background sets its color, and otherwise
// nil.
func unresolved(p *symbol.Paint) *symbol.Paint {
	if p == nil {
		return nil
	}
	switch p.Variant.(type) {
	case symbol.NodePaint, symbol.ContrastPaint:
		return p
	}
	return nil
}

func entryColor(e enumeration.Entry) *xcolor.Color {
	if e.Color == nil {
		return nil
	}
	if f, ok := e.Color.Variant.(servicecolor.FixedRef); ok {
		return &f.Value
	}
	return nil
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
