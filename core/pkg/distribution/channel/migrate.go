// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"

	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/set"
)

// migrate performs data migrations for the channel service.
// Migrations run sequentially and are tracked with an incrementing version number.
// New migrations should be appended to the end of the list.
func (s *Service) migrate(ctx context.Context) error {
	return migrate.GorpRunner{
		Key: "channel_service",
		Migrations: []migrate.GorpSpec{
			{Name: "migrate_channel_names", Migrate: s.migrateChannelNames},
		},
		Force: *s.cfg.ForceMigration,
	}.Run(ctx, s.cfg.ClusterDB)
}

// migrateChannelNames transforms existing channel names to remove invalid characters
// like spaces and ensure that the name is unique.
func (s *Service) migrateChannelNames(ctx context.Context, tx gorp.Tx) error {
	existingNames := set.Set[string]{}
	return gorp.NewUpdate[Key, Channel]().
		Change(func(_ gorp.Context, c Channel) Channel {
			transformedName := TransformName(c.Name)
			c.Name = NewUniqueName(transformedName, existingNames)
			existingNames.Add(c.Name)
			return c
		}).
		Exec(ctx, tx)
}
