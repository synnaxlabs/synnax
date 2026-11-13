// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// MigrateLog lifts a v3 log into v4, which drops the range owner.
func MigrateLog(ctx context.Context, old v3.Log) (Log, error) {
	return autoMigrateLog(ctx, old)
}

// RangeMigration deletes each stored v3 log that a range owns, and logs a warning
// for each.
var RangeMigration = projectv3.NewDeleteMigration(
	projectv3.DeleteConfig[v3.Key, v3.Log]{
		Name:   "v62_log_delete_range_owned",
		Type:   resource.TypeLog,
		NameOf: func(e v3.Log) string { return e.Name },
		Reason: func(e v3.Log) string {
			if e.Range != nil {
				return "owned by a range"
			}
			return ""
		},
	},
)

// Migration lifts each stored log from v3 to v4.
var Migration = migrate.NewEntry("v62_log_drop_range", MigrateLog)
