// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/lineplot/migrations/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/migrations/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/lineplot/migrations/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/lineplot/migrations/legacy/v2"
	v55 "github.com/synnaxlabs/synnax/pkg/service/lineplot/migrations/v55"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/text"
)

// MigrateLinePlot transforms the previous line plot snapshot (v55) into the
// current strongly-typed LinePlot. AutoMigrateLinePlot handles the
// trivially-copyable gorp-entry fields (Key, Name); the body fields are
// sourced from the per-plot blob the console used to persist alongside those
// gorp fields, after legacy.MigrateData walks the legacy migration chain up
// to v4.Data. UI-only fields (viewport, selection, mode, control, toolbar,
// measure, annotations, the wire-format key, remoteCreated) are dropped; they
// live on the console slice and never reach the server. v55 is the last
// snapshot in which LinePlot.Data is untyped; future migrations transform one
// typed snapshot into another and never need this blob handling.
func MigrateLinePlot(ctx context.Context, old v55.LinePlot) (LinePlot, error) {
	out, err := AutoMigrateLinePlot(ctx, old)
	if err != nil {
		return LinePlot{}, err
	}
	d, err := legacy.MigrateData(old.Data)
	if err != nil {
		return LinePlot{}, err
	}
	out.Title = migrateTitle(d.Title)
	out.Legend = migrateLegend(d.Legend)
	out.Channels = migrateChannels(d.Channels)
	out.Ranges = migrateRanges(d.Ranges)
	out.Axes = migrateAxes(d.Axes.Axes)
	out.Lines = migrateLines(d.Lines)
	out.Rules = migrateRules(d.Rules)
	return out, nil
}

func migrateTitle(t v0.Title) Title {
	return Title{Level: text.Level(t.Level), Visible: t.Visible}
}

func migrateLegend(l v1.Legend) Legend {
	return Legend{Hidden: !l.Visible, Position: migrateStickyXY(l.Position)}
}

func migrateStickyXY(p v1.LegendPosition) spatial.StickyXY {
	return spatial.StickyXY{
		X:     p.X,
		Y:     p.Y,
		Root:  migrateStickyRoot(p.Root),
		Units: migrateStickyUnits(p.Units),
	}
}

func migrateStickyRoot(r *v1.StickyRoot) *spatial.CornerLocation {
	if r == nil {
		return nil
	}
	return &spatial.CornerLocation{X: spatial.XLocation(r.X), Y: spatial.YLocation(r.Y)}
}

func migrateStickyUnits(u *v1.StickyUnits) *spatial.StickyUnits {
	if u == nil {
		return nil
	}
	return &spatial.StickyUnits{X: spatial.StickyUnit(u.X), Y: spatial.StickyUnit(u.Y)}
}

func migrateChannels(c v0.Channels) Channels {
	return Channels{X1: c.X1, X2: c.X2, Y1: c.Y1, Y2: c.Y2, Y3: c.Y3, Y4: c.Y4}
}

func migrateRanges(r v0.Ranges) Ranges {
	return Ranges{X1: r.X1, X2: r.X2}
}

func migrateAxes(a v2.Axes) Axes {
	return Axes{
		X1: migrateAxis(a.X1),
		X2: migrateAxis(a.X2),
		Y1: migrateAxis(a.Y1),
		Y2: migrateAxis(a.Y2),
		Y3: migrateAxis(a.Y3),
		Y4: migrateAxis(a.Y4),
	}
}

func migrateAxis(a v2.Axis) Axis {
	return Axis{
		Key:            AxisKey(a.Key),
		Label:          a.Label,
		LabelDirection: spatial.Direction(a.LabelDirection),
		LabelLevel:     text.Level(a.LabelLevel),
		Bounds:         spatial.Bounds{Lower: a.Bounds.Lower, Upper: a.Bounds.Upper},
		ManualBounds:   ManualBounds{Lower: !a.AutoBounds.Lower, Upper: !a.AutoBounds.Upper},
		TickSpacing:    a.TickSpacing,
		Type:           migrateTickType(a.Type),
	}
}

func migrateTickType(t string) *TickType {
	if t == "" {
		return nil
	}
	tt := TickType(t)
	return &tt
}

// colorPtr lifts a legacy value-typed color into the optional pointer the
// current schema uses. A zero color is the legacy "unset" sentinel, which
// maps to nil so the Console assigns a default at render time.
func colorPtr(c color.Color) *color.Color {
	if c.IsZero() {
		return nil
	}
	return &c
}

func migrateLines(in []v0.Line) []Line {
	out := make([]Line, len(in))
	for i, l := range in {
		out[i] = Line{
			Key:            l.Key,
			Label:          l.Label,
			Color:          colorPtr(l.Color),
			StrokeWidth:    l.StrokeWidth,
			Downsample:     l.Downsample,
			DownsampleMode: DownsampleMode(l.DownsampleMode),
		}
	}
	return out
}

func migrateRules(in []v0.Rule) []Rule {
	out := make([]Rule, len(in))
	for i, r := range in {
		out[i] = Rule{
			Key:       r.Key,
			Label:     r.Label,
			Color:     colorPtr(r.Color),
			Axis:      AxisKey(r.Axis),
			LineWidth: r.LineWidth,
			LineDash:  r.LineDash,
			Units:     r.Units,
			Position:  r.Position,
		}
	}
	return out
}
