// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp/migrate"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v2"
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// MigratePanel lifts a v2 panel into v3. The panel sits in no group.
func MigratePanel(ctx context.Context, old v2.Panel) (Panel, error) {
	return autoMigratePanel(ctx, old)
}

var Migration = migrate.NewEntry("v61_add_group", MigratePanel)

// MembersMigration sets the group of each stored panel from the ontology edge that
// placed it in a group.
var MembersMigration = groupv1.NewMembersMigration(groupv1.MembersConfig[Key, Panel]{
	Name:         "v61_panel_group_membership",
	Type:         resource.TypePanel,
	SetGroup:     func(e *Panel, g groupv1.Key) { e.Group = &g },
	ProjectOwned: true,
})

// OwnersMigration sets the project of each stored panel from the ontology edge that
// owned it, and deletes each panel with no owner.
var OwnersMigration = projectv3.NewOwnersMigration(
	projectv3.OwnersConfig[Key, Panel]{
		Name:     "v61_panel_owner",
		Type:     resource.TypePanel,
		Owned:    func(e Panel) bool { return e.Project != nil },
		NameOf:   func(e Panel) string { return e.Name },
		SetOwner: func(e *Panel, p uuid.UUID) { e.Project = &p },
	},
)
