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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// UntypedDeleteMigration returns the migration that deletes each stored channel that
// has no data type and no expression, together with its storage. It keeps and logs a
// channel that another entry restricts, so it must run once every table that holds a
// channel reference is open.
func (s *Service) UntypedDeleteMigration() gorp.Migration {
	return migrate.New(
		"v63_delete_untyped_channels",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var untyped []Channel
			if err := s.table.NewRetrieve().
				Where(gorp.Match(func(_ gorp.Context, c *Channel) (bool, error) {
					return c.DataType == telem.UnknownT && !c.IsCalculated(), nil
				})).
				Entries(&untyped).
				Exec(ctx, tx); err != nil {
				return err
			}
			w := s.NewWriter(tx)
			for _, ch := range untyped {
				err := w.Delete(ctx, ch.Key(), true)
				var restricted gorp.RestrictError
				if errors.As(err, &restricted) {
					ins.L.Warn(
						"kept a channel with no data type that another entry references",
						zap.Stringer("key", ch.Key()),
						zap.String("name", ch.Name),
						zap.String(
							"holder",
							restricted.Holder.Table+" "+restricted.Holder.Key,
						),
					)
					continue
				}
				if err != nil {
					return err
				}
				ins.L.Warn(
					"deleted a channel with no data type",
					zap.Stringer("key", ch.Key()),
					zap.String("name", ch.Name),
				)
			}
			return nil
		},
	)
}
