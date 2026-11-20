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

	"github.com/synnaxlabs/gorp/migrate"
	v10 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v10"
)

func MigrateLinePlot(ctx context.Context, old v10.LinePlot) (LinePlot, error) {
	return autoMigrateLinePlot(ctx, old)
}

func MigrateAxes(ctx context.Context, old v10.Axes) (Axes, error) {
	return autoMigrateAxes(ctx, old)
}

func MigrateAxis(ctx context.Context, old v10.Axis) (Axis, error) {
	return autoMigrateAxis(ctx, old)
}

var Migration = migrate.NewEntry("v63_lineplot_range_compare", MigrateLinePlot)
