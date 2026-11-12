// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v1 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v1"
	secret "github.com/synnaxlabs/synnax/pkg/service/secret/versions/v0"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	"github.com/synnaxlabs/x/errors"
)

// CreateSecret creates a stored secret with the given key, name, and value through tx.
type CreateSecret func(
	ctx context.Context,
	tx gorp.Tx,
	key secret.Key,
	name string,
	value []byte,
) error

// ResolveTaskConfig points RoutingKey at a new secret that holds the plain routing key
// of cfg, and clears the plain key. It returns the write that creates the secret,
// named "<name> routing key", where name is the name of the task that owns cfg. It
// returns nil for a config without a plain routing key.
func ResolveTaskConfig(
	create CreateSecret,
	name string,
	cfg *TaskConfig,
) func(context.Context, gorp.Tx) error {
	if cfg.PlainRoutingKey == nil {
		return nil
	}
	key, value := uuid.New(), []byte(*cfg.PlainRoutingKey)
	cfg.RoutingKey, cfg.PlainRoutingKey = &key, nil
	return func(ctx context.Context, tx gorp.Tx) error {
		return create(ctx, tx, key, name+" routing key", value)
	}
}

// NewMigration returns the migration that lifts stored v1 configs to v2 and moves each
// plain routing key into a new secret named after the config's task.
func NewMigration(create CreateSecret) gorp.Migration {
	return migrate.New(
		"v61_pagerduty_routing_key_secret",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var olds []v1.TaskConfig
			if err := gorp.NewRetrieve[uuid.UUID, v1.TaskConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil || len(olds) == 0 {
				return err
			}
			var tasks []task.Task
			if err := gorp.NewRetrieve[task.Key, task.Task]().
				Where(gorp.MatchKeys[task.Key, task.Task](
					lo.Map(
						olds,
						func(c v1.TaskConfig, _ int) task.Key { return c.Key },
					)...,
				)).
				Entries(&tasks).
				Exec(ctx, tx); err != nil {
				return errors.Wrap(err, "tasks of PagerDuty configs")
			}
			names := lo.SliceToMap(tasks, func(t task.Task) (task.Key, string) {
				return t.Key, t.Name
			})
			news := make([]TaskConfig, len(olds))
			for i, old := range olds {
				cfg, err := MigrateTaskConfig(ctx, old)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				if write := ResolveTaskConfig(
					create,
					names[old.Key],
					&cfg,
				); write != nil {
					if err = write(ctx, tx); err != nil {
						return errors.Wrapf(err, "config %s", old.Key)
					}
				}
				news[i] = cfg
			}
			return gorp.WrapWriter[uuid.UUID, TaskConfig](tx).Set(ctx, news...)
		},
	)
}
