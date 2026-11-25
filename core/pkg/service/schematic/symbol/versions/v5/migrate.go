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
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	enumeration "github.com/synnaxlabs/synnax/pkg/service/enumeration/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/selector"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	"go.uber.org/zap"
)

const (
	activeState     = "active"
	actuatorVariant = "actuator"
	deactivate      = "deactivate"
	activate        = "activate"
)

// MigrateSymbol lifts a v4 symbol to v5. Each region becomes a part, state i becomes
// value i of the state "active", and each region color becomes a color look on that
// value. An actuator gets the commands Off and On and a click that picks between them.
func MigrateSymbol(ctx context.Context, old v4.Symbol) (Symbol, error) {
	return autoMigrateSymbol(ctx, old)
}

// MigrateSpec lifts a v4 spec to v5 as MigrateSymbol does.
func MigrateSpec(ctx context.Context, old v4.Spec) (Spec, error) {
	m, err := migrateSpec(ctx, old)
	if err != nil {
		return Spec{}, err
	}
	return m.finish(), nil
}

// Intern returns the key of an enumeration with name and entries. It creates one in the
// migration's transaction, or returns one that an earlier call made with equal entries.
type Intern func(
	ctx context.Context,
	name string,
	entries []enumeration.Entry,
) (enumeration.Key, error)

// MigrationConfig is the configuration for NewMigration.
type MigrationConfig struct {
	// Intern opens the Intern of the migration's transaction.
	Intern func(gorp.Tx) Intern
}

// NewMigration returns the migration that lifts stored symbols from v4 to v5 as
// MigrateSymbol does. It gives the state of each symbol a default enumeration with an
// entry for each old state, and turns the color looks that the entries carry into entry
// looks.
func NewMigration(cfg MigrationConfig) gorp.Migration {
	return migrate.New(
		"v63_symbol_model",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) (err error) {
			var (
				intern = cfg.Intern(tx)
				reader = gorp.WrapReader[Key, v4.Symbol](tx)
				writer = gorp.WrapWriter[Key, Symbol](tx)
			)
			iter, err := reader.OpenIterator(gorp.IterOptions{})
			if err != nil {
				return err
			}
			defer func() { err = errors.Combine(err, iter.Close()) }()
			for iter.First(); iter.Valid(); iter.Next() {
				old := iter.Value()
				if err = iter.Error(); err != nil {
					return err
				}
				var s Symbol
				if s, err = migrateStored(ctx, *old, intern, ins); err != nil {
					return errors.Wrapf(err, "symbol %v", old.Key)
				}
				if err = writer.Set(ctx, s); err != nil {
					return err
				}
			}
			return err
		},
	)
}

func migrateStored(
	ctx context.Context,
	old v4.Symbol,
	intern Intern,
	ins alamos.Instrumentation,
) (Symbol, error) {
	m, err := migrateSpec(ctx, old.Data)
	if err != nil {
		return Symbol{}, err
	}
	for _, sel := range m.dropped {
		ins.L.Warn(
			"dropping a symbol part selector that names no shape",
			zap.Stringer("symbol", old.Key),
			zap.String("selector", sel),
		)
	}
	if len(old.Data.States) > 0 {
		entries := defaultEntries(old.Data.States, m.entryLooks())
		if m.actuator {
			entries, _ = ActuatorEntries(entries)
		}
		key, err := intern(ctx, old.Name, entries)
		if err != nil {
			return Symbol{}, err
		}
		m.state.Enumeration = &key
	}
	return Symbol{Key: old.Key, Name: old.Name, Data: m.finish(), Group: old.Group}, nil
}

// specMigration is a v5 spec in progress.
type specMigration struct {
	spec Spec
	// state is the state "active", or nil when the symbol has no state.
	state *EnumerationState
	// painted holds the region colors of each look, in look order.
	painted []regionPaint
	// dropped holds the selectors that select no shape.
	dropped []string
	// actuator is true when the spec gets the commands of an actuator.
	actuator bool
}

