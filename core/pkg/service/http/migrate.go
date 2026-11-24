// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"context"
	"encoding/json/v2"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
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

// ICDMigration returns the migration that moves the staged fields of each read config
// into an ICD named after its task, and deletes the staged fields. It makes no ICD for
// a config that no longer exists. It reads task names through names, so it must run
// after the task table migrates. A config with no task fails it, and no store holds
// one: each task delete cascades to the config of the task.
func (s *Service) ICDMigration(names TaskNames) gorp.Migration {
	return migrate.New(
		"v63_http_read_icd",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			staged, err := stagedFields(tx)
			if err != nil || len(staged) == 0 {
				return err
			}
			for key := range staged {
				if err = tx.Delete(ctx, v5.StagedFieldsKVKey(key)); err != nil {
					return err
				}
			}
			var cfgs []ReadConfig
			if err = gorp.NewRetrieve[uuid.UUID, ReadConfig]().
				Where(gorp.MatchKeys[uuid.UUID, ReadConfig](lo.Keys(staged)...)).
				SkipMissingKeys().
				Entries(&cfgs).
				Exec(ctx, tx); err != nil || len(cfgs) == 0 {
				return err
			}
			taskNames, err := names(
				ctx,
				tx,
				lo.Map(cfgs, func(c ReadConfig, _ int) uuid.UUID { return c.Key }),
			)
			if err != nil {
				return errors.Wrap(err, "tasks of HTTP read configs")
			}
			for i := range cfgs {
				cfg := &cfgs[i]
				cfg.Inline = new(staged[cfg.Key])
				ic, _, err := v5.ResolveReadConfig(
					ctx,
					tx,
					s.resolver.channels,
					taskNames[cfg.Key],
					cfg,
					v5.NoteLift(ins, cfg.Key),
				)
				if err != nil {
					return errors.Wrapf(err, "config %s", cfg.Key)
				}
				if err = s.resolver.icds.NewWriter(tx).Create(ctx, &ic); err != nil {
					return errors.Wrapf(err, "config %s", cfg.Key)
				}
			}
			return gorp.WrapWriter[uuid.UUID, ReadConfig](tx).Set(ctx, cfgs...)
		},
	)
}

func stagedFields(tx gorp.Tx) (map[uuid.UUID]v5.InlineRead, error) {
	iter, err := tx.OpenIterator(kv.IterPrefix([]byte(v5.StagedFieldsKVPrefix)))
	if err != nil {
		return nil, err
	}
	staged := map[uuid.UUID]v5.InlineRead{}
	for iter.First(); iter.Valid(); iter.Next() {
		key, err := uuid.Parse(
			strings.TrimPrefix(string(iter.Key()), v5.StagedFieldsKVPrefix),
		)
		if err != nil {
			return nil, errors.Combine(err, iter.Close())
		}
		var fields v5.InlineRead
		if err = json.Unmarshal(iter.Value(), &fields); err != nil {
			err = errors.Wrapf(err, "config %s", key)
			return nil, errors.Combine(err, iter.Close())
		}
		staged[key] = fields
	}
	if err = errors.Combine(iter.Error(), iter.Close()); err != nil {
		return nil, err
	}
	return staged, nil
}
