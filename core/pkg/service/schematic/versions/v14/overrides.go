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
	"encoding/json/v2"
	"maps"
	"slices"
	"strconv"
	"uuid"

	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	enumeration "github.com/synnaxlabs/synnax/pkg/service/enumeration/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	symbolv4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// overrides moves the state overrides of custom nodes to new records, and gives each
// custom actuator a symbol that actuates.
type overrides struct {
	tx     gorp.Tx
	intern symbol.Intern
	// create creates each copy.
	create func(ctx context.Context, sym *symbol.Symbol) error
	// retrievable returns false for a symbol or an enumeration that counts as missing.
	retrievable func(resource.ID) bool
	// copies holds the copy of a symbol for each source symbol and set of overrides.
	copies map[string]symbol.Key
}

// apply makes each custom node of old with state overrides a symbol node in s, with the
// look that its overrides drew: an enumeration with their entry colors, or a copy of
// its symbol when an enumeration cannot carry them. A custom actuator whose symbol does
// not actuate gets a copy that does. A node whose overrides equal its symbol, or whose
// symbol is gone, gets neither.
func (o *overrides) apply(ctx context.Context, s *Schematic, old v13.Schematic) error {
	for i, n := range old.Nodes {
		plain, states := withoutOverrides(n.Config)
		_, actuator := plain.Variant.(v13.CustomActuatorNodeConfig)
		if len(states) == 0 && !actuator {
			continue
		}
		cfg, _, err := SymbolNode(plain)
		if err != nil {
			return err
		}
		ref, ok := cfg.Spec.Variant.(symbol.CustomRef)
		if ok && o.retrievable(symbol.Symbol{Key: ref.Key}.ResourceID()) {
			src, found, err := retrieveSymbol(ctx, o.tx, ref.Key)
			if err != nil {
				return err
			}
			if found {
				if cfg, err = o.node(
					ctx,
					cfg,
					src,
					states,
					actuator,
					s.Name,
				); err != nil {
					return errors.Wrapf(err, "node %s", n.Key)
				}
			}
		}
		s.Nodes[i].Config.Variant = cfg
	}
	return nil
}

// node returns cfg with the enumeration or the copy of src that draws states. When
// actuator is true and src does not actuate, cfg gets a copy that does. cfg keeps src
// when retrievable rejects an enumeration that it would read or that a copy would link.
func (o *overrides) node(
	ctx context.Context,
	cfg SymbolNodeConfig,
	src symbol.Symbol,
	states []symbolv4.State,
	actuator bool,
	schematic string,
) (SymbolNodeConfig, error) {
	actuate := actuator && !actuates(src.Data)
	if actuate {
		src.Data = symbol.Actuate(src.Data)
	}
	var state symbol.EnumerationState
	if len(src.Data.States) > 0 {
		state, _ = src.Data.States[0].Variant.(symbol.EnumerationState)
	}
	if state.Enumeration != nil && !o.retrievable(
		enumeration.Enumeration{Key: *state.Enumeration}.ResourceID(),
	) {
		return cfg, nil
	}
	entries, err := entriesOf(ctx, o.tx, state)
	if err != nil {
		return SymbolNodeConfig{}, err
	}
	recolored, drawn, err := redraw(src.Data, state.Key, entries, states)
	if err != nil {
		return SymbolNodeConfig{}, err
	}
	if drawn == nil && actuate {
		drawn = &src.Data
	}
	if drawn != nil {
		if !o.linksRetrievable(drawn.States) {
			return cfg, nil
		}
		c := symbol.Symbol{Name: src.Name + " (" + schematic + ")", Group: src.Group}
		c.Data = *drawn
		key, err := o.copy(ctx, src.Key, states, actuate, c, recolored)
		if err != nil {
			return SymbolNodeConfig{}, err
		}
		cfg.Spec = symbol.Ref{Variant: symbol.CustomRef{Key: key}}
		return cfg, nil
	}
	if recolored == nil {
		return cfg, nil
	}
	name := src.Name + ": " + labelOr(cfg.Label.Label, schematic)
	key, err := o.intern(ctx, name, recolored)
	if err != nil {
		return SymbolNodeConfig{}, err
	}
	cfg.States[0].Enumeration = &key
	return cfg, nil
}

// copy creates c, a copy of the symbol with key src, and returns its key. When entries
// is not empty, the first state of c gets an enumeration of entries. Nodes share a copy
// when they have the same source symbol, overrides states, and actuate.
func (o *overrides) copy(
	ctx context.Context,
	src symbol.Key,
	states []symbolv4.State,
	actuate bool,
	c symbol.Symbol,
	entries []enumeration.Entry,
) (symbol.Key, error) {
	b, err := json.Marshal(states)
	if err != nil {
		return uuid.Nil(), err
	}
	id := src.String() + strconv.FormatBool(actuate) + string(b)
	if key, ok := o.copies[id]; ok {
		return key, nil
	}
	if len(entries) > 0 {
		if c.Data, err = o.enumerate(ctx, c.Name, c.Data, entries); err != nil {
			return uuid.Nil(), err
		}
	}
	if err = o.create(ctx, &c); err != nil {
		return uuid.Nil(), err
	}
	o.copies[id] = c.Key
	return c.Key, nil
}

