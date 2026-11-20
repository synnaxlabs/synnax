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

	"github.com/synnaxlabs/gorp/migrate"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
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

// OwnersMigration sets the project or range of each stored schematic from the ontology
// edge that owned it, and deletes each schematic with no owner.
var OwnersMigration = projectv3.NewOwnersMigration(
	projectv3.OwnersConfig[Key, Schematic]{
		Name:     "v61_schematic_owner",
		Type:     resource.TypeSchematic,
		Owned:    func(e Schematic) bool { return e.Project != nil || e.Range != nil },
		NameOf:   func(e Schematic) string { return e.Name },
		SetOwner: func(e *Schematic, p, r *uuid.UUID) { e.Project, e.Range = p, r },
	},
)
