// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pagerduty

import (
	"context"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v2 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
)

// TaskNames returns the name of each task in keys, by key. It returns an error when a
// key names no task.
type TaskNames func(
	ctx context.Context,
	tx gorp.Tx,
	keys []uuid.UUID,
) (map[uuid.UUID]string, error)

// SecretMigration returns the migration that moves each staged plain routing key into a
// new secret named after the task of its config, and deletes the staged key. It makes
// no secret for a config that no longer exists. It reads task names through names, so
// it must run after the task table migrates.
func (s *Service) SecretMigration(names TaskNames) gorp.Migration {
	return migrate.New(
		"v62_pagerduty_routing_key_secrets",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			staged, err := stagedRoutingKeys(tx)
			if err != nil || len(staged) == 0 {
				return err
			}
			for key := range staged {
				if err = tx.Delete(ctx, v2.PlainRoutingKeyKVKey(key)); err != nil {
					return err
				}
			}
			var cfgs []TaskConfig
			if err = gorp.NewRetrieve[uuid.UUID, TaskConfig]().
				Where(gorp.MatchKeys[uuid.UUID, TaskConfig](lo.Keys(staged)...)).
				SkipMissingKeys().
				Entries(&cfgs).
				Exec(ctx, tx); err != nil || len(cfgs) == 0 {
				return err
			}
			taskNames, err := names(
				ctx,
				tx,
				lo.Map(cfgs, func(c TaskConfig, _ int) uuid.UUID { return c.Key }),
			)
			if err != nil {
				return errors.Wrap(err, "tasks of PagerDuty configs")
			}
			for i := range cfgs {
				cfg := &cfgs[i]
				cfg.PlainRoutingKey = new(staged[cfg.Key])
				if err = v2.ResolveTaskConfig(
					s.create,
					taskNames[cfg.Key],
					cfg,
				)(ctx, tx); err != nil {
					return errors.Wrapf(err, "config %s", cfg.Key)
				}
			}
			return gorp.WrapWriter[uuid.UUID, TaskConfig](tx).Set(ctx, cfgs...)
		},
	)
}

func stagedRoutingKeys(tx gorp.Tx) (map[uuid.UUID]string, error) {
	iter, err := tx.OpenIterator(kv.IterPrefix([]byte(v2.PlainRoutingKeyKVPrefix)))
	if err != nil {
		return nil, err
	}
	staged := map[uuid.UUID]string{}
	for iter.First(); iter.Valid(); iter.Next() {
		key, err := uuid.Parse(
			strings.TrimPrefix(string(iter.Key()), v2.PlainRoutingKeyKVPrefix),
		)
		if err != nil {
			return nil, errors.Combine(err, iter.Close())
		}
		staged[key] = string(iter.Value())
	}
	if err = errors.Combine(iter.Error(), iter.Close()); err != nil {
		return nil, err
	}
	return staged, nil
}