// enumerate returns spec with an enumeration of entries, named name, on its first
// state.
func (o *overrides) enumerate(
	ctx context.Context,
	name string,
	spec symbol.Spec,
	entries []enumeration.Entry,
) (symbol.Spec, error) {
	e, err := o.intern(ctx, name, entries)
	if err != nil {
		return symbol.Spec{}, err
	}
	spec.States = slices.Clone(spec.States)
	state, ok := spec.States[0].Variant.(symbol.EnumerationState)
	if !ok {
		return symbol.Spec{}, errors.New("a copy draws only an enumeration state")
	}
	state.Enumeration = &e
	spec.States[0] = symbol.State{Variant: state}
	return spec, nil
}

// linksRetrievable returns whether retrievable allows the enumeration of each state.
func (o *overrides) linksRetrievable(states []symbol.State) bool {
	return !slices.ContainsFunc(states, func(st symbol.State) bool {
		e, ok := st.Variant.(symbol.EnumerationState)
		return ok && e.Enumeration != nil &&
			!o.retrievable(enumeration.Enumeration{Key: *e.Enumeration}.ResourceID())
	})
}

// actuates returns whether spec has the state "active" and a command that sets it.
func actuates(spec symbol.Spec) bool {
	return slices.ContainsFunc(spec.States, func(st symbol.State) bool {
		e, ok := st.Variant.(symbol.EnumerationState)
		return ok && e.Key == activeState
	}) && slices.ContainsFunc(spec.Commands, func(c symbol.Command) bool {
		return c.State != nil && *c.State == activeState
	})
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
	if err != nil {
		return nil, err
	}
	return e.Entries, nil
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

// draw returns what spec shows with entries, after its looks apply in order. A part
// with no selectors shows nothing.
func draw(spec symbol.Spec, entries []enumeration.Entry) drawing {
	shown := shownParts(spec)
	d := drawing{}
	for _, e := range entries {
		parts := map[string]paints{}
		for _, l := range spec.Looks {
			if l.Part == nil || !shown.Contains(*l.Part) || !applies(l.When, e.Value) {
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

// shownParts returns the keys of the parts of spec that select a shape.
func shownParts(spec symbol.Spec) set.Set[string] {
	shown := set.Set[string]{}
	for _, p := range spec.Parts {
		if len(p.Selectors) > 0 {
			shown.Add(p.Key)
		}
	}
	return shown
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
// where value i showed the regions of override i, and the spec with the parts and the
// markup that those regions need.
func overridden(
	spec symbol.Spec,
	entries []enumeration.Entry,
	states []symbolv4.State,
) (symbol.Spec, drawing, error) {
	want := maps.Clone(draw(spec, entries))
	svg, ok := spec.Drawing.Variant.(symbol.SvgDrawing)
	if !ok {
		return symbol.Spec{}, nil, errors.New("only an svg drawing has regions")
	}
	spec.Parts = slices.Clone(spec.Parts)
	for i, st := range states {
		value := float64(i)
		if _, ok := want[value]; !ok {
			continue
		}
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
		want[value] = parts
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

// redraw returns the entries of an enumeration that draws the overrides states on spec,
// a symbol whose enumeration state is state. When an enumeration cannot draw them, it
// also returns a copy of spec that can. It returns neither when states equal spec.
func redraw(
	spec symbol.Spec,
	state string,
	entries []enumeration.Entry,
	states []symbolv4.State,
) ([]enumeration.Entry, *symbol.Spec, error) {
	c, want, err := overridden(spec, entries, states)
	if err != nil || want.equal(draw(spec, entries)) {
		return nil, nil, err
	}
	recolored := recolor(spec, entries, want)
	if len(c.Parts) == len(spec.Parts) && draw(spec, recolored).equal(want) {
		return recolored, nil, nil
	}
	c.Looks = repaint(spec.Looks, state, recolored, want)
	return recolored, &c, nil
}

// recolor returns entries in the color that want paints, at each value, on the first
// shown part that a look of spec paints with the entry.
func recolor(
	spec symbol.Spec,
	entries []enumeration.Entry,
	want drawing,
) []enumeration.Entry {
	shown := shownParts(spec)
	out := slices.Clone(entries)
	for i, e := range out {
		out[i].Color = nil
		for _, l := range spec.Looks {
			if l.Part == nil || !shown.Contains(*l.Part) || !applies(l.When, e.Value) {
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

// withoutOverrides returns cfg with no state overrides, and the overrides it had.
func withoutOverrides(cfg v13.NodeConfig) (v13.NodeConfig, []symbolv4.State) {
	switch v := cfg.Variant.(type) {
	case v13.CustomActuatorNodeConfig:
		states := v.StateOverrides
		v.StateOverrides = nil
		return v13.NodeConfig{Variant: v}, states
	case v13.CustomStaticNodeConfig:
		states := v.StateOverrides
		v.StateOverrides = nil
		return v13.NodeConfig{Variant: v}, states
	}
	return cfg, nil
}
