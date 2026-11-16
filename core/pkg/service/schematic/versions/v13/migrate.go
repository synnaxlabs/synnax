// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v13

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"reflect"
	"slices"

	"github.com/synnaxlabs/gorp/migrate"
	v12 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v12"
	color "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// MigrateSchematic lifts a v12 schematic to v13. Every color alpha becomes a 0-255
// byte, and a zero color in a field that v12 read as "the theme picks the color"
// becomes absent. Each entry of the configs map moves onto the node or edge it
// configures, and each member of a group box gets the group box as its parent. A node
// with no node config draws nothing, so it is dropped, and so is an edge whose source
// or target is dropped. An edge with no edge config gets the default pipe, which the
// Console drew for it before v0.55. Stroke widths become whole pixels as in
// MigrateNodeConfig and MigrateSegmentedEdgeConfig. A NaN or infinite number in a
// config becomes the schema default of its field.
func MigrateSchematic(ctx context.Context, old v12.Schematic) (Schematic, error) {
	s, err := autoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	kept := make(set.Set[string], len(s.Nodes))
	var groups []groupBox
	nodes := s.Nodes[:0]
	for _, n := range s.Nodes {
		var cfg v12.NodeConfig
		ok, err := convert[v12.EdgeConfig](old.Configs, n.Key, &cfg)
		if err != nil {
			return Schematic{}, err
		}
		if !ok {
			continue
		}
		if n.Config, err = MigrateNodeConfig(ctx, cfg); err != nil {
			return Schematic{}, err
		}
		if g, isGroup := cfg.Variant.(v12.GroupBoxNodeConfig); isGroup {
			groups = append(groups, groupBox{key: n.Key, members: g.Members})
		}
		nodes = append(nodes, n)
		kept.Add(n.Key)
	}
	setParents(nodes, groups)
	s.Nodes = nodes
	edges := s.Edges[:0]
	for _, e := range s.Edges {
		if !kept.Contains(e.Source.Node) || !kept.Contains(e.Target.Node) {
			continue
		}
		var cfg v12.EdgeConfig
		ok, err := convert[v12.NodeConfig](old.Configs, e.Key, &cfg)
		if err != nil {
			return Schematic{}, err
		}
		if !ok {
			e.Config = EdgeConfig{
				Variant: PipeEdgeConfig{
					SegmentedEdgeConfig: SegmentedEdgeConfig{
						Segments:    []Segment{},
						StrokeWidth: defaultStrokeWidth,
					},
				},
			}
		} else if e.Config, err = MigrateEdgeConfig(ctx, cfg); err != nil {
			return Schematic{}, err
		}
		edges = append(edges, e)
	}
	s.Edges = edges
	return s, nil
}

type groupBox struct {
	key     string
	members []string
}

// setParents sets the parent of each node that the members of a group list. The first
// group in node order wins, and a link that would close a cycle is skipped.
func setParents(nodes []Node, groups []groupBox) {
	indexOf := make(map[string]int, len(nodes))
	for i, n := range nodes {
		indexOf[n.Key] = i
	}
	closesCycle := func(box, member string) bool {
		for p := &box; p != nil; p = nodes[indexOf[*p]].Parent {
			if *p == member {
				return true
			}
		}
		return false
	}
	for _, g := range groups {
		for _, m := range g.members {
			i, ok := indexOf[m]
			if !ok || nodes[i].Parent != nil || closesCycle(g.key, m) {
				continue
			}
			nodes[i].Parent = &g.key
		}
	}
}

// MigrateNode copies the node. MigrateSchematic sets its config from the configs map.
func MigrateNode(ctx context.Context, old v12.Node) (Node, error) {
	return autoMigrateNode(ctx, old)
}

// MigrateEdge copies the edge. MigrateSchematic sets its config from the configs map.
func MigrateEdge(ctx context.Context, old v12.Edge) (Edge, error) {
	return autoMigrateEdge(ctx, old)
}

