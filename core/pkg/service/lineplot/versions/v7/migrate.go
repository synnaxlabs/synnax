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

	"github.com/synnaxlabs/gorp/migrate"
	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
)

// MigrateLinePlot lifts a v6 line plot to v7, migrating each of its lines.
func MigrateLinePlot(ctx context.Context, old v6.LinePlot) (LinePlot, error) {
	return autoMigrateLinePlot(ctx, old)
}

// MigrateLine maps the average downsample mode to the average aggregation. A window of
// one sample or less drew raw samples, so it keeps min-max. The downsample factor has
// no mapping to a detail level, so every line takes the default detail.
func MigrateLine(ctx context.Context, old v6.Line) (Line, error) {
	l, err := autoMigrateLine(ctx, old)
	if err != nil {
		return Line{}, err
	}
	if old.DownsampleMode == v6.DownsampleModeAverage && old.Downsample > 1 {
		l.Aggregation = AggregationAverage
	}
	l.ApplyDefaults()
	return l, nil
}

// Migration lifts stored v6 line plots to v7.
var Migration = migrate.NewEntry("v59_line_reduction", MigrateLinePlot)
