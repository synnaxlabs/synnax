// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xuuid "github.com/synnaxlabs/x/uuid"
)

// migrationNamespace is the UUIDv5 namespace of the key that Migration gives the
// calculation of a channel.
var migrationNamespace = uuid.MustParse("3d7c1f2a-9e44-4c1b-8a55-2f6b0e9d4c71")

// MigratedKey returns the key that Migration gives the calculation of the channel with
// key.
func MigratedKey(key channel.Key) Key {
	return xuuid.NewV5(migrationNamespace, key.String())
}

// Migration returns the migration that builds the calculation record of each stored
// channel with an expression, keyed by MigratedKey, and deletes the status of each
// one. It reads the channel and status tables, so it runs after both open.
func (s *Service) Migration(statuses *status.Service) gorp.Migration {
	return migrate.New(
		"v63_create_calculations",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var channels []channel.Channel
			if err := s.channels.NewRetrieve().
				Where(channel.MatchCalculated()).
				Entries(&channels).
				Exec(ctx, tx); err != nil {
				return err
			}
			owners := make([]resource.ID, len(channels))
			for i, ch := range channels {
				owners[i] = channel.ResourceID(ch.Key())
				c := Calculation{
					Key:        MigratedKey(ch.Key()),
					Output:     ch.Key(),
					Expression: ch.Expression,
					Operations: FromChannelOperations(ch.Operations),
				}
				if err := s.table.NewCreate().Entry(&c).Exec(ctx, tx); err != nil {
					return err
				}
			}
			return statuses.NewWriter(tx).DeleteOwned(ctx, owners...)
		},
	)
}
