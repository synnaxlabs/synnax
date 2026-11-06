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
	"github.com/synnaxlabs/gorp/internal/progress"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// DropDangling removes from e each target of the table's references that does not
// exist in tx, through the Detach of the reference that holds it. It keeps a missing
// target whose reference has no Detach, which the write check of e then rejects. A
// target whose key is not in the text form of its table's keys does not exist.
// DropDangling reports whether it removed a target.
func (t *Table[K, E]) DropDangling(ctx context.Context, tx Tx, e *E) (bool, error) {
	return t.dropDangling(ctx, tx, e, func(Reference[K, E], Target) {})
}

// dropDangling is DropDangling that calls kept with each missing target it keeps.
func (t *Table[K, E]) dropDangling(
	ctx context.Context,
	tx Tx,
	e *E,
	kept func(Reference[K, E], Target),
) (bool, error) {
	dropped := false
	for _, ref := range t.references {
		for _, target := range ref.targets(e) {
			table, err := t.db.registry.table(target.Table)
			if err != nil {
				return false, err
			}
			if table.canonical(target.Key) {
				ok, err := table.exists(ctx, tx, target.Key)
				if err != nil {
					return false, err
				}
				if ok {
					continue
				}
			}
			if ref.detachTarget(e, target) {
				dropped = true
			} else {
				kept(ref, target)
			}
		}
	}
	return dropped, nil
}

// DanglingMigration returns a migration identified by key that runs DropDangling on
// every entry of the table, writes each entry it changes, and logs each missing target
// it keeps. It decodes entries as E, so it runs after the migrations that convert them
// to E.
func (t *Table[K, E]) DanglingMigration(key string) Migration {
	return danglingMigration{key: key, run: t.dropAllDangling}
}

// DanglingMigration returns Table.DanglingMigration for the table that a TableConfig
// opens. It goes in that config's Migrations, and fails when it runs anywhere else.
func DanglingMigration(key string) Migration { return danglingMigration{key: key} }

type danglingMigration struct {
	key string
	run func(context.Context, Tx, alamos.Instrumentation, string) error
}

var _ Migration = danglingMigration{}

// Key implements Migration.
func (m danglingMigration) Key() string { return m.key }

// Run implements Migration.
func (m danglingMigration) Run(
	ctx context.Context,
	tx Tx,
	ins alamos.Instrumentation,
) error {
	if m.run == nil {
		return errors.Newf(
			"[gorp] - dangling migration %s runs only in the config of its table",
			m.key,
		)
	}
	return m.run(ctx, tx, ins, m.key)
}

// bindDangling returns migrations with each unbound dangling migration bound to t.
func (t *Table[K, E]) bindDangling(migrations []Migration) []Migration {
	out := make([]Migration, len(migrations))
	for i, m := range migrations {
		if d, ok := m.(danglingMigration); ok && d.run == nil {
			m = t.DanglingMigration(d.key)
		}
		out[i] = m
	}
	return out
}

func (t *Table[K, E]) dropAllDangling(
	ctx context.Context,
	tx Tx,
	ins alamos.Instrumentation,
	key string,
) (err error) {
	var (
		writer = WrapWriter[K, E](tx)
		logger = &progress.Logger{Instrumentation: ins, Key: key}
	)
	iter, err := WrapReader[K, E](tx).OpenIterator(IterOptions{})
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	var dropped bool
	for iter.First(); iter.Valid(); iter.Next() {
		e := iter.Value(ctx)
		if err = iter.Error(); err != nil {
			return err
		}
		if dropped, err = t.dropDangling(ctx, tx, e, func(
			ref Reference[K, E],
			target Target,
		) {
			ins.L.Warn(
				"kept a missing target that its reference cannot drop",
				zap.String("migration", key),
				zap.String("table", t.name),
				zap.String("entry", FormatKey((*e).GorpKey())),
				zap.String("reference", ref.referenceName()),
				zap.String("target", target.Table+" "+target.Key),
			)
		}); err != nil {
			return err
		}
		if dropped {
			if err = writer.Set(ctx, *e); err != nil {
				return err
			}
		}
		logger.Increment()
	}
	return nil
}
