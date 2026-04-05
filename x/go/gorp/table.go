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
	"encoding/json"
	"io"
	"iter"
	"sort"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/types"
	"go.uber.org/zap"
)

// TableConfig configures a Table opened via OpenTable.
type TableConfig[E any] struct {
	DB         *DB
	Migrations []Migration
	alamos.Instrumentation
}

// Table provides a strongly typed interface for a specific entry type within a gorp DB.
type Table[K Key, E Entry[K]] struct {
	DB *DB
}

func (t *Table[K, E]) Close() error {
	return nil
}

// OpenTable creates or opens a table for the given entry type. It runs any provided
// versioned migrations followed by key migrations to ensure entries are stored under
// the current prefix and key encoding format.
func OpenTable[K Key, E Entry[K]](
	ctx context.Context,
	cfg TableConfig[E],
) (_ *Table[K, E], err error) {
	tableName := types.Name[E]()
	versionKey := []byte(migrationVersionPrefix + tableName)
	// Prepend built-in migrations that use the DB's default codec. These run
	// once and are tracked alongside user migrations. They must execute before
	// any codec transitions so that all entries are under the current prefix
	// and key encoding.
	builtIn := []Migration{&normalizeKeysMigration[K, E]{}}
	// Wrap user migrations to depend on normalize_keys so they always run
	// after key normalization.
	wrapped := make([]Migration, len(cfg.Migrations))
	for i, m := range cfg.Migrations {
		wrapped[i] = &afterBuiltIn{Migration: m}
	}
	migrations := append(builtIn, wrapped...)

	tx := cfg.DB.OpenTx()
	defer func() {
		err = errors.Combine(err, tx.Close())
	}()
	applied, err := readAppliedMigrations(ctx, tx, versionKey)
	if err != nil {
		return nil, err
	}
	pending, err := topoSort(migrations, applied)
	if err != nil {
		return nil, err
	}
	if len(pending) > 0 {
		totalStart := time.Now()
		cfg.L.Info(
			"starting migrations",
			zap.String("table", tableName),
			zap.Int("pending", len(pending)),
		)
		if len(applied) > 0 {
			appliedNames := applied.Slice()
			sort.Strings(appliedNames)
			cfg.L.Debug(
				"already applied",
				zap.String("table", tableName),
				zap.Strings("applied", appliedNames),
			)
		}
		for _, m := range pending {
			mStart := time.Now()
			if err := m.Run(ctx, tx, cfg.Instrumentation); err != nil {
				entries := 0
				if ec, ok := m.(EntryCounter); ok {
					entries = ec.EntriesProcessed()
				}
				cfg.L.Error(
					"migration failed",
					zap.String("table", tableName),
					zap.String("migration", m.Name()),
					zap.Int("entries_processed", entries),
					zap.Duration("elapsed", time.Since(mStart)),
					zap.Error(err),
				)
				return nil, errors.Wrapf(err, "migration (%s) failed", m.Name())
			}
			entries := 0
			if ec, ok := m.(EntryCounter); ok {
				entries = ec.EntriesProcessed()
			}
			cfg.L.Info(
				"migration complete",
				zap.String("table", tableName),
				zap.String("migration", m.Name()),
				zap.Int("entries", entries),
				zap.Duration("elapsed", time.Since(mStart)),
			)
			applied.Add(m.Name())
			if err := writeAppliedMigrations(ctx, tx, versionKey, applied); err != nil {
				return nil, err
			}
		}
		cfg.L.Info(
			"migrations complete",
			zap.String("table", tableName),
			zap.Int("migrations", len(pending)),
			zap.Duration("elapsed", time.Since(totalStart)),
		)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &Table[K, E]{DB: cfg.DB}, nil
}

// NewCreate returns a Create query builder.
func (t *Table[K, E]) NewCreate() Create[K, E] {
	return NewCreate[K, E]()
}

// NewRetrieve returns a Retrieve query builder.
func (t *Table[K, E]) NewRetrieve() Retrieve[K, E] {
	return NewRetrieve[K, E]()
}

// NewUpdate returns an Update query builder.
func (t *Table[K, E]) NewUpdate() Update[K, E] {
	return NewUpdate[K, E]()
}

// NewDelete returns a Delete query builder.
func (t *Table[K, E]) NewDelete() Delete[K, E] {
	return NewDelete[K, E]()
}

// OpenNexter opens a new Nexter over entries in the table using the DB's codec for
// decoding.
func (t *Table[K, E]) OpenNexter(ctx context.Context) (iter.Seq[E], io.Closer, error) {
	return WrapReader[K, E](t.DB).OpenNexter(ctx)
}

// normalizeKeysMigration moves entries from the old codec-encoded prefix to the
// current literal prefix and re-encodes key suffixes to use primitive encoding.
// Values are decoded with the DB's default codec only to extract GorpKey(); raw
// value bytes are written back without re-encoding.
type normalizeKeysMigration[K Key, E Entry[K]] struct{}

func (m *normalizeKeysMigration[K, E]) Name() string { return "normalize_keys" }

func (m *normalizeKeysMigration[K, E]) Run(ctx context.Context, tx Tx, _ alamos.Instrumentation) error {
	kc := newKeyCodec[K, E]()
	oldPrefix, err := msgpack.Codec.Encode(ctx, types.Name[E]())
	if err != nil {
		return err
	}
	// Phase 1: move entries from old prefix to new prefix + new key encoding.
	if string(oldPrefix) != string(kc.prefix) {
		itr, err := tx.OpenIterator(kv.IterPrefix(oldPrefix))
		if err != nil {
			return err
		}
		for itr.First(); itr.Valid(); itr.Next() {
			rawValue := itr.Value()
			var entry E
			if err = tx.Decode(ctx, rawValue, &entry); err != nil {
				return errors.Combine(
					errors.Wrapf(err, "normalize_keys: failed to decode entry at old prefix key %x", itr.Key()),
					itr.Close(),
				)
			}
			if err = tx.Delete(ctx, itr.Key()); err != nil {
				return errors.Combine(err, itr.Close())
			}
			if err = tx.Set(ctx, kc.encode(entry.GorpKey()), rawValue); err != nil {
				return errors.Combine(err, itr.Close())
			}
		}
		if err = itr.Close(); err != nil {
			return err
		}
	}
	return nil
}

// afterBuiltIn wraps a user-provided Migration so that it depends on the
// built-in reencode_keys migration. This guarantees all built-in key
// normalization completes before any user codec transitions run.
type afterBuiltIn struct {
	Migration
}

func (a *afterBuiltIn) Dependencies() []string {
	var deps []string
	if dd, ok := a.Migration.(DependencyDeclarer); ok {
		deps = dd.Dependencies()
	}
	return append(deps, "normalize_keys")
}

// readAppliedMigrations reads the set of applied migration names from the KV
// store. Names are stored as a newline-delimited string.
func readAppliedMigrations(
	ctx context.Context,
	tx Tx,
	key []byte,
) (_ set.Set[string], err error) {
	b, closer, getErr := tx.Get(ctx, key)
	if getErr != nil {
		if errors.Is(getErr, query.ErrNotFound) {
			return make(set.Set[string]), nil
		}
		return nil, getErr
	}
	defer func() {
		err = errors.Combine(err, closer.Close())
	}()
	var names []string
	if err := json.Unmarshal(b, &names); err != nil {
		return nil, err
	}
	applied := make(set.Set[string], len(names))
	for _, name := range names {
		applied.Add(name)
	}
	return applied, nil
}

// writeAppliedMigrations persists the set of applied migration names as a
// JSON string array.
func writeAppliedMigrations(
	ctx context.Context,
	tx Tx,
	key []byte,
	applied set.Set[string],
) error {
	names := applied.Slice()
	sort.Strings(names)
	b, err := json.Marshal(names)
	if err != nil {
		return err
	}
	return tx.Set(ctx, key, b)
}
