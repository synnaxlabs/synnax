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

// MigrateSchematic lifts a v10 schematic to v11, which stores every color alpha as a
// 0-255 byte.
func MigrateSchematic(ctx context.Context, old v10.Schematic) (Schematic, error) {
	return autoMigrateSchematic(ctx, old)
}

// Migration lifts stored schematics from v10 to v11.
var Migration = migrate.NewEntry("v61_color_alpha_byte", MigrateSchematic)

// MigrateElementConfig lifts a v10 element config to v11.
func MigrateElementConfig(
	ctx context.Context,
	old v10.ElementConfig,
) (ElementConfig, error) {
	return autoMigrateElementConfig(ctx, old)
}

// MigrateStaticSymbolConfig lifts a v10 static symbol config to v11.
func MigrateStaticSymbolConfig(
	ctx context.Context,
	old v10.StaticSymbolConfig,
) (StaticSymbolConfig, error) {
	return autoMigrateStaticSymbolConfig(ctx, old)
}

// MigrateToggleSymbolConfig lifts a v10 toggle symbol config to v11.
func MigrateToggleSymbolConfig(
	ctx context.Context,
	old v10.ToggleSymbolConfig,
) (ToggleSymbolConfig, error) {
	return autoMigrateToggleSymbolConfig(ctx, old)
}

// MigrateStalenessConfig lifts a v10 staleness config to v11.
func MigrateStalenessConfig(
	ctx context.Context,
	old v10.StalenessConfig,
) (StalenessConfig, error) {
	return autoMigrateStalenessConfig(ctx, old)
}

// MigrateStateMapping lifts a v10 state mapping to v11.
func MigrateStateMapping(
	ctx context.Context,
	old v10.StateMapping,
) (StateMapping, error) {
	return autoMigrateStateMapping(ctx, old)
}

// MigrateScaleIndicatorConfig lifts a v10 scale indicator config to v11.
func MigrateScaleIndicatorConfig(
	ctx context.Context,
	old v10.ScaleIndicatorConfig,
) (ScaleIndicatorConfig, error) {
	return autoMigrateScaleIndicatorConfig(ctx, old)
}

// MigrateDummyToggleSymbolConfig lifts a v10 dummy toggle symbol config to v11.
func MigrateDummyToggleSymbolConfig(
	ctx context.Context,
	old v10.DummyToggleSymbolConfig,
) (DummyToggleSymbolConfig, error) {
	return autoMigrateDummyToggleSymbolConfig(ctx, old)
}

// MigrateToggleConfig lifts a v10 toggle config to v11.
func MigrateToggleConfig(
	ctx context.Context,
	old v10.ToggleConfig,
) (ToggleConfig, error) {
	return autoMigrateToggleConfig(ctx, old)
}

// MigrateSegmentedEdgeConfig lifts a v10 segmented edge config to v11.
func MigrateSegmentedEdgeConfig(
	ctx context.Context,
	old v10.SegmentedEdgeConfig,
) (SegmentedEdgeConfig, error) {
	return autoMigrateSegmentedEdgeConfig(ctx, old)
}
