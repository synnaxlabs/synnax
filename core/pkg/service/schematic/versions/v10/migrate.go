// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v10

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v9 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v9"
)

// MigrateSchematic lifts a v9 schematic into the v10 shape, filling the schema defaults
// of the fields v10 adds. A field the v9 config carries keeps its value.
func MigrateSchematic(ctx context.Context, old v9.Schematic) (Schematic, error) {
	out, err := autoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	for k, cfg := range old.Configs {
		fields, err := v9.ElementConfigFields(cfg)
		if err != nil {
			return Schematic{}, err
		}
		if out.Configs[k], err = DecodeWithDefaults(fields); err != nil {
			return Schematic{}, err
		}
	}
	return out, nil
}

// Migration lifts stored schematics from v9 to v10, which adds read-back state to the
// setpoint, input, and select symbols.
var Migration = migrate.NewEntry("v60_add_read_back_state", MigrateSchematic)

// MigrateElementConfig lifts a v9 element config to v10. MigrateSchematic decodes each
// stored config again over the v10 defaults.
func MigrateElementConfig(
	ctx context.Context,
	old v9.ElementConfig,
) (ElementConfig, error) {
	return autoMigrateElementConfig(ctx, old)
}
