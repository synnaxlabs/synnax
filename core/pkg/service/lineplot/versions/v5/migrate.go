// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v0"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/text"
)

// MigrateLinePlot transforms the previous line plot snapshot (v0) into the v5
// strongly-typed LinePlot. autoMigrateLinePlot handles the trivially-copyable
// gorp-entry fields (Key, Name); the body fields are sourced from the per-plot blob the
// Console used to persist alongside those gorp fields, after legacy.MigrateData walks
// the legacy migration chain up to v4.Data. UI-only fields (viewport, selection, mode,
// control, toolbar, measure, annotations, the wire-format key, remoteCreated) are
// dropped; they live on the Console slice and never reach the server. v0 is the last
// snapshot in which LinePlot.Data is untyped; future migrations transform one typed
// snapshot into another and never need this blob handling.
func MigrateLinePlot(old v0.LinePlot) (LinePlot, error) {
	out, err := autoMigrateLinePlot(old)
	if err != nil {
		return LinePlot{}, err
	}
	d, err := legacy.MigrateData(old.Data)
	if err != nil {
		return LinePlot{}, err
	}
	out.Title = Title{Level: text.Level(d.Title.Level), Visible: d.Title.Visible}
	out.Legend = Legend{
		Hidden:   !d.Legend.Visible,
		Position: migrateStickyXY(d.Legend.Position),
	}
	out.Channels = migrateChannels(d.Channels)
	out.Ranges = migrateRanges(d.Ranges)
	out.Axes = migrateAxes(d.Axes.Axes)
	out.Lines = migrateLines(d.Lines)
	out.Rules = migrateRules(d.Rules)
	return out, nil
}

func migrateStickyXY(p legacy.LegendPosition) spatial.StickyXY {
	return spatial.StickyXY{
		X:     p.X,
		Y:     p.Y,
		Root:  migrateStickyRoot(p.Root),
		Units: migrateStickyUnits(p.Units),
	}
}

func migrateStickyRoot(r *legacy.StickyRoot) spatial.CornerLocation {
	if r == nil {
		return spatial.CornerLocation{X: spatial.XLocationLeft, Y: spatial.YLocationTop}
	}
	return spatial.CornerLocation{X: spatial.XLocation(r.X), Y: spatial.YLocation(r.Y)}
}

func migrateStickyUnits(u *legacy.StickyUnits) spatial.StickyUnits {
	if u == nil {
		return spatial.StickyUnits{X: spatial.StickyUnitPx, Y: spatial.StickyUnitPx}
	}
	return spatial.StickyUnits{X: spatial.StickyUnit(u.X), Y: spatial.StickyUnit(u.Y)}
}

func migrateChannels(c legacy.Channels) Channels {
	return Channels{X1: c.X1, X2: c.X2, Y1: c.Y1, Y2: c.Y2, Y3: c.Y3, Y4: c.Y4}
}

func migrateRanges(r legacy.Ranges) Ranges {
	return Ranges{X1: r.X1, X2: r.X2}
}

func migrateAxes(a legacy.Axes) Axes {
	return Axes{
		X1: migrateAxis(a.X1),
		X2: migrateAxis(a.X2),
		Y1: migrateAxis(a.Y1),
		Y2: migrateAxis(a.Y2),
		Y3: migrateAxis(a.Y3),
		Y4: migrateAxis(a.Y4),
	}
}

func migrateAxis(a legacy.Axis) Axis {
	return Axis{
		Key:            AxisKey(a.Key),
		Label:          a.Label,
		LabelDirection: spatial.Direction(a.LabelDirection),
		LabelLevel:     text.Level(a.LabelLevel),
		Bounds:         spatial.Bounds{Lower: a.Bounds.Lower, Upper: a.Bounds.Upper},
		ManualBounds: ManualBounds{
			Lower: !a.AutoBounds.Lower,
			Upper: !a.AutoBounds.Upper,
		},
		TickSpacing: a.TickSpacing,
		Type:        migrateTickType(a.Type),
	}
}

func migrateTickType(t string) *TickType {
	if t == "" {
		return nil
	}
	tt := TickType(t)
	return &tt
}

// colorPtr lifts a legacy value-typed color into the optional pointer the current
// schema uses. A zero color is the legacy "unset" sentinel, which maps to nil so the
// Console assigns a default at render time.
func colorPtr(c color.Color) *color.Color {
	if c.IsZero() {
		return nil
	}
	return &c
}

func migrateLines(in []legacy.Line) []Line {
	return lo.Map(in, func(l legacy.Line, _ int) Line {
		return Line{
			Key:            l.Key,
			Label:          l.Label,
			Color:          colorPtr(l.Color),
			StrokeWidth:    l.StrokeWidth,
			Downsample:     l.Downsample,
			DownsampleMode: DownsampleMode(l.DownsampleMode),
		}
	})
}

func migrateRules(in []legacy.Rule) []Rule {
	return lo.Map(in, func(r legacy.Rule, _ int) Rule {
		return Rule{
			Key:       r.Key,
			Label:     r.Label,
			Color:     colorPtr(r.Color),
			Axis:      AxisKey(r.Axis),
			LineWidth: r.LineWidth,
			LineDash:  r.LineDash,
			Units:     r.Units,
			Position:  r.Position,
		}
	})
}

// Migration lifts stored line plots from the v0 blob layout to the typed v5 shape.
var Migration = gorp.NewEntryMigration("v55_lift_typed_lineplot", MigrateLinePlot)
