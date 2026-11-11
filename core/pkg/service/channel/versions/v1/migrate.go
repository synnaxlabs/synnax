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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var Migration = migrate.NewEntry("v61_add_group", autoMigrateChannel)

// MembersMigration sets the group of each stored channel from the ontology edge that
// placed it in a group.
var MembersMigration = groupv1.NewMembersMigration(groupv1.MembersConfig[Key, Channel]{
	Name:         "v61_channel_group_membership",
	Type:         resource.TypeChannel,
	SetGroup:     func(e *Channel, g groupv1.Key) { e.Group = &g },
	ProjectOwned: false,
})

// IndexGroupMigration moves the index of each grouped calculated channel into the
// group of that channel when the index has no group and no other channel uses it.
var IndexGroupMigration = migrate.New("v61_calculated_index_group", migrateIndexGroups)

func migrateIndexGroups(
	ctx context.Context,
	tx gorp.Tx,
	_ alamos.Instrumentation,
) error {
	var channels []Channel
	if err := gorp.NewRetrieve[Key, Channel]().
		Entries(&channels).
		Exec(ctx, tx); err != nil {
		return err
	}
	var (
		groups = make(map[Key]*groupv1.Key)
		users  = make(map[Key]int)
	)
	for _, c := range channels {
		if c.IsIndex || c.LocalIndex == 0 {
			continue
		}
		users[c.Index()]++
		if c.IsCalculated() && c.Group != nil {
			groups[c.Index()] = c.Group
		}
	}
	var moved []Channel
	for _, c := range channels {
		if g, ok := groups[c.Key()]; ok && c.Group == nil && users[c.Key()] == 1 {
			c.Group = g
			moved = append(moved, c)
		}
	}
	return gorp.WrapWriter[Key, Channel](tx).Set(ctx, moved...)
}
