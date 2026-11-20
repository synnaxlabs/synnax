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
)

// DropDangling removes from e each target of the table's references that does not exist
// in snap, through the Detach of the reference that holds it. For a Weak reference, it
// removes each target that its table never issued and keeps a deleted one. It skips
// each reference with no Detach, whose targets the write check of e verifies. A target
// whose key is not in the text form of its table's keys does not exist. DropDangling
// reports whether it removed a target.
func (t *Table[K, E]) DropDangling(
	ctx context.Context,
	snap Snapshot,
	e *E,
) (bool, error) {
	return t.dropDangling(ctx, snap, e, nil)
}

// IssuedFunc reports whether the table of target issued its key.
type IssuedFunc func(target Target) (bool, error)

// dropDangling is DropDangling, asking issued in place of a weak target's table when
// issued is not nil.
func (t *Table[K, E]) dropDangling(
	ctx context.Context,
	snap Snapshot,
	e *E,
	issued IssuedFunc,
) (bool, error) {
	dropped := false
	for _, ref := range t.references {
		if !ref.canDetach() {
			continue
		}
		weak := ref.onDeletePolicy() == Weak
		for _, target := range ref.targets(e) {
			table, err := t.db.registry.table(target.Table)
			if err != nil {
				return false, err
			}
			if table.canonical(target.Key) {
				var ok bool
				switch {
				case weak && issued != nil:
					ok, err = issued(target)
				case weak:
					ok, err = table.issued(target.Key)
				default:
					ok, err = table.exists(ctx, snap, target.Key)
				}
				if err != nil {
					return false, err
				}
				if ok {
					continue
				}
			}
			ref.detachTarget(e, target)
			dropped = true
		}
	}
	return dropped, nil
}

// DanglingMigration returns a migration identified by key that runs DropDangling on
// every entry of the table and writes each entry it changes. It decodes entries as E,
// so it runs after the migrations that convert them to E.
func (t *Table[K, E]) DanglingMigration(key string) Migration {
	return t.IssuedDanglingMigration(key, nil)
}

// IssuedDanglingMigration is DanglingMigration that asks issued, when it is not nil,
// whether the table of a weak target issued its key.
func (t *Table[K, E]) IssuedDanglingMigration(key string, issued IssuedFunc) Migration {
	return danglingMigration{
		key: key,
		run: func(ctx context.Context, tx Tx, ins alamos.Instrumentation, key string) error {
			return t.dropAllDangling(ctx, tx, ins, key, issued)
		},
	}
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
	issued IssuedFunc,
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
		e := iter.Value()
		if err = iter.Error(); err != nil {
			return err
		}
		if dropped, err = t.dropDangling(ctx, tx, e, issued); err != nil {
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