// nonFiniteAbsent makes a NaN or infinite number, which JSON cannot hold, read as an
// absent field: it encodes as null, and a null leaves the schema default in place.
var nonFiniteAbsent = json.JoinOptions(
	json.WithMarshalers(json.MarshalToFunc(
		func(enc *jsontext.Encoder, f float64) error {
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return enc.WriteToken(jsontext.Null)
			}
			return errors.ErrUnsupported
		},
	)),
	json.WithUnmarshalers(json.UnmarshalFromFunc(
		func(dec *jsontext.Decoder, _ *float64) error {
			if dec.PeekKind() != jsontext.KindNull {
				return errors.ErrUnsupported
			}
			return dec.SkipValue()
		},
	)),
)

// convert decodes the config stored under key into dst. It returns false when no config
// is stored under key, or when the stored config is of the Other element kind.
func convert[Other, C any](
	configs map[string]v12.ElementConfig,
	key string,
	dst *C,
) (bool, error) {
	cfg, ok := configs[key]
	if !ok {
		return false, nil
	}
	b, err := json.Marshal(cfg, nonFiniteAbsent)
	if err != nil {
		return false, err
	}
	if err = json.Unmarshal(b, dst, nonFiniteAbsent); err == nil {
		return true, nil
	}
	var other Other
	if json.Unmarshal(b, &other, nonFiniteAbsent) == nil {
		return false, nil
	}
	return false, err
}

// themeColors names, for each struct, the optional colors whose zero color v12 read as
// "the theme picks the color".
var themeColors = map[reflect.Type][]string{
	reflect.TypeFor[StalenessConfig](): {"StalenessColor"},
	reflect.TypeFor[TankNodeConfig](): {
		"StalenessColor", "LevelColor", "AxisColor", "TextColor",
	},
	reflect.TypeFor[ScaleIndicatorConfig](): {"LevelColor", "AxisColor", "TextColor"},
	reflect.TypeFor[ValueNodeConfig]():      {"TextColor"},
	reflect.TypeFor[GaugeNodeConfig]():      {"StrokeColor"},
}

// clearThemeColors sets each color under v that themeColors names and that holds the
// zero color to nil. v must be settable.
func clearThemeColors(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			clearThemeColors(v.Elem())
		}
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		c := reflect.New(v.Elem().Type()).Elem()
		c.Set(v.Elem())
		clearThemeColors(c)
		v.Set(c)
	case reflect.Struct:
		names := themeColors[v.Type()]
		for sf, f := range v.Fields() {
			if !f.CanSet() {
				continue
			}
			c, isColor := reflect.TypeAssert[*color.Color](f)
			if isColor && slices.Contains(names, sf.Name) {
				if c != nil && *c == (color.Color{}) {
					f.SetZero()
				}
				continue
			}
			clearThemeColors(f)
		}
	case reflect.Slice:
		for i := range v.Len() {
			clearThemeColors(v.Index(i))
		}
	default:
	}
}

// Migration lifts stored schematics from v12 to v13.
var Migration = migrate.NewEntry("v62_color_bytes_configs_on_nodes", MigrateSchematic)

// MigrateNodeConfig lifts a v12 node config to v13. See MigrateSchematic for colors. A
// stroke width rounds to whole pixels up to 20. A width above 0, and every line width,
// keeps at least 1 pixel. A NaN width counts as 0.
func MigrateNodeConfig(ctx context.Context, old v12.NodeConfig) (NodeConfig, error) {
	cfg, err := autoMigrateNodeConfig(ctx, old)
	if err != nil {
		return NodeConfig{}, err
	}
	switch o := old.Variant.(type) {
	case v12.BoxNodeConfig:
		n := cfg.Variant.(BoxNodeConfig)
		n.StrokeWidth = migrateStrokeWidth(o.StrokeWidth)
		cfg.Variant = n
	case v12.CircleNodeConfig:
		n := cfg.Variant.(CircleNodeConfig)
		n.StrokeWidth = migrateStrokeWidth(o.StrokeWidth)
		cfg.Variant = n
	case v12.LineNodeConfig:
		n := cfg.Variant.(LineNodeConfig)
		n.StrokeWidth = max(migrateStrokeWidth(o.StrokeWidth), 1)
		cfg.Variant = n
	case v12.PolygonNodeConfig:
		n := cfg.Variant.(PolygonNodeConfig)
		n.StrokeWidth = migrateStrokeWidth(o.StrokeWidth)
		cfg.Variant = n
	}
	clearThemeColors(reflect.ValueOf(&cfg).Elem())
	return cfg, nil
}