// regionPaint is the stroke and fill that a region gives its part at a value.
type regionPaint struct {
	part   string
	value  float64
	stroke *xcolor.Color
	fill   *xcolor.Color
}

func migrateSpec(ctx context.Context, old v4.Spec) (*specMigration, error) {
	spec, err := autoMigrateSpec(ctx, old)
	if err != nil {
		return nil, err
	}
	spec.Parts = []Part{}
	spec.Commands = []Command{}
	spec.Looks = []Look{}
	spec.Interactions = []Interaction{}
	m := &specMigration{spec: spec}
	if len(old.States) > 0 || old.Variant == actuatorVariant {
		m.state = &EnumerationState{
			Key:   activeState,
			Name:  "Active",
			Match: Matcher{Variant: ExactMatcher{}},
		}
	}
	t, parsed := parseTree(old.SVG)
	parts := partMigrator{
		tree:   t,
		parsed: parsed,
		keys:   map[string]string{},
		taken:  set.Set[string]{},
	}
	for _, st := range old.States {
		for _, r := range st.Regions {
			parts.taken.Add(r.Key)
		}
	}
	for i, st := range old.States {
		for _, r := range st.Regions {
			m.addLook(parts.part(&m.spec, r), float64(i), r)
		}
	}
	m.spec.Drawing = Drawing{Variant: SvgDrawing{Markup: parts.markup(old.SVG)}}
	m.dropped = parts.dropped
	m.actuator = old.Variant == actuatorVariant
	return m, nil
}

// addLook adds a color look on value for the colors of r, when r has a color.
func (m *specMigration) addLook(part string, value float64, r v4.Region) {
	if r.StrokeColor == nil && r.FillColor == nil {
		return
	}
	look := Look{
		Key:  "look-" + strconv.Itoa(len(m.spec.Looks)+1),
		Part: &part,
		When: equals(value),
	}
	if r.StrokeColor != nil {
		look.Stroke = colorPaint(*r.StrokeColor)
	}
	if r.FillColor != nil {
		look.Fill = colorPaint(*r.FillColor)
	}
	m.spec.Looks = append(m.spec.Looks, look)
	m.painted = append(m.painted, regionPaint{
		part:   part,
		value:  value,
		stroke: r.StrokeColor,
		fill:   r.FillColor,
	})
}

// Actuate returns spec as an actuator: the commands Off and On of the state "active", a
// click that picks between them, and the variant actuator. It adds the state when spec
// has none, and makes the state match within 0.9 and 1.1. When no look holds on value
// 1, each look on value 0 holds on value 1 too. The enumeration of the state needs
// ActuatorEntries.
func Actuate(spec Spec) Spec {
	spec.Variant = actuatorVariant
	spec.Looks = activeLooks(spec.Looks)
	spec.States = slices.Clone(spec.States)
	st := EnumerationState{Key: activeState, Name: "Active"}
	i := slices.IndexFunc(spec.States, func(s State) bool {
		e, ok := s.Variant.(EnumerationState)
		if ok && e.Key == activeState {
			st = e
		}
		return ok && e.Key == activeState
	})
	if i < 0 {
		i = len(spec.States)
		spec.States = append(spec.States, State{})
	}
	st.Match = Matcher{
		Variant: WithinMatcher{Bounds: spatial.Bounds{Lower: 0.9, Upper: 1.1}},
	}
	spec.States[i] = State{Variant: st}
	active := activeState
	spec.Commands = []Command{
		{
			Key:   deactivate,
			Name:  "Off",
			State: &active,
			Value: CommandValue{Variant: FixedCommandValue{Value: 0}},
		},
		{
			Key:   activate,
			Name:  "On",
			State: &active,
			Value: CommandValue{Variant: FixedCommandValue{Value: 1}},
		},
	}
	spec.Interactions = []Interaction{{Variant: NextInteraction{
		BaseInteraction: BaseInteraction{Key: "next"},
		Commands:        []string{deactivate, activate},
	}}}
	return spec
}

