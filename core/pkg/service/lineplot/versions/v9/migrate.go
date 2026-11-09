// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v9

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
)

// MigrateLinePlot lifts a v8 line plot to v9, which stores every color alpha as a 0-255
// byte.
func MigrateLinePlot(ctx context.Context, old v8.LinePlot) (LinePlot, error) {
	return autoMigrateLinePlot(ctx, old)
}

// Migration lifts stored line plots from v8 to v9.
var Migration = migrate.NewEntry("v61_color_alpha_byte", MigrateLinePlot)

// MigrateLine lifts a v8 line to v9.
func MigrateLine(ctx context.Context, old v8.Line) (Line, error) {
	return autoMigrateLine(ctx, old)
}

// MigrateRule lifts a v8 rule to v9.
func MigrateRule(ctx context.Context, old v8.Rule) (Rule, error) {
	return autoMigrateRule(ctx, old)
}
