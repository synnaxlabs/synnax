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
	"reflect"
	"slices"
	"uuid"

	"github.com/synnaxlabs/gorp"
	enumeration "github.com/synnaxlabs/synnax/pkg/service/enumeration/versions/v0"
	symbolv4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// overrides moves the state overrides of custom nodes to new records in one run.
type overrides struct {
	intern symbol.Intern
	// redraw returns how states redraw the stored symbol with key, or false when the
	// Core does not hold it.
	redraw func(
		ctx context.Context,
		key symbol.Key,
		states []symbolv4.State,
	) (redrawn, bool, error)
	// copies holds the copy of a symbol for each source symbol and set of overrides.
	copies map[string]symbol.Key
}

// redrawn is how overrides redraw a stored symbol: the entries of an enumeration that
// draws them, or a copy that draws them when an enumeration cannot. It holds neither
// when the overrides equal the symbol.
type redrawn struct {
	// name is the name of the stored symbol.
	name    string
	entries []enumeration.Entry
	// copy creates a copy named name that draws the overrides, and returns its key.
	copy func(ctx context.Context, name string) (symbol.Key, error)
}

// upgrade returns the overrides of stored schematics in tx. The symbols are still v4,
// since the symbol migration runs after this one, and it converts each copy.
func upgrade(tx gorp.Tx, intern symbol.Intern) *overrides {
	return &overrides{
		intern: intern,
		copies: map[string]symbol.Key{},
		redraw: func(
			ctx context.Context,
			key symbol.Key,
			states []symbolv4.State,
		) (redrawn, bool, error) {
			var src symbolv4.Symbol
			err := gorp.NewRetrieve[symbolv4.Key, symbolv4.Symbol]().
				Where(gorp.MatchKeys[symbolv4.Key, symbolv4.Symbol](key)).
				Entry(&src).
				Exec(ctx, tx)
			if err != nil {
				return redrawn{}, false, errors.Skip(err, query.ErrNotFound)
			}
			entries, drawn, err := redraw(ctx, src.Data, states)
			r := redrawn{name: src.Name, entries: entries}
			if drawn != nil {
				r.copy = func(ctx context.Context, name string) (symbol.Key, error) {
					c := symbolv4.Symbol{
						Key:   uuid.New(),
						Name:  name,
						Data:  *drawn,
						Group: src.Group,
					}
					return c.Key, gorp.NewCreate[symbolv4.Key, symbolv4.Symbol]().
						Entry(&c).
						Exec(ctx, tx)
				}
			}
			return r, true, err
		},
	}
}

// apply makes each custom node of old with state overrides a symbol node in s, with the
// look that its overrides drew: an enumeration with their entry colors, or a copy of
// its symbol when an enumeration cannot carry them. A node whose overrides equal its
// symbol, or whose symbol is gone, gets neither.
func (o *overrides) apply(ctx context.Context, s *Schematic, old v13.Schematic) error {
	for i, n := range old.Nodes {
		plain, states := withoutOverrides(n.Config)
		if len(states) == 0 {
			continue
		}
		cfg, _, err := SymbolNode(plain)
		if err != nil {
			return err
		}
		if ref, ok := cfg.Spec.Variant.(symbol.CustomRef); ok {
			if cfg, err = o.node(ctx, cfg, ref.Key, states, s.Name); err != nil {
				return errors.Wrapf(err, "node %s", n.Key)
			}
		}
		s.Nodes[i].Config.Variant = cfg
	}
	return nil
}

// node returns cfg with the enumeration or the copy of the symbol with key that draws
// states.
func (o *overrides) node(
	ctx context.Context,
	cfg SymbolNodeConfig,
	key symbol.Key,
	states []symbolv4.State,
	schematic string,
) (SymbolNodeConfig, error) {
	r, found, err := o.redraw(ctx, key, states)
	if err != nil || !found {
		return cfg, err
	}
	if r.copy != nil {
		c, err := o.copy(ctx, key, r, states, schematic)
		cfg.Spec = symbol.Ref{Variant: symbol.CustomRef{Key: c}}
		return cfg, err
	}
	if r.entries == nil {
		return cfg, nil
	}
	name := r.name + ": " + labelOr(cfg.Label.Label, schematic)
	e, err := o.intern(ctx, name, r.entries)
	cfg.States[0].Enumeration = &e
	return cfg, err
}

// copy returns the key of the copy that r makes. Nodes share a copy when they have the
// same source symbol and the same overrides.
func (o *overrides) copy(
	ctx context.Context,
	src symbol.Key,
	r redrawn,
	states []symbolv4.State,
	schematic string,
) (symbol.Key, error) {
	b, err := json.Marshal(states)
	if err != nil {
		return uuid.Nil(), err
	}
	id := src.String() + string(b)
	if key, ok := o.copies[id]; ok {
		return key, nil
	}
	key, err := r.copy(ctx, r.name+" ("+schematic+")")
	if err != nil {
		return uuid.Nil(), err
	}
	o.copies[id] = key
	return key, nil
}

// redraw returns the entries of an enumeration that draws the overrides states on src,
// or the spec of a copy of src when an enumeration cannot draw them. It returns neither
// when the overrides equal src.
func redraw(
	ctx context.Context,
	src symbolv4.Spec,
	states []symbolv4.State,
) ([]enumeration.Entry, *symbolv4.Spec, error) {
	// State i drew the regions of override i, as a 0.62 Console drew it.
	drawn := src
	drawn.States = slices.Clone(src.States)
	for i := range min(len(states), len(drawn.States)) {
		drawn.States[i].Regions = states[i].Regions
	}
	want, entries, err := symbol.MigrateStoredSpec(ctx, drawn)
	if err != nil {
		return nil, nil, err
	}
	has, hasEntries, err := symbol.MigrateStoredSpec(ctx, src)
	if err != nil {
		return nil, nil, err
	}
	if !reflect.DeepEqual(want, has) {
		return nil, &drawn, nil
	}
	if reflect.DeepEqual(entries, hasEntries) {
		return nil, nil, nil
	}
	return entries, nil, nil
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