// activeLooks returns looks with each look on value 0 also on value 1, when no look
// holds on value 1.
func activeLooks(looks []Look) []Look {
	if slices.ContainsFunc(looks, func(l Look) bool {
		values, ok := activeValues(l)
		return ok && slices.Contains(values, 1)
	}) {
		return looks
	}
	looks = slices.Clone(looks)
	for i, l := range looks {
		if values, ok := activeValues(l); ok && slices.Contains(values, 0) {
			looks[i].When = equals(append(slices.Clone(values), 1)...)
		}
	}
	return looks
}

// activeValues returns the values of the state "active" that the condition of l holds
// on, and false when l has no such condition.
func activeValues(l Look) ([]float64, bool) {
	if l.When == nil {
		return nil, false
	}
	eq, ok := l.When.Variant.(EqualsCondition)
	return eq.Values, ok && eq.State == activeState
}

// finish returns the spec with the state "active" when the symbol has it.
func (m *specMigration) finish() Spec {
	m.spec.States = []State{}
	if m.state != nil {
		m.spec.States = append(m.spec.States, State{Variant: *m.state})
	}
	if m.actuator {
		return Actuate(m.spec)
	}
	return m.spec
}

// partMigrator turns regions into parts, one for each pair of region key and selector
// set.
type partMigrator struct {
	tree    *tree
	parsed  bool
	keys    map[string]string
	taken   set.Set[string]
	dropped []string
}

func (m *partMigrator) part(spec *Spec, r v4.Region) string {
	sorted := slices.Clone(r.Selectors)
	slices.Sort(sorted)
	id := r.Key + "\x00" + strings.Join(slices.Compact(sorted), "\x00")
	if key, ok := m.keys[id]; ok {
		return key
	}
	key := r.Key
	if slices.ContainsFunc(spec.Parts, func(p Part) bool { return p.Key == key }) {
		for n := 2; ; n++ {
			key = r.Key + "-" + strconv.Itoa(n)
			if !m.taken.Contains(key) {
				break
			}
		}
		m.taken.Add(key)
	}
	m.keys[id] = key
	part := Part{Key: key, Name: r.Name, Selectors: []string{}}
	for _, sel := range r.Selectors {
		for _, named := range m.resolve(sel) {
			if !slices.Contains(part.Selectors, named) {
				part.Selectors = append(part.Selectors, named)
			}
		}
	}
	spec.Parts = append(spec.Parts, part)
	return key
}

// resolve returns sel when the symbol checker accepts it as a name, and otherwise a
// data-region-id selector for each shape that sel selects. It drops a sel that selects
// no shape.
func (m *partMigrator) resolve(sel string) []string {
	if selector.IsName(sel) {
		return []string{sel}
	}
	var shapes []*element
	if chain, err := parseSelector(sel); err == nil && m.parsed {
		shapes = m.tree.selectAll(chain)
	}
	if len(shapes) == 0 {
		m.dropped = append(m.dropped, sel)
		return nil
	}
	out := make([]string, len(shapes))
	for i, el := range shapes {
		out[i] = m.tree.name(el)
	}
	return out
}

// ResolveRegion returns the selectors of the part that a v4 region with selectors sels
// gets in a drawing with markup, as MigrateSpec names parts. It also returns markup
// with a data-region-id on each shape that those selectors name.
func ResolveRegion(markup string, sels []string) ([]string, string) {
	t, parsed := parseTree(markup)
	m := partMigrator{tree: t, parsed: parsed}
	out := []string{}
	for _, sel := range sels {
		for _, named := range m.resolve(sel) {
			if !slices.Contains(out, named) {
				out = append(out, named)
			}
		}
	}
	return out, m.markup(markup)
}

func (m *partMigrator) markup(svg string) string {
	if !m.parsed {
		return svg
	}
	return m.tree.markupWithIDs()
}

func equals(values ...float64) *Condition {
	return &Condition{
		Variant: EqualsCondition{State: activeState, Values: values},
	}
}

func colorPaint(c xcolor.Color) *Paint {
	return &Paint{Variant: ColorPaint{Value: color.Ref{
		Variant: color.FixedRef{Value: c},
	}}}
}
