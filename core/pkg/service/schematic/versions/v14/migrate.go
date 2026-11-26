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

	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/set"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
)

// MigrateSchematic lifts a v13 schematic to v14, which stores each stroke, fill, text,
// level, and axis color of an edge, shape, or readout as a color reference. Each such
// color becomes a fixed color, and an absent color stays absent. It drops the staleness
// color. It turns a switch upright; see uprightSwitches.
func MigrateSchematic(ctx context.Context, old v13.Schematic) (Schematic, error) {
	s, err := autoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	uprightSwitches(&s, old)
	return s, nil
}

// upright maps each orientation that the old switch node did not draw to the one it
// drew.
var upright = map[spatial.OuterLocation]spatial.OuterLocation{
	spatial.OuterLocationRight:  spatial.OuterLocationLeft,
	spatial.OuterLocationBottom: spatial.OuterLocationTop,
}

// uprightSwitches turns a switch that faced right to face left, and one that faced
// bottom to face top, as the old switch node drew it. The old node turned its handles
// with the stored orientation, so each edge end at a turned switch swaps handles 1 and
// 2 to stay where it was drawn.
func uprightSwitches(s *Schematic, old v13.Schematic) {
	turned := set.Set[string]{}
	for i, n := range old.Nodes {
		if _, ok := n.Config.Variant.(v13.SwitchNodeConfig); !ok {
			continue
		}
		cfg := as[SymbolNodeConfig](s.Nodes[i].Config.Variant)
		o, ok := upright[cfg.Orientation]
		if !ok {
			continue
		}
		cfg.Orientation = o
		s.Nodes[i].Config.Variant = cfg
		turned.Add(n.Key)
	}
	for i := range s.Edges {
		swapSwitchHandle(&s.Edges[i].Source, turned)
		swapSwitchHandle(&s.Edges[i].Target, turned)
	}
}

func swapSwitchHandle(h *Handle, turned set.Set[string]) {
	if !turned.Contains(h.Node) {
		return
	}
	switch h.Param {
	case "1":
		h.Param = "2"
	case "2":
		h.Param = "1"
	}
}

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

// MigrateNodeConfig lifts a v13 node config to v14. An equipment node becomes a symbol
// node; see SymbolNode. See MigrateSchematic for colors. The bounds of a gauge or tank
// become closed bounds; see closed.
func MigrateNodeConfig(ctx context.Context, old v13.NodeConfig) (NodeConfig, error) {
	n, ok, err := SymbolNode(old)
	if err != nil {
		return NodeConfig{}, err
	}
	if ok {
		return NodeConfig{Variant: n}, nil
	}
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
