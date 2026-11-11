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
	"math"

	"github.com/synnaxlabs/gorp/migrate"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
)

// MigrateSchematic lifts a v11 schematic into v12. It keeps the precision of each
// numeric symbol as a whole number.
func MigrateSchematic(ctx context.Context, old v11.Schematic) (Schematic, error) {
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

// MigrateElementConfig lifts a v11 element config into v12. A stroke width rounds to
// whole pixels up to 20. A width above 0, and every line width, keeps at least 1 pixel.
// A NaN width counts as 0.
func MigrateElementConfig(
	ctx context.Context,
	old v11.ElementConfig,
) (ElementConfig, error) {
	out, err := autoMigrateElementConfig(ctx, old)
	if err != nil {
		return ElementConfig{}, err
	}
	switch o := old.Variant.(type) {
	case v10.BoxElementConfig:
		n := out.Variant.(BoxElementConfig)
		n.StrokeWidth = migrateStrokeWidth(o.StrokeWidth)
		out.Variant = n
	case v10.CircleElementConfig:
		n := out.Variant.(CircleElementConfig)
		n.StrokeWidth = migrateStrokeWidth(o.StrokeWidth)
		out.Variant = n
	case v10.LineElementConfig:
		n := out.Variant.(LineElementConfig)
		n.StrokeWidth = max(migrateStrokeWidth(o.StrokeWidth), 1)
		out.Variant = n
	case v10.PolygonElementConfig:
		n := out.Variant.(PolygonElementConfig)
		n.StrokeWidth = migrateStrokeWidth(o.StrokeWidth)
		out.Variant = n
	}
	return out, nil
}

// Migration lifts stored schematics from v11 to v12, which stores the precision and
// stroke widths as bytes.
var Migration = migrate.NewEntry("v62_byte_precision", MigrateSchematic)

// MigrateNumericTelemConfig lifts a v11 numeric telemetry config into v12. It rounds
// the precision to a whole number and clamps it to 0 through 17. A NaN precision
// becomes 0.
func MigrateNumericTelemConfig(
	ctx context.Context,
	old v11.NumericTelemConfig,
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

func migrateStrokeWidth(w float64) uint8 {
	if math.IsNaN(w) || w <= 0 {
		return 0
	}
	return uint8(min(max(math.Round(w), 1), 20))
}

// MigrateSegmentedEdgeConfig lifts a v11 edge config to v12, which fills the default
// stroke width.
func MigrateSegmentedEdgeConfig(
	ctx context.Context,
	old v11.SegmentedEdgeConfig,
) (SegmentedEdgeConfig, error) {
	out, err := autoMigrateSegmentedEdgeConfig(ctx, old)
	if err != nil {
		return SegmentedEdgeConfig{}, err
	}
	out.ApplyDefaults()
	return out, nil
}
