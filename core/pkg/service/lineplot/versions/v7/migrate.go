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
	"github.com/synnaxlabs/x/gorp"
)

func MigrateLinePlot(ctx context.Context, old v6.LinePlot) (LinePlot, error) {
	return autoMigrateLinePlot(ctx, old)
}

// MigrateLine maps the average downsample mode to the average aggregation. The
// downsample factor has no mapping to a detail level, so every line takes the default
// detail.
func MigrateLine(ctx context.Context, old v6.Line) (Line, error) {
	l, err := autoMigrateLine(ctx, old)
	if err != nil {
		return Line{}, err
	}
	l.Aggregation = AggregationMinMax
	if old.DownsampleMode == v6.DownsampleModeAverage {
		l.Aggregation = AggregationAverage
	}
	l.Detail = DetailMedium
	return l, nil
}

var Migration = gorp.NewEntryMigration("v59_line_reduction", MigrateLinePlot)
