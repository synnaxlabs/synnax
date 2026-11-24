// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v14

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp/migrate"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	symbolv4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
)

// MigrateSchematic lifts a v13 schematic to v14, which stores each stroke, fill, text,
// level, and axis color of an edge, shape, or readout as a color reference. Each such
// color becomes a fixed color, and an absent color stays absent. It drops the staleness
// color.
func MigrateSchematic(ctx context.Context, old v13.Schematic) (Schematic, error) {
	return autoMigrateSchematic(ctx, old)
}

// Migration lifts stored schematics from v13 to v14.
var Migration = migrate.NewEntry("v63_color_ref", MigrateSchematic)

// MigrateNode lifts a v13 node to v14.
func MigrateNode(ctx context.Context, old v13.Node) (Node, error) {
	return autoMigrateNode(ctx, old)
}

// MigrateEdge lifts a v13 edge to v14.
func MigrateEdge(ctx context.Context, old v13.Edge) (Edge, error) {
	return autoMigrateEdge(ctx, old)
}

// MigrateSegmentedEdgeConfig lifts a v13 segmented edge config to v14. See
// MigrateSchematic for colors.
func MigrateSegmentedEdgeConfig(
	ctx context.Context,
	old v13.SegmentedEdgeConfig,
) (SegmentedEdgeConfig, error) {
	cfg, err := autoMigrateSegmentedEdgeConfig(ctx, old)
	if err != nil {
		return SegmentedEdgeConfig{}, err
	}
	cfg.StrokeColor = fixed(old.StrokeColor)
	return cfg, nil
}

// MigrateScaleIndicatorConfig lifts a v13 scale indicator config to v14. See
// MigrateSchematic for colors. Its bounds become closed bounds; see closed.
func MigrateScaleIndicatorConfig(
	ctx context.Context,
	old v13.ScaleIndicatorConfig,
) (ScaleIndicatorConfig, error) {
	cfg, err := autoMigrateScaleIndicatorConfig(ctx, old)
	if err != nil {
		return ScaleIndicatorConfig{}, err
	}
	cfg.LevelColor = fixed(old.LevelColor)
	cfg.AxisColor = fixed(old.AxisColor)
	cfg.TextColor = fixed(old.TextColor)
	cfg.Bounds = closed(old.Bounds)
	return cfg, nil
}

// MigrateStalenessConfig lifts a v13 staleness config to v14. It drops the staleness
// color.
func MigrateStalenessConfig(
	ctx context.Context,
	old v13.StalenessConfig,
) (StalenessConfig, error) {
	return autoMigrateStalenessConfig(ctx, old)
}

// MigrateToggleConfig lifts a v13 toggle config to v14. See MigrateStalenessConfig.
func MigrateToggleConfig(
	ctx context.Context,
	old v13.ToggleConfig,
) (ToggleConfig, error) {
	return autoMigrateToggleConfig(ctx, old)
}

// MigrateNodeConfig lifts a v13 node config to v14. See MigrateSchematic for colors. A
// custom node keeps its state overrides. The bounds of a gauge or tank become closed
// bounds; see closed.
func MigrateNodeConfig(ctx context.Context, old v13.NodeConfig) (NodeConfig, error) {
	cfg, err := autoMigrateNodeConfig(ctx, old)
	if err != nil {
		return NodeConfig{}, err
	}
	if n, ok := migrateShapeColors(old.Variant, cfg.Variant); ok {
		cfg.Variant = n
	} else if n, ok = migrateControlColors(old.Variant, cfg.Variant); ok {
		cfg.Variant = n
	} else {
		cfg.Variant = migrateReadoutColors(old.Variant, cfg.Variant)
	}
	cfg.Variant = migrateCustomStates(old.Variant, cfg.Variant)
	return cfg, nil
}

func migrateShapeColors(
	old v13.NodeConfigVariant,
	migrated NodeConfigVariant,
) (NodeConfigVariant, bool) {
	switch o := old.(type) {
	case v13.BoxNodeConfig:
		n := as[BoxNodeConfig](migrated)
		n.StrokeColor, n.FillColor = fixed(o.StrokeColor), fixed(o.FillColor)
		return n, true
	case v13.CircleNodeConfig:
		n := as[CircleNodeConfig](migrated)
		n.StrokeColor, n.FillColor = fixed(o.StrokeColor), fixed(o.FillColor)
		return n, true
	case v13.CylinderNodeConfig:
		n := as[CylinderNodeConfig](migrated)
		n.StrokeColor, n.FillColor = fixed(o.StrokeColor), fixed(o.FillColor)
		return n, true
	case v13.LineNodeConfig:
		n := as[LineNodeConfig](migrated)
		n.StrokeColor = fixed(o.StrokeColor)
		return n, true
	case v13.PolygonNodeConfig:
		n := as[PolygonNodeConfig](migrated)
		n.StrokeColor, n.FillColor = fixed(o.StrokeColor), fixed(o.FillColor)
		return n, true
	case v13.TextBoxNodeConfig:
		n := as[TextBoxNodeConfig](migrated)
		n.TextColor = fixed(o.TextColor)
		return n, true
	case v13.OffPageReferenceNodeConfig:
		n := as[OffPageReferenceNodeConfig](migrated)
		n.FillColor = fixed(o.FillColor)
		return n, true
	}
	return migrated, false
}

