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

	"github.com/synnaxlabs/gorp/migrate"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
)

// MigrateSchematic lifts an imported v10 schematic to v11. An imported file can omit
// any field, so the lift fills the v10 defaults before those fields become optional.
func MigrateSchematic(ctx context.Context, old v10.Schematic) (Schematic, error) {
	old.ApplyDefaults()
	return autoMigrateSchematic(ctx, old)
}

// migrateStored lifts a stored v10 schematic to v11, keeping every value, 0 included.
func migrateStored(ctx context.Context, old v10.Schematic) (Schematic, error) {
	return autoMigrateSchematic(ctx, old)
}

// MigrateElementConfig lifts a v10 element config to v11. Every field that v11 makes
// optional keeps its stored value.
func MigrateElementConfig(
	ctx context.Context,
	old v10.ElementConfig,
) (ElementConfig, error) {
	out, err := autoMigrateElementConfig(ctx, old)
	if err != nil {
		return ElementConfig{}, err
	}
	switch o := old.Variant.(type) {
	case v10.BoxElementConfig:
		n := out.Variant.(BoxElementConfig)
		n.BorderRadius = new(o.BorderRadius)
		n.StrokeWidth = new(o.StrokeWidth)
		out.Variant = n
	case v10.CircleElementConfig:
		n := out.Variant.(CircleElementConfig)
		n.StrokeWidth = new(o.StrokeWidth)
		out.Variant = n
	case v10.GaugeElementConfig:
		n := out.Variant.(GaugeElementConfig)
		n.Bounds = new(o.Bounds)
		out.Variant = n
	case v10.LineElementConfig:
		n := out.Variant.(LineElementConfig)
		n.End = new(o.End)
		n.StrokeWidth = new(o.StrokeWidth)
		out.Variant = n
	case v10.PolygonElementConfig:
		n := out.Variant.(PolygonElementConfig)
		n.StrokeWidth = new(o.StrokeWidth)
		out.Variant = n
	case v10.TankElementConfig:
		n := out.Variant.(TankElementConfig)
		n.BorderRadius = new(o.BorderRadius)
		n.Bounds = new(o.Bounds)
		n.Precision = migratePrecision(o.Precision)
		out.Variant = n
	}
	return out, nil
}

// MigrateNumericTelemConfig lifts a v10 numeric telemetry config to v11, keeping its
// precision.
func MigrateNumericTelemConfig(
	ctx context.Context,
	old v10.NumericTelemConfig,
) (NumericTelemConfig, error) {
	out, err := autoMigrateNumericTelemConfig(ctx, old)
	if err != nil {
		return NumericTelemConfig{}, err
	}
	out.Precision = migratePrecision(old.Precision)
	return out, nil
}

// migratePrecision rounds a stored decimal place count into the byte that v11 holds.
func migratePrecision(p float64) *uint8 {
	return new(uint8(min(max(math.Round(p), 0), math.MaxUint8)))
}

// MigrateScaleIndicatorConfig lifts a v10 scale indicator config to v11, keeping its
// bounds.
func MigrateScaleIndicatorConfig(
	ctx context.Context,
	old v10.ScaleIndicatorConfig,
) (ScaleIndicatorConfig, error) {
	out, err := autoMigrateScaleIndicatorConfig(ctx, old)
	if err != nil {
		return ScaleIndicatorConfig{}, err
	}
	out.Bounds = new(old.Bounds)
	return out, nil
}

// Migration lifts stored schematics from v10 to v11.
var Migration = migrate.NewEntry("v61_optional_zero_valid_configs", migrateStored)
