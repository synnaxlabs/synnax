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

// Actuators returns the keys of the symbols that a custom_actuator node draws.
type Actuators func(ctx context.Context, tx gorp.Tx) (set.Set[Key], error)

// NewMigration returns the migration that lifts stored symbols from v4 to v5 as
// MigrateSymbol does. It gives the state of each symbol a default enumeration with an
// entry for each old state, and turns the color looks that the entries carry into entry
// looks. A symbol that a custom_actuator node draws gets the commands of an actuator.
// newIntern opens the Intern of the migration's transaction.
func NewMigration(newIntern func(gorp.Tx) Intern, actuators Actuators) gorp.Migration {
	return migrate.New(
		"v62_symbol_model",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) (err error) {
			drawn, err := actuators(ctx, tx)
			if err != nil {
				return err
			}
			var (
				intern = newIntern(tx)
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
				if s, err = migrateStored(ctx, *old, drawn, intern, ins); err != nil {
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
	drawn set.Set[Key],
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
	if drawn.Contains(old.Key) {
		m.addActuator()
	}
	if len(old.Data.States) > 0 {
		entries := defaultEntries(old.Data.States, m.entryLooks())
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
	parts := partMigrator{tree: t, parsed: parsed, keys: map[string]string{}}
	for _, st := range old.States {
		for _, r := range st.Regions {
			parts.taken = append(parts.taken, r.Key)
		}
	}
	for i, st := range old.States {
		for _, r := range st.Regions {
			m.addLook(parts.part(&m.spec, r), float64(i), r)
		}
	}
	m.spec.Drawing = Drawing{Variant: SvgDrawing{Markup: parts.markup(old.SVG)}}
	m.dropped = parts.dropped
	if old.Variant == actuatorVariant {
		m.addActuator()
	}
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

// addActuator gives the spec the commands Off and On and a click between them.
func (m *specMigration) addActuator() {
	if m.state == nil {
		m.state = &EnumerationState{Key: activeState, Name: "Active"}
	}
	m.state.Match = Matcher{
		Variant: WithinMatcher{Bounds: spatial.Bounds{Lower: 0.9, Upper: 1.1}},
	}
	active := activeState
	m.spec.Commands = []Command{
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
	m.spec.Interactions = []Interaction{{Variant: NextInteraction{
		BaseInteraction: BaseInteraction{Key: "next"},
		Commands:        []string{deactivate, activate},
	}}}
}

// finish returns the spec with the state "active" when the symbol has it.
func (m *specMigration) finish() Spec {
	m.spec.States = []State{}
	if m.state != nil {
		m.spec.States = append(m.spec.States, State{Variant: *m.state})
	}
	return m.spec
}

// partMigrator turns regions into parts, one for each pair of region key and selector
// set.
type partMigrator struct {
	tree    *tree
	parsed  bool
	keys    map[string]string
	taken   []string
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
			if !slices.Contains(m.taken, key) {
				break
			}
		}
		m.taken = append(m.taken, key)
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
