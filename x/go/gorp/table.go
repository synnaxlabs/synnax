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
	"bytes"
	"context"
	"io"
	"iter"
	"sort"
	"strings"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/types"
	"go.uber.org/zap"
)

// TableConfig configures a Table opened via OpenTable.
type TableConfig[E any] struct {
	DB         *DB
	Codec      binary.Codec
	Migrations []Migration
	alamos.Instrumentation
}

// Table provides a strongly typed interface for a specific entry type within a gorp DB.
// It holds a resolved Codec (either custom or the DB's default) and provides methods
// for creating query builders that are automatically configured with the codec.
type Table[K Key, E Entry[K]] struct {
	codec binary.Codec
	DB    *DB
}

// Codec returns the table's codec.
func (t *Table[K, E]) Codec() binary.Codec { return t.codec }

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
	codec := resolveCodec(cfg.Codec, cfg.DB)
	tableName := types.Name[E]()
	prefix := []byte(magicPrefix + tableName)
	versionKey := []byte(migrationVersionPrefix + tableName)
	// Prepend built-in migrations that use the DB's default codec. These run
	// once and are tracked alongside user migrations. They must execute before
	// any codec transitions so that all entries are under the current prefix
	// and key encoding.
	builtIn := []Migration{
		&oldPrefixMigration[K, E]{dbCodec: cfg.DB},
		&reEncodeKeysMigration[K, E]{dbCodec: cfg.DB},
	}
	// Wrap user migrations to depend on reencode_keys so they always run
	// after the built-in key normalization.
	wrapped := make([]Migration, len(cfg.Migrations))
	for i, m := range cfg.Migrations {
		wrapped[i] = &afterBuiltIn{Migration: m}
	}
	migrations := append(builtIn, wrapped...)

	kvTx := cfg.DB.KV().OpenTx()
	defer func() {
		err = errors.Combine(err, kvTx.Close())
	}()
	migCfg := MigrationConfig{Prefix: prefix, DBCodec: cfg.DB, L: cfg.L}
	applied, err := readAppliedMigrations(ctx, kvTx, versionKey)
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
			appliedNames := make([]string, 0, len(applied))
			for name := range applied {
				appliedNames = append(appliedNames, name)
			}
			sort.Strings(appliedNames)
			cfg.L.Debug(
				"already applied",
				zap.String("table", tableName),
				zap.Strings("applied", appliedNames),
			)
		}
		for _, m := range pending {
			mStart := time.Now()
			if err := m.Run(ctx, kvTx, migCfg); err != nil {
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
			applied[m.Name()] = true
			if err := writeAppliedMigrations(ctx, kvTx, versionKey, applied); err != nil {
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
	if err = kvTx.Commit(ctx); err != nil {
		return nil, err
	}
	return &Table[K, E]{codec: codec, DB: cfg.DB}, nil
}

// resolveCodec returns the override codec if non-nil, otherwise falls back to the
// fallback codec.
func resolveCodec(override binary.Codec, fallback binary.Codec) binary.Codec {
	if override != nil {
		return override
	}
	return fallback
}

// NewCreate returns a Create query builder configured with this table's codec.
func (t *Table[K, E]) NewCreate() Create[K, E] {
	return Create[K, E]{entries: new(Entries[K, E]), codec: t.codec}
}

// NewRetrieve returns a Retrieve query builder configured with this table's codec.
func (t *Table[K, E]) NewRetrieve() Retrieve[K, E] {
	return Retrieve[K, E]{entries: new(Entries[K, E]), codec: t.codec}
}

// NewUpdate returns an Update query builder configured with this table's codec.
func (t *Table[K, E]) NewUpdate() Update[K, E] {
	return Update[K, E]{retrieve: t.NewRetrieve(), codec: t.codec}
}

// NewDelete returns a Delete query builder configured with this table's codec.
func (t *Table[K, E]) NewDelete() Delete[K, E] {
	return Delete[K, E]{retrieve: t.NewRetrieve(), codec: t.codec}
}

// OpenNexter opens a new Nexter over entries in the table using the table's codec for
// decoding.
func (t *Table[K, E]) OpenNexter(ctx context.Context) (iter.Seq[E], io.Closer, error) {
	return wrapReader[K, E](t.DB, t.codec).OpenNexter(ctx)
}

// migrateOldPrefixKeys finds entries stored under the old codec-based prefix
// (e.g. msgpack-encoded type name) and re-writes them under the new prefix
// format (__gorp__//TypeName).
// oldPrefixMigration moves entries from the old codec-encoded prefix to the
// current literal prefix. It copies raw value bytes without decoding them,
// so it is independent of the value codec.
type oldPrefixMigration[K Key, E Entry[K]] struct {
	dbCodec binary.Codec
}

func (m *oldPrefixMigration[K, E]) Name() string { return "migrate_old_prefix_keys" }

func (m *oldPrefixMigration[K, E]) Run(ctx context.Context, tx kv.Tx, _ MigrationConfig) (err error) {
	dbTx := WrapTx(tx, m.dbCodec)
	oldPrefix, err := dbTx.Encode(ctx, types.Name[E]())
	if err != nil {
		return err
	}
	newKeyCodec := newKeyCodec[K, E]()
	if string(oldPrefix) == string(newKeyCodec.prefix) {
		return nil
	}
	iter, err := tx.OpenIterator(kv.IterPrefix(oldPrefix))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	type move struct {
		oldKey, newKey, rawValue []byte
	}
	var moves []move
	for iter.First(); iter.Valid(); iter.Next() {
		oldKey := bytes.Clone(iter.Key())
		rawValue := bytes.Clone(iter.Value())
		keySuffix := oldKey[len(oldPrefix):]
		newKey := make([]byte, len(newKeyCodec.prefix)+len(keySuffix))
		copy(newKey, newKeyCodec.prefix)
		copy(newKey[len(newKeyCodec.prefix):], keySuffix)
		moves = append(moves, move{oldKey: oldKey, newKey: newKey, rawValue: rawValue})
	}
	for _, mv := range moves {
		if err = tx.Delete(ctx, mv.oldKey); err != nil {
			return err
		}
		if err = tx.Set(ctx, mv.newKey, mv.rawValue); err != nil {
			return err
		}
	}
	return err
}

// reEncodeKeysMigration iterates entries under the current prefix and re-writes
// them to ensure the key portion uses the current primitive encoding. It decodes
// values with the DB's default codec to extract GorpKey(), but writes the raw
// value bytes back without re-encoding.
type reEncodeKeysMigration[K Key, E Entry[K]] struct {
	dbCodec binary.Codec
}

func (m *reEncodeKeysMigration[K, E]) Name() string { return "reencode_keys" }

func (m *reEncodeKeysMigration[K, E]) Dependencies() []string {
	return []string{"migrate_old_prefix_keys"}
}

func (m *reEncodeKeysMigration[K, E]) Run(ctx context.Context, tx kv.Tx, _ MigrationConfig) (err error) {
	newKeyCodec := newKeyCodec[K, E]()
	iter, err := tx.OpenIterator(kv.IterPrefix(newKeyCodec.prefix))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	// Collect all re-key operations first to avoid mutating during iteration.
	type rekey struct {
		oldKey   []byte
		newKey   []byte
		rawValue []byte
	}
	var rekeys []rekey
	for iter.First(); iter.Valid(); iter.Next() {
		oldKey := bytes.Clone(iter.Key())
		rawValue := bytes.Clone(iter.Value())
		var entry E
		if err = m.dbCodec.Decode(ctx, rawValue, &entry); err != nil {
			return errors.Wrapf(err, "reencode_keys: failed to decode entry at key %x", oldKey)
		}
		newKey := bytes.Clone(newKeyCodec.encode(entry.GorpKey()))
		if !bytes.Equal(oldKey, newKey) {
			rekeys = append(rekeys, rekey{oldKey: oldKey, newKey: newKey, rawValue: rawValue})
		}
	}
	for _, r := range rekeys {
		if err = tx.Delete(ctx, r.oldKey); err != nil {
			return err
		}
		if err = tx.Set(ctx, r.newKey, r.rawValue); err != nil {
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
	return append(deps, "reencode_keys")
}

// readAppliedMigrations reads the set of applied migration names from the KV
// store. Names are stored as a newline-delimited string.
func readAppliedMigrations(
	ctx context.Context,
	kvTx kv.Tx,
	key []byte,
) (map[string]bool, error) {
	b, closer, err := kvTx.Get(ctx, key)
	if err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return make(map[string]bool), nil
		}
		return nil, err
	}
	defer func() {
		err = errors.Combine(err, closer.Close())
	}()
	names := strings.Split(string(b), "\n")
	applied := make(map[string]bool, len(names))
	for _, name := range names {
		if name != "" {
			applied[name] = true
		}
	}
	return applied, err
}

// writeAppliedMigrations persists the set of applied migration names as a
// newline-delimited string.
func writeAppliedMigrations(
	ctx context.Context,
	kvTx kv.Tx,
	key []byte,
	applied map[string]bool,
) error {
	names := make([]string, 0, len(applied))
	for name := range applied {
		names = append(names, name)
	}
	sort.Strings(names)
	return kvTx.Set(ctx, key, []byte(strings.Join(names, "\n")))
}