const (
	defaultStrokeWidth = 2
	maxStrokeWidth     = 20
)

func migrateStrokeWidth(w float64) uint8 {
	if math.IsNaN(w) || w <= 0 {
		return 0
	}
	return uint8(min(max(math.Round(w), 1), maxStrokeWidth))
}

// MigrateEdgeConfig lifts a v12 edge config to v13. See MigrateSchematic for colors.
func MigrateEdgeConfig(ctx context.Context, old v12.EdgeConfig) (EdgeConfig, error) {
	cfg, err := autoMigrateEdgeConfig(ctx, old)
	if err != nil {
		return EdgeConfig{}, err
	}
	clearThemeColors(reflect.ValueOf(&cfg).Elem())
	return cfg, nil
}

// MigrateStaticSymbolConfig lifts a v12 static symbol config to v13.
func MigrateStaticSymbolConfig(
	ctx context.Context,
	old v12.StaticSymbolConfig,
) (StaticSymbolConfig, error) {
	return autoMigrateStaticSymbolConfig(ctx, old)
}

// MigrateToggleSymbolConfig lifts a v12 toggle symbol config to v13.
func MigrateToggleSymbolConfig(
	ctx context.Context,
	old v12.ToggleSymbolConfig,
) (ToggleSymbolConfig, error) {
	return autoMigrateToggleSymbolConfig(ctx, old)
}

// MigrateStalenessConfig lifts a v12 staleness config to v13.
func MigrateStalenessConfig(
	ctx context.Context,
	old v12.StalenessConfig,
) (StalenessConfig, error) {
	return autoMigrateStalenessConfig(ctx, old)
}

// MigrateStateMapping lifts a v12 state mapping to v13.
func MigrateStateMapping(
	ctx context.Context,
	old v12.StateMapping,
) (StateMapping, error) {
	return autoMigrateStateMapping(ctx, old)
}

// MigrateScaleIndicatorConfig lifts a v12 scale indicator config to v13.
func MigrateScaleIndicatorConfig(
	ctx context.Context,
	old v12.ScaleIndicatorConfig,
) (ScaleIndicatorConfig, error) {
	return autoMigrateScaleIndicatorConfig(ctx, old)
}

// MigrateDummyToggleSymbolConfig lifts a v12 dummy toggle symbol config to v13.
func MigrateDummyToggleSymbolConfig(
	ctx context.Context,
	old v12.DummyToggleSymbolConfig,
) (DummyToggleSymbolConfig, error) {
	return autoMigrateDummyToggleSymbolConfig(ctx, old)
}

// MigrateToggleConfig lifts a v12 toggle config to v13.
func MigrateToggleConfig(
	ctx context.Context,
	old v12.ToggleConfig,
) (ToggleConfig, error) {
	return autoMigrateToggleConfig(ctx, old)
}

// MigrateSegmentedEdgeConfig lifts a v12 segmented edge config to v13 with the default
// stroke width.
func MigrateSegmentedEdgeConfig(
	ctx context.Context,
	old v12.SegmentedEdgeConfig,
) (SegmentedEdgeConfig, error) {
	cfg, err := autoMigrateSegmentedEdgeConfig(ctx, old)
	if err != nil {
		return SegmentedEdgeConfig{}, err
	}
	cfg.StrokeWidth = defaultStrokeWidth
	return cfg, nil
}
