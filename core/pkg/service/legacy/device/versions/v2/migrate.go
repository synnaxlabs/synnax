// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"github.com/synnaxlabs/gorp/migrate"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var Migration = migrate.NewEntry("v61_add_group", autoMigrateDevice)

// MembersMigration sets the group of each stored device from the ontology edge that
// placed it in a group.
var MembersMigration = groupv1.NewMembersMigration(groupv1.MembersConfig[Key, Device]{
	Name:         "v61_device_group_membership",
	Type:         resource.TypeDevice,
	SetGroup:     func(e *Device, g groupv1.Key) { e.Group = &g },
	ProjectOwned: false,
})
