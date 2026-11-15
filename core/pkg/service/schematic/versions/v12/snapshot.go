// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v12

import (
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
)

// SnapshotMigration deletes each stored v11 schematic that is a snapshot or that a
// range owns, and logs a warning for each.
var SnapshotMigration = projectv3.NewDeleteMigration(
	projectv3.DeleteConfig[v11.Key, v11.Schematic]{
		Name:   "v62_schematic_delete_snapshots",
		Type:   resource.TypeSchematic,
		NameOf: func(e v11.Schematic) string { return e.Name },
		Reason: func(e v11.Schematic) string {
			switch {
			case e.Snapshot:
				return "snapshot"
			case e.Range != nil:
				return "owned by a range"
			}
			return ""
		},
	},
)
