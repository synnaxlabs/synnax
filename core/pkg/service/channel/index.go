// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
)

// indexReference returns the config of the reference from a data channel to the index
// channel that holds its timestamps. A delete of an index fails while it keeps a data
// channel that uses the index.
func indexReference() gorp.ReferenceConfig[Key, Channel] {
	return gorp.ReferenceConfig[Key, Channel]{
		Name:     "local_index",
		Tables:   []string{tableName},
		OnDelete: gorp.Restrict,
		Targets: func(c *Channel) []gorp.Target {
			if c.IsIndex || c.LocalIndex == 0 {
				return nil
			}
			return []gorp.Target{gorp.To(tableName, c.Index())}
		},
		Detach: func(c *Channel, _ gorp.Target) { c.LocalIndex = 0 },
	}
}

// newDanglingIndexMigration returns the migration that clears the index of each
// channel whose index channel does not exist. A calculated channel whose index was
// deleted gets a new index on its next write. It decodes channels with their current
// type, so it runs after the channel version chain.
func newDanglingIndexMigration(db *gorp.DB) gorp.Migration {
	refs := []gorp.ReferenceConfig[Key, Channel]{indexReference()}
	return migrate.New(
		"dangling_index",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var channels []Channel
			if err := gorp.NewRetrieve[Key, Channel]().
				Entries(&channels).
				Exec(ctx, tx); err != nil {
				return err
			}
			changed := make([]Channel, 0, len(channels))
			for i := range channels {
				dropped, err := gorp.DropDangling(ctx, tx, db, refs, &channels[i])
				if err != nil {
					return err
				}
				if dropped {
					changed = append(changed, channels[i])
				}
			}
			return gorp.NewCreate[Key, Channel]().Entries(&changed).Exec(ctx, tx)
		},
	)
}
