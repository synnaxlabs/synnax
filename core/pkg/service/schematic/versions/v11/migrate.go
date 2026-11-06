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

	"github.com/synnaxlabs/gorp/migrate"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
)

// MigrateSchematic lifts a v10 schematic into the v11 shape.
func MigrateSchematic(ctx context.Context, old v10.Schematic) (Schematic, error) {
	return autoMigrateSchematic(ctx, old)
}

// MigrateSegmentedEdgeConfig lifts a v10 edge config to v11, which fills the default
// stroke width.
func MigrateSegmentedEdgeConfig(
	ctx context.Context,
	old v10.SegmentedEdgeConfig,
) (SegmentedEdgeConfig, error) {
	out, err := autoMigrateSegmentedEdgeConfig(ctx, old)
	if err != nil {
		return SegmentedEdgeConfig{}, err
	}
	out.ApplyDefaults()
	return out, nil
}

// Migration lifts stored schematics from v10 to v11, which adds a stroke width to every
// segmented edge.
var Migration = migrate.NewEntry("v61_add_edge_stroke_width", MigrateSchematic)
