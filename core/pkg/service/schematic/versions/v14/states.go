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
// MigrateSchematic does, then gives their selects and state indicators enumerations as
// InternStates does. Each symbol that a custom_actuator node draws gets the commands of
// an actuator. A symbol node whose custom symbol is gone gets a missing ref, and a node
// drops each state binding that its symbol lacks. The stored symbols must be v5.
func NewMigration(cfg MigrationConfig) gorp.Migration {
	const key = "v63_color_ref"
	return migrate.New(
		key,
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			intern := cfg.Intern(tx)
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
					if err = InternStates(ctx, &s, old, intern, ins); err != nil {
						return Schematic{}, err
					}
					if err = actuate(ctx, tx, old); err != nil {
						return Schematic{}, err
					}
					if err = resolveSymbols(ctx, tx, &s); err != nil {
						return Schematic{}, err
					}
					warnEmptyThresholds(old, ins)
					return s, nil
				},
			).Run(ctx, tx, ins)
		},
	)
}

// stateList returns the options of n when n is a select or a state indicator.
func stateList(n v13.Node) ([]v13.StateMapping, bool) {
	switch cfg := n.Config.Variant.(type) {
	case v13.SelectNodeConfig:
		return cfg.Options, true
	case v13.StateIndicatorNodeConfig:
		return cfg.Options, true
	}
	return nil, false
}

// InternStates gives each select and state indicator of s an enumeration with the
// entries of its options in old, through intern, and clears its options. s must be old
// lifted to v14, which keeps each node at its index. A node with no entries left gets
// no enumeration. The enumeration is named for the node type and its label, or the
// schematic name when the label is empty.
//
// An entry with an empty or repeated key gets a new key. A state whose value is not
// finite, or repeats an earlier state, is dropped with a warning.
func InternStates(
	ctx context.Context,
	s *Schematic,
	old v13.Schematic,
	intern symbol.Intern,
	ins alamos.Instrumentation,
) error {
	if len(s.Nodes) != len(old.Nodes) {
		return errors.Newf(
			"schematic has %d nodes, but its v13 shape has %d",
			len(s.Nodes),
			len(old.Nodes),
		)
	}
	for i, n := range s.Nodes {
		list, ok := stateList(old.Nodes[i])
		if !ok {
			continue
		}
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
		if entries := stateEntries(*s, n.Key, list, ins); len(entries) > 0 {
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
