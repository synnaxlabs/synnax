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
	"uuid"

	"github.com/synnaxlabs/gorp/migrate"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v3 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v3"
)

// MigrateTable lifts a v3 table into v4. The table sits in no group.
func MigrateTable(ctx context.Context, old v3.Table) (Table, error) {
	return autoMigrateTable(ctx, old)
}

var Migration = migrate.NewEntry("v61_add_group", MigrateTable)

// MembersMigration sets the group of each stored table from the ontology edge that
// placed it in a group.
var MembersMigration = groupv1.NewMembersMigration(groupv1.MembersConfig[Key, Table]{
	Name:         "v61_table_group_membership",
	Type:         resource.TypeTable,
	SetGroup:     func(e *Table, g groupv1.Key) { e.Group = &g },
	ProjectOwned: true,
})

// OwnersMigration sets the project of each stored table from the ontology edge that
// owned it, and deletes each table with no owner.
var OwnersMigration = projectv3.NewOwnersMigration(
	projectv3.OwnersConfig[Key, Table]{
		Name:     "v61_table_owner",
		Type:     resource.TypeTable,
		Owned:    func(e Table) bool { return e.Project != nil },
		NameOf:   func(e Table) string { return e.Name },
		SetOwner: func(e *Table, p uuid.UUID) { e.Project = &p },
	},
)
