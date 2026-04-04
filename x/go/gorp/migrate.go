// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// Migration is a versioned schema migration that transforms entries stored in gorp.
type Migration struct {
	key          string
	tx           Tx
	dependencies set.Set[string]
	fn           func(ctx context.Context, tx Tx, ins alamos.Instrumentation) error
}

var _ migrate.Migration = (*Migration)(nil)

func (m *Migration) Key() string { return m.key }

func (m *Migration) Dependencies() set.Set[string] { return m.dependencies }

func (m *Migration) Run(ctx context.Context, ins alamos.Instrumentation) error {
	return m.fn(ctx, m.tx, ins)
}

const migrationProgressMax = 1000

// shouldLogProgress returns true at logarithmically spaced intervals
// (1, 10, 100, 1000) then every 1000 entries after that.
func shouldLogProgress(entries int) bool {
	if entries <= 0 {
		return false
	}
	if entries >= migrationProgressMax {
		return entries%migrationProgressMax == 0
	}
	for entries >= 10 {
		if entries%10 != 0 {
			return false
		}
		entries /= 10
	}
	return entries == 1
}

// NewEntryMigration creates a Migration that iterates over all entries with the
// configured prefix, decodes each as type I, transforms it to type O via the
// transform function, and encodes the result. Both decoding and encoding use
// the DB's codec from MigrationContext.
func NewEntryMigration[IK Key, OK Key, I Entry[IK], O Entry[OK]](
	key string,
	transform func(ctx context.Context, old I) (O, error),
) *Migration {
	return NewMigration(key, func(ctx context.Context, tx Tx, ins alamos.Instrumentation) error {
		var (
			reader    = WrapReader[IK, I](tx)
			writer    = WrapWriter[OK, O](tx)
			entries   int
			iter, err = reader.OpenIterator(IterOptions{})
		)
		if err != nil {
			return err
		}
		defer func() {
			err = errors.Combine(err, iter.Close())
		}()
		for iter.First(); iter.Valid(); iter.Next() {
			entries++
			if shouldLogProgress(entries) {
				ins.L.Info(
					"migration progress",
					zap.String("migration", key),
					zap.Int("entries", entries),
				)
			}
			old := iter.Value(ctx)
			if old == nil {
				if err := iter.Error(); err != nil {
					return err
				}
				continue
			}
			newEntry, err := transform(ctx, *old)
			if err != nil {
				return errors.Wrapf(err, "entry %v (transform)", (*old).GorpKey())
			}
			if err = writer.Set(ctx, newEntry); err != nil {
				return err
			}
		}
		return err

	})
}

// NewMigration creates a Migration that receives a fully wrapped gorp.Tx,
// allowing arbitrary read/write operations on the store.
func NewMigration(
	key string,
	fn func(ctx context.Context, tx Tx, ins alamos.Instrumentation) error,
	deps ...string,
) *Migration {
	return &Migration{key: key, fn: fn, dependencies: set.New(deps...)}
}
