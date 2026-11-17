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
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// MigrateLinePlot lifts a v8 line plot into v9, which drops the range owner.
func MigrateLinePlot(ctx context.Context, old v8.LinePlot) (LinePlot, error) {
	return autoMigrateLinePlot(ctx, old)
}

// RangeMigration deletes each stored v8 line plot that a range owns, and logs a warning
// for each.
var RangeMigration = projectv3.NewDeleteMigration(
	projectv3.DeleteConfig[v8.Key, v8.LinePlot]{
		Name:   "v62_lineplot_delete_range_owned",
		Type:   resource.TypeLineplot,
		NameOf: func(e v8.LinePlot) string { return e.Name },
		Reason: func(e v8.LinePlot) string {
			if e.Range != nil {
				return "owned by a range"
			}
			return ""
		},
	},
)

// Migration lifts each stored line plot from v8 to v9.
var Migration = migrate.NewEntry("v62_lineplot_drop_range", MigrateLinePlot)
