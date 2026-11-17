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
	v9 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v9"
)

// MigrateLinePlot lifts a v9 line plot to v10, which stores every color alpha as a
// 0-255 byte.
func MigrateLinePlot(ctx context.Context, old v9.LinePlot) (LinePlot, error) {
	return autoMigrateLinePlot(ctx, old)
}

// Migration lifts stored line plots from v9 to v10.
var Migration = migrate.NewEntry("v62_color_alpha_byte", MigrateLinePlot)

// MigrateLine lifts a v9 line to v10.
func MigrateLine(ctx context.Context, old v9.Line) (Line, error) {
	return autoMigrateLine(ctx, old)
}

// MigrateRule lifts a v9 rule to v10.
func MigrateRule(ctx context.Context, old v9.Rule) (Rule, error) {
	return autoMigrateRule(ctx, old)
}
