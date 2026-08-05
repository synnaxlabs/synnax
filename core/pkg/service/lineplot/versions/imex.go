// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v5 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v5"
	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/text"
)

// consoleAxis mirrors Axis as Console-written files serialize it: camelCase
// keys. Frozen; Console files no longer evolve.
type consoleAxis struct {
	// Key identifies the axis (e.g. "x1", "y1").
	Key AxisKey `json:"key" msgpack:"key"`
	// Label is the axis label text.
	Label string `json:"label" msgpack:"label"`
	// LabelDirection is the label orientation.
	LabelDirection spatial.Direction `json:"labelDirection" msgpack:"labelDirection"`
	// LabelLevel is the typography level of the label.
	LabelLevel text.Level `json:"labelLevel" msgpack:"labelLevel"`
	// Bounds are the manual axis bounds.
	Bounds spatial.Bounds `json:"bounds" msgpack:"bounds"`
	// ManualBounds reports which bounds are manually set.
	ManualBounds ManualBounds `json:"manualBounds" msgpack:"manualBounds"`
	// TickSpacing is the spacing between axis ticks in pixels.
	TickSpacing float64 `json:"tickSpacing" msgpack:"tickSpacing"`
	// Type is the optional tick type; empty means linear.
	Type *TickType `json:"type,omitempty" msgpack:"type,omitempty"`
}

func (a consoleAxis) axis() Axis {
	return Axis{
		Key: a.Key, Label: a.Label, LabelDirection: a.LabelDirection,
		LabelLevel: a.LabelLevel, Bounds: a.Bounds, ManualBounds: a.ManualBounds,
		TickSpacing: a.TickSpacing, Type: a.Type,
	}
}

// consoleAxes mirrors Axes as Console-written files serialize it.
type consoleAxes struct {
	// X1 is the primary x-axis configuration.
	X1 consoleAxis `json:"x1" msgpack:"x1"`
	// X2 is the secondary x-axis configuration.
	X2 consoleAxis `json:"x2" msgpack:"x2"`
	// Y1 is the first y-axis configuration.
	Y1 consoleAxis `json:"y1" msgpack:"y1"`
	// Y2 is the second y-axis configuration.
	Y2 consoleAxis `json:"y2" msgpack:"y2"`
	// Y3 is the third y-axis configuration.
	Y3 consoleAxis `json:"y3" msgpack:"y3"`
	// Y4 is the fourth y-axis configuration.
	Y4 consoleAxis `json:"y4" msgpack:"y4"`
}

func (a consoleAxes) axes() Axes {
	return Axes{
		X1: a.X1.axis(), X2: a.X2.axis(), Y1: a.Y1.axis(),
		Y2: a.Y2.axis(), Y3: a.Y3.axis(), Y4: a.Y4.axis(),
	}
}

// consoleLine mirrors Line as Console-written files serialize it.
type consoleLine struct {
	// Key is the line's unique key.
	Key string `json:"key" msgpack:"key"`
	// Label overrides the derived line label; absent derives it.
	Label *string `json:"label,omitempty" msgpack:"label,omitempty"`
	// Color is the line color.
	Color *color.Color `json:"color,omitempty" msgpack:"color,omitempty"`
	// StrokeWidth is the line width in pixels.
	StrokeWidth float64 `json:"strokeWidth" msgpack:"strokeWidth"`
	// Downsample is the downsampling factor.
	Downsample uint32 `json:"downsample" msgpack:"downsample"`
	// DownsampleMode selects the downsampling strategy.
	DownsampleMode DownsampleMode `json:"downsampleMode" msgpack:"downsampleMode"`
}

func (l consoleLine) line() Line {
	return Line{
		Key: l.Key, Label: l.Label, Color: l.Color, StrokeWidth: l.StrokeWidth,
		Downsample: l.Downsample, DownsampleMode: l.DownsampleMode,
	}
}

// consoleRule mirrors Rule as Console-written files serialize it.
type consoleRule struct {
	// Key is the rule's unique key.
	Key string `json:"key" msgpack:"key"`
	// Label is the rule label text.
	Label string `json:"label" msgpack:"label"`
	// Color is the rule line color.
	Color *color.Color `json:"color,omitempty" msgpack:"color,omitempty"`
	// Axis is the key of the axis the rule is bound to.
	Axis AxisKey `json:"axis" msgpack:"axis"`
	// LineWidth is the rule line width in pixels.
	LineWidth float64 `json:"lineWidth" msgpack:"lineWidth"`
	// LineDash is the rule dash spacing.
	LineDash float64 `json:"lineDash" msgpack:"lineDash"`
	// Units is the unit label displayed with the position.
	Units string `json:"units" msgpack:"units"`
	// Position is the rule position in axis units.
	Position float64 `json:"position" msgpack:"position"`
}

func (r consoleRule) rule() Rule {
	return Rule{
		Key: r.Key, Label: r.Label, Color: r.Color, Axis: r.Axis,
		LineWidth: r.LineWidth, LineDash: r.LineDash, Units: r.Units,
		Position: r.Position,
	}
}

// consoleDocument mirrors the typed LinePlot body fields as Console-written
// files serialize them at the typed export's top level.
type consoleDocument struct {
	// Title is the plot title configuration.
	Title Title `json:"title" msgpack:"title"`
	// Legend is the plot legend configuration.
	Legend Legend `json:"legend" msgpack:"legend"`
	// Channels binds channel keys to each axis.
	Channels Channels `json:"channels" msgpack:"channels"`
	// Ranges binds range keys to each x-axis.
	Ranges Ranges `json:"ranges" msgpack:"ranges"`
	// Axes is the per-axis configuration container.
	Axes consoleAxes `json:"axes" msgpack:"axes"`
	// Lines are the per-line styling configurations.
	Lines []consoleLine `json:"lines" msgpack:"lines"`
	// Rules are the annotation-line configurations.
	Rules []consoleRule `json:"rules" msgpack:"rules"`
}

// linePlot lifts the Console document into the current LinePlot shape.
func (d consoleDocument) linePlot() LinePlot {
	return LinePlot{
		Title: d.Title, Legend: d.Legend, Channels: d.Channels, Ranges: d.Ranges,
		Axes:  d.Axes.axes(),
		Lines: lo.Map(d.Lines, func(l consoleLine, _ int) Line { return l.line() }),
		Rules: lo.Map(d.Rules, func(r consoleRule, _ int) Rule { return r.rule() }),
	}
}

// DecodeImport materializes the envelope's body as a current-version LinePlot.
// Envelopes stamped at or above Floor decode through the generated migration
// chain; older ones are Console-era files — camelCase typed exports or console
// states — and are lifted forward. An envelope newer than Latest is rejected
// with a path-scoped validation error.
func DecodeImport(ctx context.Context, env imex.Envelope) (LinePlot, error) {
	if env.Version >= Floor {
		return decodeMigrate(ctx, env)
	}
	// Console-era typed exports (versionless) carry the current shape with
	// camelCase keys; console states never carry a name.
	if env.BodyNamed() {
		doc, err := imex.Decode[consoleDocument](ctx, env)
		if err != nil {
			return LinePlot{}, err
		}
		return doc.linePlot(), nil
	}
	// Console states embed the body inline: ride the storage lift, which
	// dispatches on the version stamped inside the body.
	body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
	if err != nil {
		return LinePlot{}, err
	}
	return v6.MigrateLinePlot(ctx, v5.LinePlot{Name: env.Name, Data: body})
}
