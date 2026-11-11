// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v11

import (
	"context"
	"math"
	"uuid"

	"github.com/synnaxlabs/gorp/migrate"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
)

// MigrateSchematic lifts a v10 schematic into v11. The schematic sits in no group.
func MigrateSchematic(ctx context.Context, old v10.Schematic) (Schematic, error) {
	out, err := autoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	// A tank declares its numeric fields inline, so the generated copy skips the
	// precision whose type changed.
	for k, cfg := range old.Configs {
		if o, ok := cfg.Variant.(v10.TankElementConfig); ok {
			n := out.Configs[k].Variant.(TankElementConfig)
			n.Precision = migratePrecision(o.Precision)
			out.Configs[k] = ElementConfig{Variant: n}
		}
	}
	return out, nil
}

var Migration = migrate.NewEntry("v61_add_group", MigrateSchematic)

// MigrateNumericTelemConfig lifts a v10 numeric telemetry config into v11. It rounds
// the precision to a whole number and clamps it to 0 through 17. A NaN precision
// becomes 0.
func MigrateNumericTelemConfig(
	ctx context.Context,
	old v10.NumericTelemConfig,
) (NumericTelemConfig, error) {
	cfg, err := autoMigrateNumericTelemConfig(ctx, old)
	if err != nil {
		return NumericTelemConfig{}, err
	}
	cfg.Precision = migratePrecision(old.Precision)
	return cfg, nil
}

// maxPrecision is the most significant digits a float64 holds.
const maxPrecision = 17

func migratePrecision(p float64) *uint8 {
	if math.IsNaN(p) {
		return new(uint8(0))
	}
	return new(uint8(min(max(math.Round(p), 0), maxPrecision)))
}

// MembersMigration sets the group of each stored schematic from the ontology edge that
// placed it in a group.
var MembersMigration = groupv1.NewMembersMigration(
	groupv1.MembersConfig[Key, Schematic]{
		Name:         "v61_schematic_group_membership",
		Type:         resource.TypeSchematic,
		SetGroup:     func(e *Schematic, g groupv1.Key) { e.Group = &g },
		ProjectOwned: true,
	},
)

// OwnersMigration sets the project or range of each stored schematic from the ontology
// edge that owned it, and deletes each schematic with no owner.
var OwnersMigration = projectv3.NewOwnersMigration(
	projectv3.OwnersConfig[Key, Schematic]{
		Name:     "v61_schematic_owner",
		Type:     resource.TypeSchematic,
		Owned:    func(e Schematic) bool { return e.Project != nil || e.Range != nil },
		NameOf:   func(e Schematic) string { return e.Name },
		SetOwner: func(e *Schematic, p, r *uuid.UUID) { e.Project, e.Range = p, r },
	},
)
