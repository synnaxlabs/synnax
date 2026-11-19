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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp/migrate"
	symbolv4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
)

// MigrateSchematic lifts a v13 schematic into v14. No select or state indicator names
// an
// enumeration yet.
func MigrateSchematic(ctx context.Context, old v13.Schematic) (Schematic, error) {
	return autoMigrateSchematic(ctx, old)
}

// MigrateNode lifts a v13 node into v14.
func MigrateNode(ctx context.Context, old v13.Node) (Node, error) {
	return autoMigrateNode(ctx, old)
}

// MigrateNodeConfig lifts a v13 node config into v14. A select or a state indicator
// keeps its options and names no enumeration. A custom node keeps its state overrides.
func MigrateNodeConfig(ctx context.Context, old v13.NodeConfig) (NodeConfig, error) {
	cfg, err := autoMigrateNodeConfig(ctx, old)
	if err != nil {
		return NodeConfig{}, err
	}
	switch o := old.Variant.(type) {
	case v13.CustomActuatorNodeConfig:
		n := cfg.Variant.(CustomActuatorNodeConfig)
		n.StateOverrides = migrateStateOverrides(o.StateOverrides)
		cfg.Variant = n
	case v13.CustomStaticNodeConfig:
		n := cfg.Variant.(CustomStaticNodeConfig)
		n.StateOverrides = migrateStateOverrides(o.StateOverrides)
		cfg.Variant = n
	}
	return cfg, nil
}

func migrateStateOverrides(old []symbolv4.State) []symbol.StateOverride {
	return lo.Map(old, func(s symbolv4.State, _ int) symbol.StateOverride {
		return symbol.StateOverride{Key: s.Key, Name: s.Name, Regions: s.Regions}
	})
}

// Migration lifts stored schematics from v13 to v14.
var Migration = migrate.NewEntry("v63_add_enumeration", MigrateSchematic)
