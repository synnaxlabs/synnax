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
		Key: "sy_channel_migration_version",
		Migrations: []migrate.GorpSpec{
			{Name: "name_validation", Migrate: s.migrateChannelNames},
		},
	}.Run(ctx, s.db)
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
	channelsToUpdate := make([]Channel, 0)

	for _, ch := range channels {
		// Check if name is valid
		if err := ValidateName(ch.Name); err != nil || existingNames.Contains(ch.Name) {
			transformedName := TransformName(ch.Name)
			uniqueName := NewUniqueName(transformedName, existingNames)
			ch.Name = uniqueName
			channelsToUpdate = append(channelsToUpdate, ch)
		}
		// Track the name (either original or transformed)
		existingNames.Add(ch.Name)
	}

	// Update channels with transformed names
	for _, ch := range channelsToUpdate {
		if err := gorp.NewUpdate[Key, Channel]().
			WhereKeys(ch.Key()).
			Change(func(_ gorp.Context, c Channel) Channel {
				c.Name = ch.Name
				return c
			}).
			Exec(ctx, tx); err != nil {
			return err
		}
	}

	return nil
}
