// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v12

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
)

// MigrateSchematic lifts a v11 schematic to v12, which stores every color alpha as a
// 0-255 byte.
func MigrateSchematic(ctx context.Context, old v11.Schematic) (Schematic, error) {
	return autoMigrateSchematic(ctx, old)
}

// Migration lifts stored schematics from v11 to v12.
var Migration = migrate.NewEntry("v61_color_alpha_byte", MigrateSchematic)

// MigrateElementConfig lifts a v11 element config to v12.
func MigrateElementConfig(
	ctx context.Context,
	old v11.ElementConfig,
) (ElementConfig, error) {
	return autoMigrateElementConfig(ctx, old)
}

// MigrateStaticSymbolConfig lifts a v11 static symbol config to v12.
func MigrateStaticSymbolConfig(
	ctx context.Context,
	old v11.StaticSymbolConfig,
) (StaticSymbolConfig, error) {
	return autoMigrateStaticSymbolConfig(ctx, old)
}

// MigrateToggleSymbolConfig lifts a v11 toggle symbol config to v12.
func MigrateToggleSymbolConfig(
	ctx context.Context,
	old v11.ToggleSymbolConfig,
) (ToggleSymbolConfig, error) {
	return autoMigrateToggleSymbolConfig(ctx, old)
}

// MigrateStalenessConfig lifts a v11 staleness config to v12.
func MigrateStalenessConfig(
	ctx context.Context,
	old v11.StalenessConfig,
) (StalenessConfig, error) {
	return autoMigrateStalenessConfig(ctx, old)
}

// MigrateStateMapping lifts a v11 state mapping to v12.
func MigrateStateMapping(
	ctx context.Context,
	old v11.StateMapping,
) (StateMapping, error) {
	return autoMigrateStateMapping(ctx, old)
}

// MigrateScaleIndicatorConfig lifts a v11 scale indicator config to v12.
func MigrateScaleIndicatorConfig(
	ctx context.Context,
	old v11.ScaleIndicatorConfig,
) (ScaleIndicatorConfig, error) {
	return autoMigrateScaleIndicatorConfig(ctx, old)
}

// MigrateDummyToggleSymbolConfig lifts a v11 dummy toggle symbol config to v12.
func MigrateDummyToggleSymbolConfig(
	ctx context.Context,
	old v11.DummyToggleSymbolConfig,
) (DummyToggleSymbolConfig, error) {
	return autoMigrateDummyToggleSymbolConfig(ctx, old)
}

// MigrateToggleConfig lifts a v11 toggle config to v12.
func MigrateToggleConfig(
	ctx context.Context,
	old v11.ToggleConfig,
) (ToggleConfig, error) {
	return autoMigrateToggleConfig(ctx, old)
}

// MigrateSegmentedEdgeConfig lifts a v11 segmented edge config to v12.
func MigrateSegmentedEdgeConfig(
	ctx context.Context,
	old v11.SegmentedEdgeConfig,
) (SegmentedEdgeConfig, error) {
	return autoMigrateSegmentedEdgeConfig(ctx, old)
}
