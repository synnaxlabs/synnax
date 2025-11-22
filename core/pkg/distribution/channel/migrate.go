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

	"github.com/samber/lo"
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

// migrateChannelNames transforms existing channel names to meet validation
// requirements.
func (s *Service) migrateChannelNames(ctx context.Context, tx gorp.Tx) error {
	// Retrieve all channels
	var channels []Channel
	if err := gorp.NewRetrieve[Key, Channel]().
		Entries(&channels).
		Exec(ctx, tx); err != nil {
		return err
	}

	existingNames := make(set.Set[string], len(channels))
	channelsToUpdate := map[Key]string{}

	for _, ch := range channels {
		if err := ValidateName(ch.Name); err == nil && !existingNames.Contains(ch.Name) {
			existingNames.Add(ch.Name)
			continue
		}
		transformedName := TransformName(ch.Name)
		newName := NewUniqueName(transformedName, existingNames)
		channelsToUpdate[ch.Key()] = newName
		existingNames.Add(newName)
	}

	// Update channels with transformed names
	return gorp.NewUpdate[Key, Channel]().
		WhereKeys(lo.Keys(channelsToUpdate)...).
		Change(func(_ gorp.Context, c Channel) Channel {
			c.Name = channelsToUpdate[c.Key()]
			return c
		}).
		Exec(ctx, tx)
}
