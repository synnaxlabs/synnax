// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"

	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

// migrateArc lifts a v0 arc into the current shape, dropping the persisted program
// status.
func migrateArc(ctx context.Context, old v0.Arc) (Arc, error) {
	return autoMigrateArc(ctx, old)
}

// renameSetStatus rewrites every deprecated set_status flow node in a to status.set.
func renameSetStatus(_ context.Context, a Arc) (Arc, error) {
	for i, n := range a.Graph.Nodes {
		if n.Type != "set_status" {
			continue
		}
		n.Type = "status.set"
		n.Config = msgpack.EncodedJSON{
			"key_or_name": legacyStatusConfigString(n.Config, "statusKey", ""),
			"variant":     legacyStatusConfigString(n.Config, "variant", "success"),
			"message":     legacyStatusConfigString(n.Config, "message", ""),
		}
		a.Graph.Nodes[i] = n
	}
	return a, nil
}

// legacyStatusConfigString returns the string value stored at key in config, falling
// back to def when the key is absent or does not hold a string.
func legacyStatusConfigString(config msgpack.EncodedJSON, key, def string) string {
	if v, ok := config[key].(string); ok {
		return v
	}
	return def
}

// codecMigration re-encodes stored arcs from msgpack to orc. It is pinned to the v0
// shape so its output stays stable as Arc evolves.
var codecMigration = gorp.CodecMigration[Key, v0.Arc]("msgpack_to_orc")

// dropProgramStatusMigration lifts stored arcs from v0 to v1, dropping the persisted
// program status.
var dropProgramStatusMigration = gorp.NewEntryMigration(
	"v54_drop_program_status", migrateArc, codecMigration.Key(),
)

// renameSetStatusMigration renames legacy set_status graph nodes onto the current node
// type.
var renameSetStatusMigration = gorp.NewEntryMigration(
	"v55_rename_set_status", renameSetStatus, dropProgramStatusMigration.Key(),
)

// Migrations is the ordered set of migrations introduced at this version.
var Migrations = []migrate.Migration{
	codecMigration,
	dropProgramStatusMigration,
	renameSetStatusMigration,
}
