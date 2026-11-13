// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v4 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v4"
)

// MigrateTable lifts a v4 table into v5, which drops the range owner.
func MigrateTable(ctx context.Context, old v4.Table) (Table, error) {
	return autoMigrateTable(ctx, old)
}

// RangeMigration deletes each stored v4 table that a range owns, and logs a warning
// for each.
var RangeMigration = projectv3.NewDeleteMigration(
	projectv3.DeleteConfig[v4.Key, v4.Table]{
		Name:   "v62_table_delete_range_owned",
		Type:   resource.TypeTable,
		NameOf: func(e v4.Table) string { return e.Name },
		Reason: func(e v4.Table) string {
			if e.Range != nil {
				return "owned by a range"
			}
			return ""
		},
	},
)

// Migration lifts each stored table from v4 to v5.
var Migration = migrate.NewEntry("v62_table_drop_range", MigrateTable)
