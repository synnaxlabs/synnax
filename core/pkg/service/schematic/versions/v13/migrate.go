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
	"fmt"
	"math"
	"reflect"
	"slices"

	"github.com/synnaxlabs/gorp/migrate"
	v12 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v12"
	colorv0 "github.com/synnaxlabs/x/color/versions/v0"
	color "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

// MigrateSchematic lifts a v12 schematic to v13. Every color alpha becomes a 0-255
// byte, and a zero color in a field that v12 read as "the theme picks the color"
// becomes absent. Each entry of the configs map moves onto the node or edge it
// configures, and each member of a group box gets the group box as its parent. A node
// with no node config draws nothing, so it is dropped, and so is an edge whose source
// or target is dropped. An edge with no edge config gets the default pipe, which the
// Console drew for it before v0.55. Stroke widths become whole pixels as in
// MigrateNodeConfig and MigrateSegmentedEdgeConfig. A NaN or infinite number in a
// config becomes the schema default of its field, and a color with such an alpha
// becomes opaque.
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

func nonFinite(f float64) bool { return math.IsNaN(f) || math.IsInf(f, 0) }

// nonFiniteAbsent makes a NaN or infinite number, which JSON cannot hold, read as an
// absent field: it encodes as null, and a null leaves the schema default in place. A
// color with such an alpha encodes as opaque, so it stays visible.
var nonFiniteAbsent = json.JoinOptions(
	json.WithMarshalers(json.JoinMarshalers(
		json.MarshalToFunc(func(enc *jsontext.Encoder, c colorv0.Color) error {
			if !nonFinite(c.A) {
				return errors.ErrUnsupported
			}
			c.A = 1
			return json.MarshalEncode(enc, c)
		}),
		json.MarshalToFunc(func(enc *jsontext.Encoder, f float64) error {
			if !nonFinite(f) {
				return errors.ErrUnsupported
			}
			return enc.WriteToken(jsontext.Null)
		}),
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
// keeps at least 1 pixel. A NaN line width counts as 0, and a NaN box, circle, or
// polygon width becomes absent. A button label migrates as in MigrateLabelConfig,
// except that a width of 150, which did not wrap in v12, becomes 65535. Other pixel
// sizes round and clamp to 0 through 65535, and a NaN size takes the default. An inline
// size, circle radius, polygon side, or gauge bar width keeps at least 1 pixel. A NaN
// box border radius becomes absent. A polygon keeps 3 through 255 sides. A click delay
// in milliseconds becomes a span. A scale of 0 or less becomes 1. A tank's numeric and
// staleness fields migrate as in MigrateNumericTelemConfig and MigrateStalenessConfig.
// Renamed fields keep their values. An absent channel becomes 0, an absent page links
// to no page, and an absent position, dimensions, or control state takes its default.
func MigrateNodeConfig(ctx context.Context, old v12.NodeConfig) (NodeConfig, error) {
	cfg, err := autoMigrateNodeConfig(ctx, old)
	if err != nil {
		return NodeConfig{}, err
	}
	cfg.Variant = migrateNodeVariant(old.Variant, cfg.Variant)
	clearThemeColors(reflect.ValueOf(&cfg).Elem())
	resetScales(reflect.ValueOf(&cfg).Elem())
	return cfg, nil
}

// migrateNodeVariant applies the hand-written v13 rules for old's variant to migrated,
// the generated copy of it.
func migrateNodeVariant(
	old v12.NodeConfigVariant,
	migrated NodeConfigVariant,
) NodeConfigVariant {
	if n, ok := migrateStaticVariant(old, migrated); ok {
		return n
	}
	if n, ok := migrateControlVariant(old, migrated); ok {
		return n
	}
	return migrateDisplayVariant(old, migrated)
}

func migrateStaticVariant(
	old v12.NodeConfigVariant,
	migrated NodeConfigVariant,
) (NodeConfigVariant, bool) {
	switch o := old.(type) {
	case v12.BoxNodeConfig:
		n := as[BoxNodeConfig](migrated)
		n.StrokeWidth = migrateShapeStrokeWidth(o.StrokeWidth)
		n.BorderRadius = migrateOptionalPixels(o.BorderRadius)
		return n, true
	case v12.CircleNodeConfig:
		n := as[CircleNodeConfig](migrated)
		n.StrokeWidth = migrateShapeStrokeWidth(o.StrokeWidth)
		n.Radius = migrateSize(o.Radius, 20)
		return n, true
	case v12.LineNodeConfig:
		n := as[LineNodeConfig](migrated)
		n.StrokeWidth = max(migrateStrokeWidth(o.StrokeWidth), 1)
		return n, true
	case v12.PolygonNodeConfig:
		n := as[PolygonNodeConfig](migrated)
		n.StrokeWidth = migrateShapeStrokeWidth(o.StrokeWidth)
		n.NumSides = migrateNumSides(o.NumSides)
		n.SideLength = migrateSize(o.SideLength, 20)
		n.BorderRadius = migratePixels(o.CornerRounding, 0)
		return n, true
	case v12.TextBoxNodeConfig:
		n := as[TextBoxNodeConfig](migrated)
		n.InlineSize = migrateSize(o.Width, 75)
		return n, true
	case v12.OffPageReferenceNodeConfig:
		n := as[OffPageReferenceNodeConfig](migrated)
		n.DoubleClickNavigationDisabled = o.DblClickNavDisabled
		n.Page = valueOr(o.Page, Page{Type: PageTypeSchematic})
		return n, true
	}
	return migrated, false
}

func migrateControlVariant(
	old v12.NodeConfigVariant,
	migrated NodeConfigVariant,
) (NodeConfigVariant, bool) {
	switch o := old.(type) {
	case v12.ButtonNodeConfig:
		n := as[ButtonNodeConfig](migrated)
		n.CommandChannel = valueOr(o.CommandChannel, 0)
		n.Control = migrateControl(o.Control)
		n.OnClickDelay = migrateOnClickDelay(o.OnClickDelay)
		n.Label.MaxInlineSize = migrateButtonMaxInlineSize(o.Label.MaxInlineSize)
		return n, true
	case v12.InputNodeConfig:
		n := as[InputNodeConfig](migrated)
		n.CommandChannel = valueOr(o.CommandChannel, 0)
		n.StateChannel = valueOr(o.StateChannel, 0)
		n.Dimensions = valueOr(o.Dimensions, spatial.Dimensions{})
		n.Control = migrateControl(o.Control)
		n.InlineSize = migrateSize(o.InlineSize, 150)
		n.OnClickDelay = migrateOnClickDelay(o.OnClickDelay)
		return n, true
	case v12.SelectNodeConfig:
		n := as[SelectNodeConfig](migrated)
		n.CommandChannel = valueOr(o.CommandChannel, 0)
		n.StateChannel = valueOr(o.StateChannel, 0)
		n.Control = migrateControl(o.Control)
		n.InlineSize = migrateSize(o.InlineSize, 100)
		n.OnClickDelay = migrateOnClickDelay(o.OnClickDelay)
		return n, true
	case v12.SetpointNodeConfig:
		n := as[SetpointNodeConfig](migrated)
		n.CommandChannel = valueOr(o.CommandChannel, 0)
		n.StateChannel = valueOr(o.StateChannel, 0)
		n.Dimensions = valueOr(o.Dimensions, spatial.Dimensions{})
		n.Control = migrateControl(o.Control)
		n.InlineSize = migrateSize(o.InlineSize, 100)
		n.OnClickDelay = migrateOnClickDelay(o.OnClickDelay)
		return n, true
	}
	return migrated, false
}

func migrateDisplayVariant(
	old v12.NodeConfigVariant,
	migrated NodeConfigVariant,
) NodeConfigVariant {
	switch o := old.(type) {
	case v12.StateIndicatorNodeConfig:
		n := as[StateIndicatorNodeConfig](migrated)
		n.Channel = valueOr(o.Channel, 0)
		n.InlineSize = migrateSize(o.InlineSize, 100)
		return n
	case v12.TankNodeConfig:
		// A tank declares its numeric and staleness fields inline, so the generated
		// copy skips the fields whose types changed.
		n := as[TankNodeConfig](migrated)
		n.Channel = valueOr(o.Channel, 0)
		n.Position = valueOr(o.Position, spatial.XY{})
		n.BorderRadius = &o.BorderRadius
		n.RollingAverage = migrateRollingAverage(o.RollingAverage)
		n.StalenessTimeout = migrateStalenessTimeout(o.StalenessTimeout)
		return n
	case v12.LightNodeConfig:
		n := as[LightNodeConfig](migrated)
		n.Channel = valueOr(o.Channel, 0)
		return n
	case v12.ScaleNodeConfig:
		n := as[ScaleNodeConfig](migrated)
		n.Position = valueOr(o.Position, spatial.XY{})
		return n
	case v12.GaugeNodeConfig:
		n := as[GaugeNodeConfig](migrated)
		n.Position = valueOr(o.Position, spatial.XY{})
		n.BarWidth = migrateSize(o.BarWidth, 10)
		return n
	case v12.StringDisplayNodeConfig:
		n := as[StringDisplayNodeConfig](migrated)
		n.Channel = valueOr(o.Channel, 0)
		n.InlineSize = migrateSize(o.InlineSize, 100)
		return n
	case v12.ValueNodeConfig:
		n := as[ValueNodeConfig](migrated)
		n.Position = valueOr(o.Position, spatial.XY{})
		n.InlineSize = migrateSize(o.InlineSize, 70)
		return n
	}
	return migrated
}

const (
	defaultStrokeWidth = 2
	maxStrokeWidth     = 20
)

func migrateShapeStrokeWidth(w float64) *uint8 {
	if math.IsNaN(w) {
		return nil
	}
	return new(migrateStrokeWidth(w))
}

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

// MigrateLabeledConfig lifts a v12 labeled config to v13.
func MigrateLabeledConfig(
	ctx context.Context,
	old v12.LabeledConfig,
) (LabeledConfig, error) {
	return autoMigrateLabeledConfig(ctx, old)
}

// MigrateLabelConfig lifts a v12 label config to v13 and keeps the label's v12 layout.
// It rounds the wrap width to whole pixels and clamps it to 1 through 65535. A NaN
// width becomes 150, a width of 0 becomes 1, and a negative width, which did not wrap,
// becomes 65535.
func MigrateLabelConfig(ctx context.Context, old v12.LabelConfig) (LabelConfig, error) {
	cfg, err := autoMigrateLabelConfig(ctx, old)
	if err != nil {
		return LabelConfig{}, err
	}
	cfg.MaxInlineSize = migrateMaxInlineSize(old.MaxInlineSize)
	return cfg, nil
}

const defaultMaxInlineSize = 150

func migrateMaxInlineSize(w float64) uint16 {
	if math.IsNaN(w) {
		return defaultMaxInlineSize
	}
	if w < 0 {
		return math.MaxUint16
	}
	return uint16(min(max(math.Round(w), 1), math.MaxUint16))
}

func migrateButtonMaxInlineSize(w float64) uint16 {
	if w == defaultMaxInlineSize {
		return math.MaxUint16
	}
	return migrateMaxInlineSize(w)
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

// MigrateStalenessConfig lifts a v12 staleness config to v13. It converts the timeout
// from seconds to a span. A timeout under 1 second becomes 1 second. A NaN timeout, or
// one of 0 or less, becomes 5 seconds.
func MigrateStalenessConfig(
	ctx context.Context,
	old v12.StalenessConfig,
) (StalenessConfig, error) {
	cfg, err := autoMigrateStalenessConfig(ctx, old)
	if err != nil {
		return StalenessConfig{}, err
	}
	cfg.StalenessTimeout = migrateStalenessTimeout(old.StalenessTimeout)
	return cfg, nil
}

// MigrateNumericTelemConfig lifts a v12 numeric telemetry config to v13. It clamps the
// rolling average to 1 through 65535. An absent channel becomes 0.
func MigrateNumericTelemConfig(
	ctx context.Context,
	old v12.NumericTelemConfig,
) (NumericTelemConfig, error) {
	cfg, err := autoMigrateNumericTelemConfig(ctx, old)
	if err != nil {
		return NumericTelemConfig{}, err
	}
	cfg.Channel = valueOr(old.Channel, 0)
	cfg.RollingAverage = migrateRollingAverage(old.RollingAverage)
	return cfg, nil
}

func migrateRollingAverage(n *int32) uint16 {
	if n == nil {
		return 1
	}
	return uint16(min(max(*n, 1), math.MaxUint16))
}

const defaultStalenessTimeout = 5 * telem.Second

func migrateStalenessTimeout(seconds float64) telem.TimeSpan {
	if math.IsNaN(seconds) || seconds <= 0 {
		return defaultStalenessTimeout
	}
	return migrateSpan(max(seconds, 1), telem.Second)
}

// maxOnClickDelay is the longest delay, in milliseconds, that a browser timer waits.
const maxOnClickDelay = math.MaxInt32

func migrateOnClickDelay(ms float64) telem.TimeSpan {
	if math.IsNaN(ms) || ms <= 0 {
		return 0
	}
	return migrateSpan(min(ms, maxOnClickDelay), telem.Millisecond)
}

// maxSpan is the largest span a JavaScript number holds exactly, so a client can send
// a migrated span back to the Core unchanged.
const maxSpan telem.TimeSpan = 1<<53 - 1

func migrateSpan(v float64, unit telem.TimeSpan) telem.TimeSpan {
	return telem.TimeSpan(min(math.Round(v*float64(unit)), float64(maxSpan)))
}

func migrateSize(v float64, def uint16) uint16 { return max(migratePixels(v, def), 1) }

func migrateOptionalPixels(v float64) *uint16 {
	if math.IsNaN(v) {
		return nil
	}
	return new(migratePixels(v, 0))
}

func migratePixels(v float64, def uint16) uint16 {
	if math.IsNaN(v) {
		return def
	}
	return uint16(min(max(math.Round(v), 0), math.MaxUint16))
}

func migrateNumSides(n float64) uint8 {
	if math.IsNaN(n) {
		return 6
	}
	return uint8(min(max(math.Round(n), 3), math.MaxUint8))
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

// MigrateToggleConfig lifts a v12 toggle config to v13. It converts the click delay
// from milliseconds to a span. A NaN or negative delay becomes 0. A delay longer than
// 2^31 - 1 ms, the longest that a browser timer waits, takes that length. An absent
// channel becomes 0, and an absent control state takes its schema defaults.
func MigrateToggleConfig(
	ctx context.Context,
	old v12.ToggleConfig,
) (ToggleConfig, error) {
	cfg, err := autoMigrateToggleConfig(ctx, old)
	if err != nil {
		return ToggleConfig{}, err
	}
	cfg.OnClickDelay = migrateOnClickDelay(old.OnClickDelay)
	cfg.StateChannel = valueOr(old.StateChannel, 0)
	cfg.CommandChannel = valueOr(old.CommandChannel, 0)
	cfg.Control = migrateControl(old.Control)
	return cfg, nil
}

func valueOr[T any](v *T, def T) T {
	if v == nil {
		return def
	}
	return *v
}

func migrateControl(old *v12.ControlStateConfig) ControlStateConfig {
	if old != nil {
		return *old
	}
	var cfg ControlStateConfig
	cfg.SetDefaults()
	return cfg
}

// resetScales sets each scale under v that is 0 or less, NaN, or infinite to 1. v must
// be settable.
func resetScales(v reflect.Value) {
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		c := reflect.New(v.Elem().Type()).Elem()
		c.Set(v.Elem())
		resetScales(c)
		v.Set(c)
	case reflect.Struct:
		if s, ok := reflect.TypeAssert[ScaledConfig](v); ok {
			if nonFinite(s.Scale) || s.Scale <= 0 {
				v.Set(reflect.ValueOf(ScaledConfig{Scale: 1}))
			}
			return
		}
		for _, f := range v.Fields() {
			if f.CanSet() {
				resetScales(f)
			}
		}
	default:
	}
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

// as returns a migrated variant as a T. It panics when v is not a T, since a generated
// migration keeps each variant's kind.
func as[T any](v any) T {
	t, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("migrated variant is a %T, not a %T", v, t))
	}
	return t
}
