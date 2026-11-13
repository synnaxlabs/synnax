// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// Migration moves each stored task to the v3 shape, which drops the snapshot flag.
var Migration = migrate.NewEntry("v61_task_drop_snapshot", autoMigrateTask)

// NewSnapshotMigration returns a migration that deletes each stored v2 snapshot task
// with its config record and the statuses it owns, and logs a warning for each. It
// returns an error when a snapshot has a type with no config store.
func NewSnapshotMigration(cfg MigrationConfig) gorp.Migration {
	return migrate.New(
		"v61_task_delete_snapshots",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var tasks []v2.Task
			if err := gorp.NewRetrieve[v2.Key, v2.Task]().
				Entries(&tasks).
				Exec(ctx, tx); err != nil {
				return err
			}
			rows := gorp.WrapWriter[v2.Key, v2.Task](tx)
			statuses := cfg.Status.NewWriter(tx)
			for _, t := range tasks {
				if !t.Snapshot {
					continue
				}
				store, ok := cfg.Configs.Store(t.Type)
				if !ok {
					return errors.Newf(
						"unknown type %q of snapshot task %s",
						t.Type,
						t.Key,
					)
				}
				if err := store.Delete(ctx, tx, t.Key); err != nil {
					return err
				}
				if err := rows.Delete(ctx, t.Key); err != nil {
					return err
				}
				if err := statuses.DeleteOwned(
					ctx,
					resource.ID{Type: resource.TypeTask, Key: t.Key.String()},
				); err != nil {
					return err
				}
				ins.L.Warn(
					"deleted a snapshot task",
					zap.String("key", t.Key.String()),
					zap.String("name", t.Name),
					zap.String("type", t.Type),
				)
			}
			return nil
		},
	)
}
