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
	"strings"
	"uuid"

	"github.com/synnaxlabs/gorp"
	enumeration "github.com/synnaxlabs/synnax/pkg/service/enumeration/versions/v0"
	symbolv4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// overrides moves the state overrides of custom nodes to new records in one migration
// run. The symbols are still v4, since the symbol migration runs after this one.
type overrides struct {
	tx     gorp.Tx
	intern symbol.Intern
	// copies holds the copy of a symbol for each source symbol and set of overrides.
	copies map[string]symbol.Key
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
			src, found, err := o.source(ctx, ref.Key)
			if err != nil {
				return err
			}
			if found {
				if cfg, err = o.node(ctx, cfg, src, states, s.Name); err != nil {
					return errors.Wrapf(err, "node %s", n.Key)
				}
			}
		}
		s.Nodes[i].Config.Variant = cfg
	}
	return nil
}

// source returns the stored symbol with key, or false when the store does not hold it.
func (o *overrides) source(
	ctx context.Context,
	key symbol.Key,
) (symbolv4.Symbol, bool, error) {
	var src symbolv4.Symbol
	err := gorp.NewRetrieve[symbolv4.Key, symbolv4.Symbol]().
		Where(gorp.MatchKeys[symbolv4.Key, symbolv4.Symbol](key)).
		Entry(&src).
		Exec(ctx, o.tx)
	if errors.Is(err, query.ErrNotFound) {
		return src, false, nil
	}
	return src, err == nil, err
}

func (o *overrides) node(
	ctx context.Context,
	cfg SymbolNodeConfig,
	src symbolv4.Symbol,
	states []symbolv4.State,
	schematic string,
) (SymbolNodeConfig, error) {
	entries, drawn, err := redraw(ctx, src.Data, states)
	if err != nil || (entries == nil && drawn == nil) {
		return cfg, err
	}
	if drawn != nil {
		key, err := o.copy(ctx, src, *drawn, states, schematic)
		cfg.Spec = symbol.Ref{Variant: symbol.CustomRef{Key: key}}
		return cfg, err
	}
	label := strings.TrimSpace(cfg.Label.Label)
	if label == "" {
		label = schematic
	}
	key, err := o.intern(ctx, src.Name+": "+label, entries)
	cfg.States[0].Enumeration = &key
	return cfg, err
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

// copy returns the key of a copy of src that draws spec, in the group of src. Nodes
// share a copy when they have the same source symbol and the same overrides.
func (o *overrides) copy(
	ctx context.Context,
	src symbolv4.Symbol,
	spec symbolv4.Spec,
	states []symbolv4.State,
	schematic string,
) (symbol.Key, error) {
	b, err := json.Marshal(states)
	if err != nil {
		return uuid.Nil(), err
	}
	id := src.Key.String() + string(b)
	if key, ok := o.copies[id]; ok {
		return key, nil
	}
	c := symbolv4.Symbol{
		Key:   uuid.New(),
		Name:  src.Name + " (" + schematic + ")",
		Data:  spec,
		Group: src.Group,
	}
	if err = gorp.NewCreate[symbolv4.Key, symbolv4.Symbol]().
		Entry(&c).
		Exec(ctx, o.tx); err != nil {
		return uuid.Nil(), err
	}
	o.copies[id] = c.Key
	return c.Key, nil
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
