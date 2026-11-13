// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v11

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	"go.uber.org/zap"
)

// SnapshotMigration deletes each stored v10 schematic snapshot and logs a warning for
// each.
var SnapshotMigration = migrate.New(
	"v61_schematic_delete_snapshots",
	func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
		var schematics []v10.Schematic
		if err := gorp.NewRetrieve[v10.Key, v10.Schematic]().
			Entries(&schematics).
			Exec(ctx, tx); err != nil {
			return err
		}
		var snapshots []v10.Key
		for _, s := range schematics {
			if !s.Snapshot {
				continue
			}
			ins.L.Warn(
				"deleted a schematic snapshot",
				zap.String("key", s.Key.String()),
				zap.String("name", s.Name),
			)
			snapshots = append(snapshots, s.Key)
		}
		return gorp.WrapWriter[v10.Key, v10.Schematic](tx).Delete(ctx, snapshots...)
	},
)

// MigrateSchematic lifts a v10 schematic into v11. The schematic sits in no group.
func MigrateSchematic(ctx context.Context, old v10.Schematic) (Schematic, error) {
	return autoMigrateSchematic(ctx, old)
}

var Migration = migrate.NewEntry("v61_add_group", MigrateSchematic)

// MembersMigration sets the group of each stored schematic from the ontology edge that
// placed it in a group.
var MembersMigration = groupv1.NewMembersMigration(
	groupv1.MembersConfig[Key, Schematic]{
		Name:         "v61_schematic_group_membership",
		Type:         resource.TypeSchematic,
		SetGroup:     func(e *Schematic, g groupv1.Key) { e.Group = &g },
		ProjectOwned: true,
	},
)

// OwnersMigration sets the project of each stored schematic from the ontology edge that
// owned it, and deletes each schematic with no owner.
var OwnersMigration = projectv3.NewOwnersMigration(
	projectv3.OwnersConfig[Key, Schematic]{
		Name:     "v61_schematic_owner",
		Type:     resource.TypeSchematic,
		Owned:    func(e Schematic) bool { return e.Project != nil },
		NameOf:   func(e Schematic) string { return e.Name },
		SetOwner: func(e *Schematic, p uuid.UUID) { e.Project = &p },
	},
)
