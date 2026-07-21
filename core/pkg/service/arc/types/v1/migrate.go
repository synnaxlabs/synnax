// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Retargeted by oracle. Edit freely.
//
// autoMigrate handles field copying. Customize non-zero defaults below.

package v1

import (
	"context"

	graphv1 "github.com/synnaxlabs/arc/graph/types/v1"
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

func migrateArc(ctx context.Context, old v0.Arc) (Arc, error) {
	return autoMigrateArc(ctx, old)
}

// renameSetStatus rewrites every deprecated set_status flow node in a to status.set.
func renameSetStatus(_ context.Context, a Arc) (Arc, error) {
	for i := range a.Graph.Nodes {
		a.Graph.Nodes[i] = migrateLegacySetStatusNode(a.Graph.Nodes[i])
	}
	return a, nil
}

// migrateLegacySetStatusNode rewrites a deprecated set_status flow node to status.set,
// remapping the legacy statusKey config parameter to key_or_name and dropping any
// parameters the new function no longer accepts. Nodes of any other type are returned
// unchanged. This mirrors the Console-side migration in console/src/arc/types/v2.ts.
func migrateLegacySetStatusNode(n graphv1.Node) graphv1.Node {
	if n.Type != "set_status" {
		return n
	}
	n.Type = "status.set"
	n.Config = msgpack.EncodedJSON{
		"key_or_name": legacyStatusConfigString(n.Config, "statusKey", ""),
		"variant":     legacyStatusConfigString(n.Config, "variant", "success"),
		"message":     legacyStatusConfigString(n.Config, "message", ""),
	}
	return n
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
