// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package migrate builds the migrations that a Gorp table runs when it opens. Pass them
// to gorp.TableConfig.Migrations or gorp.MigrateConfig.Migrations.
package migrate

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

type migration struct {
	key string
	run func(context.Context, gorp.Tx, alamos.Instrumentation) error
}

var _ gorp.Migration = migration{}

// Key implements gorp.Migration.
func (m migration) Key() string { return m.key }

// Run implements gorp.Migration.
func (m migration) Run(
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
) error {
	return m.run(ctx, tx, ins)
}

// New returns a migration identified by key that runs fn against the migration's
// transaction.
func New(
	key string,
	fn func(context.Context, gorp.Tx, alamos.Instrumentation) error,
) gorp.Migration {
	return migration{key: key, run: fn}
}

// NewEntry returns a migration identified by key that decodes every entry of type I,
// transforms it to type O, and writes the result. Decoding and encoding use the DB's
// codec.
func NewEntry[IK, OK gorp.Key, I gorp.Entry[IK], O gorp.Entry[OK]](
	key string,
	transform func(context.Context, I) (O, error),
) gorp.Migration {
	return NewInstrumentedEntry(
		key,
		func(ctx context.Context, old I, _ alamos.Instrumentation) (O, error) {
			return transform(ctx, old)
		},
	)
}

// NewInstrumentedEntry is NewEntry for a transform that reports through the runner's
// instrumentation, such as one that must log a lossy step.
func NewInstrumentedEntry[IK, OK gorp.Key, I gorp.Entry[IK], O gorp.Entry[OK]](
	key string,
	transform func(context.Context, I, alamos.Instrumentation) (O, error),
) gorp.Migration {
	return New(
		key,
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) (err error) {
			var (
				reader   = gorp.WrapReader[IK, I](tx)
				writer   = gorp.WrapWriter[OK, O](tx)
				logger   = &progressLogger{key: key, ins: ins}
				newEntry O
			)
			iter, err := reader.OpenIterator(gorp.IterOptions{})
			if err != nil {
				return err
			}
			defer func() { err = errors.Combine(err, iter.Close()) }()
			for iter.First(); iter.Valid(); iter.Next() {
				old := iter.Value(ctx)
				if err = iter.Error(); err != nil {
					return err
				}
				newEntry, err = transform(ctx, *old, ins)
				if err != nil {
					return errors.Wrapf(err, "entry %v (transform)", (*old).GorpKey())
				}
				if err = writer.Set(ctx, newEntry); err != nil {
					return err
				}
				logger.increment()
			}
			return err
		},
	)
}

// Codec returns a migration identified by key that re-encodes every entry of type E
// with the DB's codec.
func Codec[K gorp.Key, E gorp.Entry[K]](key string) gorp.Migration {
	return NewEntry[K, K, E, E](
		key,
		func(_ context.Context, e E) (E, error) { return e, nil },
	)
}

const progressMax = 1000

// progressLogger emits periodic info-level progress messages during a long-running
// migration.
type progressLogger struct {
	// ins is the instrumentation used to emit progress logs.
	ins alamos.Instrumentation
	// key identifies the migration the logger is reporting on.
	key string
	// count is the number of entries processed so far.
	count int
}

// increment counts one more processed entry and logs at the progress interval.
func (p *progressLogger) increment() {
	p.count++
	if p.shouldLog() {
		p.ins.L.Info(
			"migration progress",
			zap.String("migration", p.key),
			zap.Int("entries", p.count),
		)
	}
}

// shouldLog returns true at logarithmically spaced intervals (1, 10, 100, 1000) then
// every 1000 entries after that.
func (p *progressLogger) shouldLog() bool {
	if p.count >= progressMax {
		return p.count%progressMax == 0
	}
	n := p.count
	for n >= 10 {
		if n%10 != 0 {
			return false
		}
		n /= 10
	}
	return n == 1
}
