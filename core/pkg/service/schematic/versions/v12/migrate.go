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
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	"github.com/synnaxlabs/x/encoding/orc"
	text "github.com/synnaxlabs/x/text/versions/v0"
)

// MigrateSchematic lifts a v11 schematic into v12. Each edge takes the variant, stroke
// color, and segments of the edge config stored under its key, and the default stroke
// width. An edge with no edge config takes the defaults. An edge config whose key names
// no edge is dropped. Every other config migrates through MigrateNodeConfig.
func MigrateSchematic(ctx context.Context, old v11.Schematic) (Schematic, error) {
	out, err := autoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	edges := make(map[string]Edge, len(old.Configs))
	out.Configs = make(map[string]NodeConfig, len(old.Configs))
	for key, cfg := range old.Configs {
		if e, ok := edgeStyle(cfg); ok {
			edges[key] = e
			continue
		}
		node, err := nodeConfig(cfg)
		if err != nil {
			return Schematic{}, err
		}
		if out.Configs[key], err = MigrateNodeConfig(ctx, node); err != nil {
			return Schematic{}, err
		}
	}
	for i, e := range out.Edges {
		style, ok := edges[e.Key]
		if !ok {
			continue
		}
		out.Edges[i].Variant = style.Variant
		out.Edges[i].StrokeColor = style.StrokeColor
		out.Edges[i].Segments = style.Segments
	}
	return out, nil
}

// MigrateEdge lifts a v11 edge into v12 with the default variant and stroke width.
func MigrateEdge(ctx context.Context, old v11.Edge) (Edge, error) {
	e, err := autoMigrateEdge(ctx, old)
	if err != nil {
		return Edge{}, err
	}
	e.ApplyDefaults()
	return e, nil
}

// edgeStyle returns the style that an edge config of cfg gives an edge, and false when
// cfg is not an edge config.
func edgeStyle(cfg v11.ElementConfig) (Edge, bool) {
	var (
		variant EdgeVariant
		seg     v11.SegmentedEdgeConfig
	)
	switch v := cfg.Variant.(type) {
	case v11.PipeElementConfig:
		variant, seg = EdgeVariantPipe, v.SegmentedEdgeConfig
	case v11.ElectricElementConfig:
		variant, seg = EdgeVariantElectric, v.SegmentedEdgeConfig
	case v11.SecondaryElementConfig:
		variant, seg = EdgeVariantSecondary, v.SegmentedEdgeConfig
	case v11.JacketedElementConfig:
		variant, seg = EdgeVariantJacketed, v.SegmentedEdgeConfig
	case v11.HydraulicElementConfig:
		variant, seg = EdgeVariantHydraulic, v.SegmentedEdgeConfig
	case v11.PneumaticElementConfig:
		variant, seg = EdgeVariantPneumatic, v.SegmentedEdgeConfig
	case v11.DataElementConfig:
		variant, seg = EdgeVariantData, v.SegmentedEdgeConfig
	default:
		return Edge{}, false
	}
	return Edge{
		Variant:     variant,
		StrokeColor: seg.StrokeColor,
		Segments:    seg.Segments,
	}, true
}

// nodeConfig converts a v11 element config that is not an edge config to the v11 node
// config of the same variant. Both unions encode a node variant with the same bytes.
func nodeConfig(cfg v11.ElementConfig) (v11.NodeConfig, error) {
	if cfg.Variant == nil {
		return v11.NodeConfig{}, nil
	}
	w := orc.NewWriter(0)
	if err := cfg.EncodeOrc(w); err != nil {
		return v11.NodeConfig{}, err
	}
	r := orc.NewReader(nil)
	r.ResetBytes(w.Bytes())
	var node v11.NodeConfig
	if err := node.DecodeOrc(r); err != nil {
		return v11.NodeConfig{}, err
	}
	return node, nil
}

