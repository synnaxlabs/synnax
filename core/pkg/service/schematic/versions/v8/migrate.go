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

	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	"github.com/synnaxlabs/x/gorp"
)

// MigrateSchematic lifts a v7 schematic into the v8 shape, dropping each node's
// measured dimensions.
func MigrateSchematic(ctx context.Context, old v7.Schematic) (Schematic, error) {
	return autoMigrateSchematic(ctx, old)
}

// MigrateNode lifts a v7 node into the v8 shape, dropping its measured dimensions.
func MigrateNode(ctx context.Context, old v7.Node) (Node, error) {
	return autoMigrateNode(ctx, old)
}

// Migration lifts stored schematics from v7 to v8.
var Migration = gorp.NewEntryMigration("v57_drop_node_measured", MigrateSchematic)
