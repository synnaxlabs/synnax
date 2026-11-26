// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package config

import (
	"context"
	"encoding/json/v2"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
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

// StagedMigration returns the migration named name that moves each value staged as JSON
// under prefix plus a config key into the config with that key, and deletes the staged
// values. move moves staged into cfg, whose task is named task. A staged value whose
// config no longer exists is deleted and moves nothing. The migration reads task names
// through names, so it must run after the task table migrates.
func StagedMigration[E gorp.Entry[uuid.UUID], V any](
	name string,
	prefix string,
	names TaskNames,
	move func(
		ctx context.Context,
		tx gorp.Tx,
		ins alamos.Instrumentation,
		task string,
		cfg *E,
		staged V,
	) error,
) gorp.Migration {
	return migrate.New(
		name,
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			staged, err := stagedValues[V](tx, prefix)
			if err != nil || len(staged) == 0 {
				return err
			}
			for key := range staged {
				if err = tx.Delete(ctx, []byte(prefix+key.String())); err != nil {
					return err
				}
			}
			var cfgs []E
			if err = gorp.NewRetrieve[uuid.UUID, E]().
				Where(gorp.MatchKeys[uuid.UUID, E](lo.Keys(staged)...)).
				SkipMissingKeys().
				Entries(&cfgs).
				Exec(ctx, tx); err != nil || len(cfgs) == 0 {
				return err
			}
			tasks, err := names(
				ctx,
				tx,
				lo.Map(cfgs, func(c E, _ int) uuid.UUID { return c.GorpKey() }),
			)
			if err != nil {
				return errors.Wrap(err, "task names")
			}
			for i := range cfgs {
				key := cfgs[i].GorpKey()
				err = move(ctx, tx, ins, tasks[key], &cfgs[i], staged[key])
				if err != nil {
					return errors.Wrapf(err, "config %s", key)
				}
			}
			return gorp.WrapWriter[uuid.UUID, E](tx).Set(ctx, cfgs...)
		},
	)
}

func stagedValues[V any](tx gorp.Tx, prefix string) (map[uuid.UUID]V, error) {
	iter, err := tx.OpenIterator(kv.IterPrefix([]byte(prefix)))
	if err != nil {
		return nil, err
	}
	staged := map[uuid.UUID]V{}
	for iter.First(); iter.Valid(); iter.Next() {
		key, err := uuid.Parse(strings.TrimPrefix(string(iter.Key()), prefix))
		if err != nil {
			return nil, errors.Combine(err, iter.Close())
		}
		var v V
		if err = json.Unmarshal(iter.Value(), &v); err != nil {
			err = errors.Wrapf(err, "config %s", key)
			return nil, errors.Combine(err, iter.Close())
		}
		staged[key] = v
	}
	if err = errors.Combine(iter.Error(), iter.Close()); err != nil {
		return nil, err
	}
	return staged, nil
}
