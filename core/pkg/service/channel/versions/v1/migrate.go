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
// group of that channel when the index has no group.
var IndexGroupMigration = migrate.New("v61_calculated_index_group", migrateIndexGroups)

func migrateIndexGroups(
	ctx context.Context,
	tx gorp.Tx,
	_ alamos.Instrumentation,
) error {
	var calculated []Channel
	if err := gorp.NewRetrieve[Key, Channel]().
		Where(gorp.Match(func(_ gorp.Context, c *Channel) (bool, error) {
			return c.IsCalculated() && c.LocalIndex != 0 && c.Group != nil, nil
		})).
		Entries(&calculated).
		Exec(ctx, tx); err != nil || len(calculated) == 0 {
		return err
	}
	owners := make(map[Key]Channel, len(calculated))
	for _, c := range calculated {
		owners[c.Index()] = c
	}
	var indexes []Channel
	if err := gorp.NewRetrieve[Key, Channel]().
		Where(gorp.Match(func(_ gorp.Context, c *Channel) (bool, error) {
			_, ok := owners[c.Key()]
			return ok && c.Group == nil, nil
		})).
		Entries(&indexes).
		Exec(ctx, tx); err != nil {
		return err
	}
	for i := range indexes {
		indexes[i].Group = owners[indexes[i].Key()].Group
	}
	return gorp.WrapWriter[Key, Channel](tx).Set(ctx, indexes...)
}
