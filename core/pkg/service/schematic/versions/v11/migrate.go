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

// MigrateSchematic lifts a v10 schematic into the v11 shape. An edge carries no stroke
// width, so it draws at the default. A field symbol carries no text level, so it keeps
// its rendered size.
func MigrateSchematic(ctx context.Context, old v10.Schematic) (Schematic, error) {
	return autoMigrateSchematic(ctx, old)
}

// MigrateSegmentedEdgeConfig lifts a v10 edge config to v11 and leaves the stroke width
// absent.
func MigrateSegmentedEdgeConfig(
	ctx context.Context,
	old v10.SegmentedEdgeConfig,
) (SegmentedEdgeConfig, error) {
	return autoMigrateSegmentedEdgeConfig(ctx, old)
}

// MigrateElementConfig lifts a v10 element config to v11. A button loses its label wrap
// width, because a button rendered without wrapping whatever width it stored.
func MigrateElementConfig(
	ctx context.Context,
	old v10.ElementConfig,
) (ElementConfig, error) {
	cfg, err := autoMigrateElementConfig(ctx, old)
	if err != nil {
		return ElementConfig{}, err
	}
	if button, ok := cfg.Variant.(ButtonElementConfig); ok {
		button.Label.MaxInlineSize = nil
		cfg.Variant = button
	}
	return cfg, nil
}

// MigrateLabelConfig lifts a v10 label config to v11 and keeps its wrap width.
func MigrateLabelConfig(ctx context.Context, old v10.LabelConfig) (LabelConfig, error) {
	cfg, err := autoMigrateLabelConfig(ctx, old)
	if err != nil {
		return LabelConfig{}, err
	}
	cfg.MaxInlineSize = &old.MaxInlineSize
	return cfg, nil
}

// MigrateLabeledConfig lifts a v10 labeled config to v11.
func MigrateLabeledConfig(
	ctx context.Context,
	old v10.LabeledConfig,
) (LabeledConfig, error) {
	return autoMigrateLabeledConfig(ctx, old)
}

// MigrateToggleConfig lifts a v10 toggle config to v11.
func MigrateToggleConfig(
	ctx context.Context,
	old v10.ToggleConfig,
) (ToggleConfig, error) {
	return autoMigrateToggleConfig(ctx, old)
}

// Migration lifts stored schematics from v10 to v11, which adds a stroke width to every
// segmented edge, a text level to every field symbol, and an optional label wrap width.
var Migration = migrate.NewEntry("v61_add_edge_stroke_width", MigrateSchematic)