func migrateControlColors(
	old v13.NodeConfigVariant,
	migrated NodeConfigVariant,
) (NodeConfigVariant, bool) {
	switch o := old.(type) {
	case v13.ButtonNodeConfig:
		n := as[ButtonNodeConfig](migrated)
		n.FillColor = fixed(o.FillColor)
		return n, true
	case v13.InputNodeConfig:
		n := as[InputNodeConfig](migrated)
		n.FillColor = fixed(o.FillColor)
		return n, true
	case v13.SelectNodeConfig:
		n := as[SelectNodeConfig](migrated)
		n.FillColor = fixed(o.FillColor)
		return n, true
	case v13.SetpointNodeConfig:
		n := as[SetpointNodeConfig](migrated)
		n.FillColor = fixed(o.FillColor)
		return n, true
	}
	return migrated, false
}

func migrateReadoutColors(
	old v13.NodeConfigVariant,
	migrated NodeConfigVariant,
) NodeConfigVariant {
	switch o := old.(type) {
	case v13.StateIndicatorNodeConfig:
		n := as[StateIndicatorNodeConfig](migrated)
		n.StrokeColor = fixed(o.StrokeColor)
		return n
	case v13.TankNodeConfig:
		n := as[TankNodeConfig](migrated)
		n.StrokeColor, n.FillColor = fixed(o.StrokeColor), fixed(o.FillColor)
		n.LevelColor = fixed(o.LevelColor)
		n.AxisColor = fixed(o.AxisColor)
		n.TextColor = fixed(o.TextColor)
		n.Bounds = closed(o.Bounds)
		return n
	case v13.GaugeNodeConfig:
		n := as[GaugeNodeConfig](migrated)
		n.StrokeColor = fixed(o.StrokeColor)
		n.Bounds = closed(o.Bounds)
		return n
	case v13.StringDisplayNodeConfig:
		n := as[StringDisplayNodeConfig](migrated)
		n.StrokeColor, n.TextColor = fixed(o.StrokeColor), fixed(o.TextColor)
		return n
	case v13.ValueNodeConfig:
		n := as[ValueNodeConfig](migrated)
		n.StrokeColor, n.TextColor = fixed(o.StrokeColor), fixed(o.TextColor)
		n.FillColor = fixed(o.FillColor)
		return n
	}
	return migrated
}

func migrateCustomStates(
	old v13.NodeConfigVariant,
	migrated NodeConfigVariant,
) NodeConfigVariant {
	switch o := old.(type) {
	case v13.CustomActuatorNodeConfig:
		n := as[CustomActuatorNodeConfig](migrated)
		n.StateOverrides = migrateStateOverrides(o.StateOverrides)
		return n
	case v13.CustomStaticNodeConfig:
		n := as[CustomStaticNodeConfig](migrated)
		n.StateOverrides = migrateStateOverrides(o.StateOverrides)
		return n
	}
	return migrated
}

func migrateStateOverrides(old []symbolv4.State) []symbol.StateOverride {
	return lo.Map(old, func(s symbolv4.State, _ int) symbol.StateOverride {
		return symbol.StateOverride{Key: s.Key, Name: s.Name, Regions: s.Regions}
	})
}

// as returns a migrated variant as a T. It panics when v is not a T, since a generated
// migration keeps each variant's kind.
func as[T any](v any) T {
	t, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("migrated variant is a %T, not a %T", v, t))
	}
	return t
}

func fixed(c *xcolor.Color) *servicecolor.Ref {
	if c == nil {
		return nil
	}
	return &servicecolor.Ref{Variant: servicecolor.FixedRef{Value: *c}}
}

// closed returns b as closed bounds. Reversed ends swap, and equal ends take the
// default of 0 to 100, since the Core refuses a lower that is not below the upper.
func closed(b spatial.Bounds) spatial.ClosedBounds {
	switch {
	case b.Lower > b.Upper:
		return spatial.ClosedBounds{Lower: b.Upper, Upper: b.Lower}
	case b.Lower == b.Upper:
		return spatial.ClosedBounds{Lower: 0, Upper: 100}
	}
	return spatial.ClosedBounds(b)
}
