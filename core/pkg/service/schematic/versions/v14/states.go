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
	"math"
	"strings"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	enumeration "github.com/synnaxlabs/synnax/pkg/service/enumeration/versions/v0"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// MigrationConfig is the configuration for NewMigration.
type MigrationConfig struct {
	// Intern opens the Intern of the migration's transaction.
	Intern func(gorp.Tx) symbol.Intern
}

// NewMigration returns the migration that lifts stored schematics from v13 to v14 as
// MigrateSchematic does, then gives their selects, state indicators, and lights with an
// on color enumerations. A custom node with state overrides gets an enumeration or a
// copy of its symbol. A symbol node whose custom symbol is gone gets a missing ref.
func NewMigration(cfg MigrationConfig) gorp.Migration {
	const key = "v63_color_ref"
	return migrate.New(
		key,
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			intern := cfg.Intern(tx)
			o := &overrides{tx: tx, intern: intern, copies: map[string]symbol.Key{}}
			return migrate.NewInstrumentedEntry(
				key,
				func(
					ctx context.Context,
					old v13.Schematic,
					ins alamos.Instrumentation,
				) (Schematic, error) {
					s, err := MigrateSchematic(ctx, old)
					if err != nil {
						return Schematic{}, err
					}
					lists := stateLists(old)
					if err = internStates(ctx, &s, lists, intern, ins); err != nil {
						return Schematic{}, err
					}
					if err = internLights(ctx, &s, old, intern); err != nil {
						return Schematic{}, err
					}
					if err = o.apply(ctx, &s, old); err != nil {
						return Schematic{}, err
					}
					warnEmptyThresholds(old, ins)
					return s, markMissing(ctx, tx, &s)
				},
			).Run(ctx, tx, ins)
		},
	)
}

// stateLists returns the options of each select and state indicator in s, by node key.
func stateLists(s v13.Schematic) map[string][]v13.StateMapping {
	lists := make(map[string][]v13.StateMapping)
	for _, n := range s.Nodes {
		switch cfg := n.Config.Variant.(type) {
		case v13.SelectNodeConfig:
			lists[n.Key] = cfg.Options
		case v13.StateIndicatorNodeConfig:
			lists[n.Key] = cfg.Options
		}
	}
	return lists
}

// internStates gives each select and state indicator in s an enumeration with the
// entries of its list in lists, through intern, and clears its options. A node with no
// list, or with no entries left, gets no enumeration. The enumeration is named for the
// node type and its label, or the schematic name when the label is empty.
//
// An entry with an empty or repeated key gets a new key. A state whose value is not
// finite, or repeats an earlier state, is dropped with a warning.
func internStates(
	ctx context.Context,
	s *Schematic,
	lists map[string][]v13.StateMapping,
	intern symbol.Intern,
	ins alamos.Instrumentation,
) error {
	for i, n := range s.Nodes {
		var (
			kind, label string
			link        func(*enumeration.Key) NodeConfigVariant
		)
		switch cfg := n.Config.Variant.(type) {
		case SelectNodeConfig:
			kind, label = "Select", cfg.Label.Label
			link = func(k *enumeration.Key) NodeConfigVariant {
				cfg.Enumeration, cfg.Options = k, nil
				return cfg
			}
		case StateIndicatorNodeConfig:
			kind, label = "State indicator", cfg.Label.Label
			link = func(k *enumeration.Key) NodeConfigVariant {
				cfg.Enumeration, cfg.Options = k, nil
				return cfg
			}
		default:
			continue
		}
		var key *enumeration.Key
		if entries := stateEntries(*s, n.Key, lists[n.Key], ins); len(entries) > 0 {
			if label = strings.TrimSpace(label); label == "" {
				label = s.Name
			}
			k, err := intern(ctx, kind+": "+label, entries)
			if err != nil {
				return errors.Wrapf(err, "node %s", n.Key)
			}
			key = &k
		}
		s.Nodes[i].Config.Variant = link(key)
	}
	return nil
}

func stateEntries(
	s Schematic,
	node string,
	list []v13.StateMapping,
	ins alamos.Instrumentation,
) []enumeration.Entry {
	entries := make([]enumeration.Entry, 0, len(list))
	keys := make(set.Set[string], len(list))
	values := make(set.Set[float64], len(list))
	drop := func(reason string, o v13.StateMapping) {
		ins.L.Warn(
			"dropping a state whose value "+reason,
			zap.Stringer("schematic", s.Key),
			zap.String("schematic_name", s.Name),
			zap.String("node", node),
			zap.String("state", o.Name),
		)
	}
	for _, o := range list {
		if math.IsNaN(o.Value) || math.IsInf(o.Value, 0) {
			drop("is not finite", o)
			continue
		}
		if values.Contains(o.Value) {
			drop("repeats an earlier state", o)
			continue
		}
		values.Add(o.Value)
		key := o.Key
		if key == "" || keys.Contains(key) {
			key = uuid.New().String()
		}
		keys.Add(key)
		entries = append(entries, enumeration.Entry{
			Key:   key,
			Value: o.Value,
			Name:  o.Name,
			Color: fixed(o.Color),
		})
	}
	return entries
}
