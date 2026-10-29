// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v8

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v7 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v7"
)

// MigrateLinePlot lifts a v7 line plot to v8.
func MigrateLinePlot(ctx context.Context, old v7.LinePlot) (LinePlot, error) {
	return autoMigrateLinePlot(ctx, old)
}

// MigrateLine lifts a v7 line to v8, keeping its stroke width.
func MigrateLine(ctx context.Context, old v7.Line) (Line, error) {
	l, err := autoMigrateLine(ctx, old)
	if err != nil {
		return Line{}, err
	}
	l.StrokeWidth = new(old.StrokeWidth)
	return l, nil
}

// Migration lifts stored line plots from v7 to v8.
var Migration = migrate.NewEntry("v61_optional_line_stroke_width", MigrateLinePlot)
