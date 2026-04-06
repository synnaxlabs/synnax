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

// migration is a versioned schema migration that transforms entries stored in gorp.
type migration struct {
	key          string
	tx           Tx
	dependencies set.Set[string]
	fn           func(ctx context.Context, tx Tx, ins alamos.Instrumentation) error
}

var _ migrate.Migration = (*migration)(nil)

// Key implements migrate.Migration.
func (m *migration) Key() string { return m.key }

// Dependencies implements migrate.Migration.
func (m *migration) Dependencies() set.Set[string] { return m.dependencies }

// Run implements migrate.Migration.
func (m *migration) Run(ctx context.Context, ins alamos.Instrumentation) error {
	return m.fn(ctx, m.tx, ins)
}

const migrationProgressMax = 1000

type progressLogger struct {
	ins   alamos.Instrumentation
	key   string
	count int
}

func (p *progressLogger) logProgress() {
	if p.shouldLogProgress() {
		p.ins.L.Info(
			"migration progress",
			zap.String("migration", p.key),
			zap.Int("entries", p.count),
		)
	}
}

// shouldLogProgress returns true at logarithmically spaced intervals
// (1, 10, 100, 1000) then every 1000 entries after that.
func (p *progressLogger) shouldLogProgress() bool {
	if p.count <= 0 {
		return false
	}
	if p.count >= migrationProgressMax {
		return p.count%migrationProgressMax == 0
	}
	for p.count >= 10 {
		if p.count%10 != 0 {
			return false
		}
		p.count /= 10
	}
	return p.count == 1
}

// NewEntryMigration creates a migration that iterates over all entries with the
// configured prefix, decodes each as type I, transforms it to type O via the
// transform function, and encodes the result. Both decoding and encoding use
// the DB's codec from MigrationContext.
func NewEntryMigration[IK Key, OK Key, I Entry[IK], O Entry[OK]](
	key string,
	transform func(ctx context.Context, old I) (O, error),
) migrate.Migration {
	return NewMigration(key, func(ctx context.Context, tx Tx, ins alamos.Instrumentation) (err error) {
		var (
			reader   = WrapReader[IK, I](tx)
			writer   = WrapWriter[OK, O](tx)
			logger   = &progressLogger{key: key, ins: ins}
			newEntry O
		)
		iter, err := reader.OpenIterator(IterOptions{})
		if err != nil {
			return err
		}
		defer func() {
			err = errors.Combine(err, iter.Close())
		}()
		for iter.First(); iter.Valid(); iter.Next() {
			logger.logProgress()
			old := iter.Value(ctx)
			if err = iter.Error(); err != nil {
				return err
			}
			newEntry, err = transform(ctx, *old)
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

// NewMigration creates a migration that receives a fully wrapped gorp.Tx,
// allowing arbitrary read/write operations on the store.
//
// The returned migrate.Migration will ONLY work properly when run within the
// context of gorp.OpenTable or gorp.Migrate.
func NewMigration(
	key string,
	fn func(ctx context.Context, tx Tx, ins alamos.Instrumentation) error,
	deps ...string,
) migrate.Migration {
	return &migration{key: key, fn: fn, dependencies: set.New(deps...)}
}

func setMigrationTx(mig migrate.Migration, tx Tx) {
	if tMig, ok := migrate.Unwrap(mig).(*migration); ok {
		tMig.tx = tx
	}
}

type MigrateConfig struct {
	alamos.Instrumentation
	DB         *DB
	Namespace  string
	Migrations []migrate.Migration
}

func Migrate(ctx context.Context, cfg MigrateConfig) (err error) {
	txn := cfg.DB.OpenTx()
	defer func() {
		err = errors.Combine(err, txn.Close())
	}()
	for _, mig := range cfg.Migrations {
		setMigrationTx(mig, txn)
	}
	applied, err := readAppliedMigrations(ctx, txn, cfg.Namespace)
	if err != nil {
		return err
	}
	applied, err = migrate.Migrate(ctx, migrate.Config{
		Migrations:      cfg.Migrations,
		Applied:         applied,
		Instrumentation: cfg.Instrumentation,
	})
	if err != nil {
		return err
	}
	if err = writeAppliedMigrations(ctx, txn, cfg.Namespace, applied); err != nil {
		return err
	}
	return txn.Commit(ctx)
}

func CodecMigration[K Key, E Entry[K]](key string, deps ...string) migrate.Migration {
	return NewMigration(key, func(ctx context.Context, tx Tx, ins alamos.Instrumentation) (err error) {
		writer := WrapWriter[K, E](tx)
		logger := &progressLogger{key: key, ins: ins}
		iter, err := WrapReader[K, E](tx).OpenIterator(IterOptions{})
		if err != nil {
			return err
		}
		defer func() {
			err = errors.Combine(err, iter.Close())
		}()
		for iter.First(); iter.Valid(); iter.Next() {
			v := iter.Value(ctx)
			logger.logProgress()
			if err = iter.Error(); err != nil {
				return err
			}
			if err = writer.Set(ctx, *v); err != nil {
				return err
			}
		}
		return nil
	}, deps...)
}