// MigrateNodeConfig lifts a v11 node config into v12. A stroke width rounds to whole
// pixels up to 20. A width above 0, and every line width, keeps at least 1 pixel. A NaN
// width counts as 0. A tank's precision migrates as in MigrateNumericTelemConfig. A
// button, input, select, setpoint, or state indicator keeps its rendered text size.
func MigrateNodeConfig(ctx context.Context, old v11.NodeConfig) (NodeConfig, error) {
	out, err := autoMigrateNodeConfig(ctx, old)
	if err != nil {
		return NodeConfig{}, err
	}
	switch o := old.Variant.(type) {
	case v11.BoxNodeConfig:
		n := out.Variant.(BoxNodeConfig)
		n.StrokeWidth = migrateStrokeWidth(o.StrokeWidth)
		out.Variant = n
	case v11.ButtonNodeConfig:
		n := out.Variant.(ButtonNodeConfig)
		n.Level = renderedLevel(o.Size)
		if o.Level != nil && o.Level.IsValid() {
			n.Level = *o.Level
		}
		out.Variant = n
	case v11.CircleNodeConfig:
		n := out.Variant.(CircleNodeConfig)
		n.StrokeWidth = migrateStrokeWidth(o.StrokeWidth)
		out.Variant = n
	case v11.InputNodeConfig:
		n := out.Variant.(InputNodeConfig)
		n.Level = renderedLevel(o.Size)
		out.Variant = n
	case v11.LineNodeConfig:
		n := out.Variant.(LineNodeConfig)
		n.StrokeWidth = max(migrateStrokeWidth(o.StrokeWidth), 1)
		out.Variant = n
	case v11.PolygonNodeConfig:
		n := out.Variant.(PolygonNodeConfig)
		n.StrokeWidth = migrateStrokeWidth(o.StrokeWidth)
		out.Variant = n
	case v11.SelectNodeConfig:
		n := out.Variant.(SelectNodeConfig)
		n.Level = renderedLevel(o.Size)
		out.Variant = n
	case v11.SetpointNodeConfig:
		n := out.Variant.(SetpointNodeConfig)
		n.Level = renderedLevel(o.Size)
		out.Variant = n
	case v11.StateIndicatorNodeConfig:
		n := out.Variant.(StateIndicatorNodeConfig)
		n.Level = indicatorLevel(o.Size)
		out.Variant = n
	case v11.TankNodeConfig:
		// A tank declares its numeric fields inline, so the generated copy skips the
		// precision whose type changed.
		n := out.Variant.(TankNodeConfig)
		n.Precision = migratePrecision(o.Precision)
		out.Variant = n
	}
	return out, nil
}

// Migration lifts stored schematics from v11 to v12. It stores the precision, stroke
// widths, and label wrap widths as integers, sets the text level of each field symbol,
// and moves the style of each edge onto the edge.
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

// MigrateLabeledConfig lifts a v11 labeled config into v12.
func MigrateLabeledConfig(
	ctx context.Context,
	old v11.LabeledConfig,
) (LabeledConfig, error) {
	return autoMigrateLabeledConfig(ctx, old)
}

// MigrateToggleConfig lifts a v11 toggle config into v12.
func MigrateToggleConfig(
	ctx context.Context,
	old v11.ToggleConfig,
) (ToggleConfig, error) {
	return autoMigrateToggleConfig(ctx, old)
}

// MigrateLabelConfig lifts a v11 label config into v12. It rounds the wrap width to
// whole pixels and clamps it to 1 through 65535. A NaN width, or one of 0 or less,
// becomes 150.
func MigrateLabelConfig(ctx context.Context, old v11.LabelConfig) (LabelConfig, error) {
	cfg, err := autoMigrateLabelConfig(ctx, old)
	if err != nil {
		return LabelConfig{}, err
	}
	cfg.MaxInlineSize = migrateMaxInlineSize(old.MaxInlineSize)
	return cfg, nil
}

const defaultMaxInlineSize = 150

func migrateMaxInlineSize(w float64) uint16 {
	if math.IsNaN(w) || w <= 0 {
		return defaultMaxInlineSize
	}
	return uint16(min(max(math.Round(w), 1), math.MaxUint16))
}

// renderedLevel returns the text level a v11 symbol of the given size rendered at when
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

// indicatorLevel returns the text level a v11 state indicator of the given size
// rendered at.
func indicatorLevel(size ComponentSize) text.Level {
	switch size {
	case ComponentSizeTiny:
		return text.LevelSmall
	case ComponentSizeSmall:
		return text.LevelH5
	case ComponentSizeLarge:
		return text.LevelH3
	case ComponentSizeHuge:
		return text.LevelH2
	default:
		return text.LevelH4
	}
}
