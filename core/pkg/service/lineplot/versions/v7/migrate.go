// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v7

import (
	"context"

	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

func MigrateLinePlot(ctx context.Context, old v6.LinePlot) (LinePlot, error) {
	return autoMigrateLinePlot(ctx, old)
}

// MigrateRanges carries the v6 custom range variants into the widened v7 union.
func MigrateRanges(_ context.Context, old v6.Ranges) (Ranges, error) {
	out := Ranges{X1: old.X1, X2: old.X2}
	if old.Custom == nil {
		return out, nil
	}
	switch v := old.Custom.Variant.(type) {
	case v6.DynamicCustomRange:
		out.Custom = &CustomRange{Variant: DynamicCustomRange(v)}
	case v6.StaticCustomRange:
		out.Custom = &CustomRange{Variant: StaticCustomRange(v)}
	case nil:
		out.Custom = &CustomRange{}
	default:
		return Ranges{}, errors.Newf("unknown v6 custom range variant %T", v)
	}
	return out, nil
}

func MigrateAxes(ctx context.Context, old v6.Axes) (Axes, error) {
	return autoMigrateAxes(ctx, old)
}

// MigrateAxis adds the samples mode and the default spectrum to every axis.
func MigrateAxis(ctx context.Context, old v6.Axis) (Axis, error) {
	axis, err := autoMigrateAxis(ctx, old)
	if err != nil {
		return Axis{}, err
	}
	axis.Mode = XAxisModeSamples
	axis.Spectrum = Spectrum{
		Window:     WindowFunctionHann,
		Scale:      MagnitudeScaleLinear,
		PointLimit: 65536,
	}
	return axis, nil
}

var Migration = gorp.NewEntryMigration("v59_line_plot_windows", MigrateLinePlot)
