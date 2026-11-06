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
	text "github.com/synnaxlabs/x/text/versions/v0"
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
// optional keeps its stored value. A button, input, select, or setpoint keeps its
// rendered text size. A button takes a label wrap width of 0, because a button rendered
// without wrapping whatever width it stored.
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
	case v10.ButtonElementConfig:
		n := out.Variant.(ButtonElementConfig)
		n.Label.MaxInlineSize = new(0.0)
		n.Level = renderedLevel(o.Size)
		if o.Level != nil {
			n.Level = *o.Level
		}
		out.Variant = n
	case v10.CircleElementConfig:
		n := out.Variant.(CircleElementConfig)
		n.StrokeWidth = new(o.StrokeWidth)
		out.Variant = n
	case v10.GaugeElementConfig:
		n := out.Variant.(GaugeElementConfig)
		n.Bounds = new(o.Bounds)
		out.Variant = n
	case v10.InputElementConfig:
		n := out.Variant.(InputElementConfig)
		n.Level = renderedLevel(o.Size)
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
	case v10.SelectElementConfig:
		n := out.Variant.(SelectElementConfig)
		n.Level = renderedLevel(o.Size)
		out.Variant = n
	case v10.SetpointElementConfig:
		n := out.Variant.(SetpointElementConfig)
		n.Level = renderedLevel(o.Size)
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

// MigrateLabelConfig lifts a v10 label config to v11 and keeps its wrap width.
func MigrateLabelConfig(ctx context.Context, old v10.LabelConfig) (LabelConfig, error) {
	out, err := autoMigrateLabelConfig(ctx, old)
	if err != nil {
		return LabelConfig{}, err
	}
	out.MaxInlineSize = new(old.MaxInlineSize)
	return out, nil
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

// renderedLevel returns the text level a v10 symbol of the given size rendered at when
// it stored none.
func renderedLevel(size ComponentSize) text.Level {
	switch size {
	case ComponentSizeTiny:
		return text.LevelTiny
	case ComponentSizeSmall:
		return text.LevelSmall
	case ComponentSizeLarge:
		return text.LevelH5
	case ComponentSizeHuge:
		return text.LevelH2
	default:
		return text.LevelP
	}
}

// Migration lifts stored schematics from v10 to v11, which keeps every stored value,
// fills the edge stroke width, and sets the text level of each button, input, select,
// and setpoint.
var Migration = migrate.NewEntry(
	"v61_optional_zero_valid_configs_edge_stroke_width",
	migrateStored,
)
