// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v10

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v9 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v9"
	xcolorv0 "github.com/synnaxlabs/x/color/versions/v0"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
)

// palette holds the colors the Console painted on a v9 line or rule with no color. The
// element at index i of the lines, or of the rules, took palette[i%len(palette)].
var palette = [...]uint32{
	0xDC136CFF, 0x20A4F3FF, 0x7AC74FFF, 0xFFC43DFF, 0xFE5F55FF, 0x8075FFFF, 0xD90429FF,
	0xF7AEF8FF, 0xF18F01FF, 0x791E94FF, 0x279AF1FF, 0xD9FFF5FF, 0xFF6B6BFF, 0x52FFB8FF,
}

// MigrateLinePlot lifts a v9 line plot to v10. Each line and rule color becomes a fixed
// color with a 0-255 alpha. A line or rule with no color gets the palette color the
// Console painted on it, so the plot looks the same.
func MigrateLinePlot(ctx context.Context, old v9.LinePlot) (LinePlot, error) {
	p, err := autoMigrateLinePlot(ctx, old)
	if err != nil {
		return LinePlot{}, err
	}
	for i := range p.Lines {
		if p.Lines[i].Color == nil {
			p.Lines[i].Color = paletteColor(i)
		}
	}
	for i := range p.Rules {
		if p.Rules[i].Color == nil {
			p.Rules[i].Color = paletteColor(i)
		}
	}
	return p, nil
}

// Migration lifts stored line plots from v9 to v10.
var Migration = migrate.NewEntry("v62_color_alpha_byte", MigrateLinePlot)

// MigrateLine lifts a v9 line to v10. Its color becomes a fixed color, and no color
// stays absent.
func MigrateLine(ctx context.Context, old v9.Line) (Line, error) {
	l, err := autoMigrateLine(ctx, old)
	if err != nil {
		return Line{}, err
	}
	l.Color, err = fixedColor(ctx, old.Color)
	return l, err
}

// MigrateRule lifts a v9 rule to v10. Its color becomes a fixed color, and no color
// stays absent.
func MigrateRule(ctx context.Context, old v9.Rule) (Rule, error) {
	r, err := autoMigrateRule(ctx, old)
	if err != nil {
		return Rule{}, err
	}
	r.Color, err = fixedColor(ctx, old.Color)
	return r, err
}

func fixedColor(ctx context.Context, old *xcolorv0.Color) (*color.Ref, error) {
	if old == nil {
		return nil, nil
	}
	c, err := xcolor.MigrateColor(ctx, *old)
	if err != nil {
		return nil, err
	}
	return &color.Ref{Variant: color.FixedRef{Value: c}}, nil
}

func paletteColor(i int) *color.Ref {
	c := xcolor.FromUint32(palette[i%len(palette)])
	return &color.Ref{Variant: color.FixedRef{Value: c}}
}
