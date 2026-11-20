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
	"fmt"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	xuuid "github.com/synnaxlabs/x/uuid"
)

// migrationNamespace is the UUIDv5 namespace of the key that Migration gives the
// calculation of a channel.
var migrationNamespace = uuid.MustParse("3d7c1f2a-9e44-4c1b-8a55-2f6b0e9d4c71")

// migratedKey returns the key that Migration gives the calculation of the channel with
// key.
func migratedKey(key channel.Key) Key {
	return xuuid.NewV5(migrationNamespace, key.String())
}

// Migration returns the migration that builds the calculation record of each stored
// channel with an expression, keyed by a UUIDv5 of the channel key, and deletes the
// status of each one. A reset channel that issuedAtOpen says the Core never issued
// becomes absent. An operation of no type is dropped, with a warning status that names
// the channel. Migration reads the channel and status tables, so it runs after both
// open.
func (s *Service) Migration(
	statuses *status.Service,
	issuedAtOpen func(channel.Key) bool,
) gorp.Migration {
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
			var warnings []Status
			for i, ch := range channels {
				owners[i] = channel.ResourceID(ch.Key())
				ops, dropped := storedOperations(ch.Operations, issuedAtOpen)
				c := Calculation{
					Key:        migratedKey(ch.Key()),
					Output:     ch.Key(),
					Expression: ch.Expression,
					Operations: fromChannelOperations(ops),
				}
				if err := s.table.NewCreate().Entry(&c).Exec(ctx, tx); err != nil {
					return err
				}
				if dropped {
					warnings = append(warnings, droppedStatus(ch.Name))
				}
			}
			w := statuses.NewWriter(tx)
			if err := w.DeleteOwned(ctx, owners...); err != nil {
				return err
			}
			return w.SetMany(ctx, &warnings)
		},
	)
}

// storedOperations returns the operations of ops that have a type, with each reset
// channel that the Core never issued cleared, and whether it dropped any.
func storedOperations(
	ops []channel.Operation,
	issuedAtOpen func(channel.Key) bool,
) ([]channel.Operation, bool) {
	kept := make([]channel.Operation, 0, len(ops))
	for _, op := range ops {
		if !op.Type.IsValid() {
			continue
		}
		if op.ResetChannel != 0 && !issuedAtOpen(op.ResetChannel) {
			op.ResetChannel = 0
		}
		kept = append(kept, op)
	}
	return kept, len(kept) < len(ops)
}

// droppedStatus returns the warning that the upgrade dropped operations of the channel
// named name. It has no owner, since the graph writes the status of a calculation.
func droppedStatus(name string) Status {
	return Status{
		Name:    name,
		Variant: status.VariantWarning,
		Message: "Dropped operations with no type",
		Description: fmt.Sprintf(
			"The upgrade dropped the stored operations of %s that had no type. Add "+
				"them again to keep them.",
			name,
		),
		Time: telem.Now(),
	}
}
