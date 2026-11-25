// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v11

import (
	"context"
	"math"

	"github.com/synnaxlabs/gorp/migrate"
	v10 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v10"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
)

// MigrateLinePlot migrates old to v11 and drops each repeated range reference.
func MigrateLinePlot(ctx context.Context, old v10.LinePlot) (LinePlot, error) {
	lp, err := autoMigrateLinePlot(ctx, old)
	if err != nil {
		return LinePlot{}, err
	}
	lp.Ranges = UniqueRanges(lp.Ranges)
	return lp, nil
}

func MigrateAxes(ctx context.Context, old v10.Axes) (Axes, error) {
	return autoMigrateAxes(ctx, old)
}

// MigrateAxis migrates old to v11. Only a manual edge gives the bounds meaning, so an
// axis with no manual edge keeps no bounds. A single manual edge keeps its value, and
// the auto edge moves where needed to keep the pair valid. Two manual edges swap when
// reversed, and return to auto when equal.
func MigrateAxis(ctx context.Context, old v10.Axis) (Axis, error) {
	a, err := autoMigrateAxis(ctx, old)
	if err != nil {
		return Axis{}, err
	}
	lower, upper := old.Bounds.Lower, old.Bounds.Upper
	m := old.ManualBounds
	switch {
	case !m.Lower && !m.Upper:
		return a, nil
	case m.Lower && m.Upper && lower == upper:
		a.ManualBounds = ManualBounds{}
		return a, nil
	case m.Lower && m.Upper:
		lower, upper = min(lower, upper), max(lower, upper)
	case m.Lower && upper <= lower:
		upper = lower + step(lower)
	case m.Upper && lower >= upper:
		lower = upper - step(upper)
	}
	a.Bounds = &spatial.ClosedBounds{Lower: lower, Upper: upper}
	return a, nil
}

var Migration = migrate.NewEntry("v63_lineplot_range_compare", MigrateLinePlot)

// step returns an amount that changes v when added to it. A step of 1 is lost above
// 2^53, which nanosecond timestamps pass.
func step(v float64) float64 { return max(1, math.Abs(v)*0x1p-51) }
