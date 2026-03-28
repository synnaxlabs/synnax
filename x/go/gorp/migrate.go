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
	"fmt"

	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

// Deprecated: Use Migration interface and OpenTable with variadic migrations.
var ErrMigrationCountExceeded = errors.New(
	"migration count is greater than the maximum of 255",
)

// Deprecated: Use Migration interface and OpenTable with variadic migrations.
//
// MigrationSpec defines a single migration that should be run with a transaction.
type MigrationSpec struct {
	// Migrate is the migration function to execute
	Migrate func(context.Context, Tx) error
	// Name is a unique identifier for this migration (e.g., "name_validation")
	Name string
}

// Deprecated: Use Migration interface and OpenTable with variadic migrations.
//
// Migrator executes a series of migrations in order, tracking progress with
// incrementing versions. Migrations are run sequentially from current_version + 1 up to
// the latest version. The version starts at 0 (no migrations) and increments to
// len(Migrations).
type Migrator struct {
	// Key is the KV store key used to track migration version (e.g.,
	// "sy_channel_migration_version").
	Key string
	// Migrations is the ordered list of migrations to run.
	Migrations []MigrationSpec
	// Force, when true, will run all migrations, regardless of whether they have been
	// completed or not. This is useful for testing or debugging.
	Force bool
}

// Deprecated: Use Migration interface and OpenTable with variadic migrations.
//
// Run executes all migrations that haven't been completed yet. Migrations run
// sequentially and the version is incremented after each successful migration.
func (r Migrator) Run(ctx context.Context, db *DB) error {
	return db.WithTx(ctx, func(tx Tx) error {
		migrationCount := len(r.Migrations)
		if migrationCount > 255 {
			return errors.Wrapf(
				ErrMigrationCountExceeded,
				"migration count is greater than the maximum of 255: %d",
				migrationCount,
			)
		}
		var currentVersion uint8
		if !r.Force {
			versionBytes, closer, err := tx.Get(ctx, []byte(r.Key))
			if err := errors.Skip(err, query.ErrNotFound); err != nil {
				return err
			}
			if closer != nil {
				if err := closer.Close(); err != nil {
					return err
				}
			}
			if len(versionBytes) > 0 {
				currentVersion = versionBytes[0]
			}
		}
		for i := currentVersion; i < uint8(migrationCount); i++ {
			migration := r.Migrations[i]
			newVersion := i + 1
			if err := migration.Migrate(ctx, tx); err != nil {
				return errors.Wrapf(
					err,
					"migration %d (%s) failed",
					newVersion,
					migration.Name,
				)
			}
			if err := tx.Set(ctx, []byte(r.Key), []byte{newVersion}); err != nil {
				return errors.Wrapf(
					err,
					"failed to migrate to version %d",
					newVersion,
				)
			}
		}
		return nil
	})
}

// Migration is a versioned schema migration that transforms entries stored in gorp.
type Migration interface {
	// Name returns a human-readable identifier for this migration.
	Name() string
	// Run executes the migration within the provided kv.Tx.
	Run(ctx context.Context, tx kv.Tx, cfg MigrationConfig) error
}

// MigrationConfig provides the configuration needed by Migration implementations
// to locate and encode/decode entries. DBCodec is always the DB's default codec
// (typically msgpack), used for decoding pre-transition legacy data. Individual
// migrations carry their own codecs for version-specific encoding/decoding.
type MigrationConfig struct {
	Prefix  []byte
	DBCodec binary.Codec
}

// TransformFunc transforms an old entry of type I into a new entry of type O.
type TransformFunc[I, O any] func(ctx context.Context, old I) (O, error)

type typedMigration[I, O any] struct {
	name        string
	inputCodec  binary.Codec
	outputCodec binary.Codec
	transform   TransformFunc[I, O]
}

// NewTypedMigration creates a Migration that iterates over all entries with the
// configured prefix, decodes each as type I using inputCodec, transforms it to
// type O via the transform function, and encodes the result using outputCodec.
// If inputCodec is nil, MigrationConfig.DBCodec is used for decoding.
// If outputCodec is nil, MigrationConfig.DBCodec is used for encoding.
func NewTypedMigration[I, O any](
	name string,
	inputCodec binary.Codec,
	outputCodec binary.Codec,
	transform TransformFunc[I, O],
) Migration {
	return &typedMigration[I, O]{
		name:        name,
		inputCodec:  inputCodec,
		outputCodec: outputCodec,
		transform:   transform,
	}
}

func (m *typedMigration[I, O]) Name() string { return m.name }

func (m *typedMigration[I, O]) Run(
	ctx context.Context,
	kvTx kv.Tx,
	cfg MigrationConfig,
) error {
	inCodec := m.inputCodec
	if inCodec == nil {
		inCodec = cfg.DBCodec
	}
	outCodec := m.outputCodec
	if outCodec == nil {
		outCodec = cfg.DBCodec
	}
	iter, err := kvTx.OpenIterator(kv.IterPrefix(cfg.Prefix))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		var old I
		if err = inCodec.Decode(ctx, iter.Value(), &old); err != nil {
			return err
		}
		newEntry, err := m.transform(ctx, old)
		if err != nil {
			return err
		}
		var data []byte
		if data, err = outCodec.Encode(ctx, newEntry); err != nil {
			return err
		}
		if err = kvTx.Set(ctx, iter.Key(), data); err != nil {
			return err
		}
	}
	return err
}

type codecTransitionMigration[K Key, E Entry[K]] struct {
	name  string
	codec binary.Codec
}

// NewCodecTransition creates a Migration that re-encodes all entries from the DB's
// default codec (e.g. msgpack) to the provided target codec (e.g. protobuf).
func NewCodecTransition[K Key, E Entry[K]](name string, codec binary.Codec) Migration {
	return &codecTransitionMigration[K, E]{name: name, codec: codec}
}

func (m *codecTransitionMigration[K, E]) Name() string { return m.name }

func (m *codecTransitionMigration[K, E]) Run(
	ctx context.Context,
	kvTx kv.Tx,
	cfg MigrationConfig,
) error {
	iter, err := kvTx.OpenIterator(kv.IterPrefix(cfg.Prefix))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		var entry E
		if err = cfg.DBCodec.Decode(ctx, iter.Value(), &entry); err != nil {
			return err
		}
		var data []byte
		if data, err = m.codec.Encode(ctx, entry); err != nil {
			return err
		}
		if err = kvTx.Set(ctx, iter.Key(), data); err != nil {
			return err
		}
	}
	return err
}

type rawMigration struct {
	name string
	fn   func(ctx context.Context, tx Tx) error
}

// NewRawMigration creates a Migration that receives a fully wrapped gorp.Tx,
// allowing arbitrary read/write operations on the store.
func NewRawMigration(
	name string,
	fn func(ctx context.Context, tx Tx) error,
) Migration {
	return &rawMigration{name: name, fn: fn}
}

func (m *rawMigration) Name() string { return m.name }

func (m *rawMigration) Run(
	ctx context.Context,
	kvTx kv.Tx,
	cfg MigrationConfig,
) error {
	return m.fn(ctx, WrapTx(kvTx, cfg.DBCodec))
}

// DependencyDeclarer is optionally implemented by Migration values that
// need to run after specific other migrations.
type DependencyDeclarer interface {
	Dependencies() []string
}

// WithDependencies wraps a Migration to declare dependencies on other
// migrations by name.
func WithDependencies(m Migration, deps ...string) Migration {
	return &dependentMigration{Migration: m, deps: deps}
}

type dependentMigration struct {
	Migration
	deps []string
}

func (d *dependentMigration) Dependencies() []string { return d.deps }

// ErrCyclicDependency is returned when migrations form a dependency cycle.
var ErrCyclicDependency = errors.New("cyclic dependency detected in migrations")

// ErrMissingDependency is returned when a migration depends on a name that
// does not exist in the migration list and has not already been applied.
var ErrMissingDependency = errors.New("missing migration dependency")

// topoSort filters out already-applied migrations, then produces a valid
// execution order using Kahn's algorithm. Dependencies that are already applied
// are considered satisfied and do not need to appear in the pending set.
func topoSort(migrations []Migration, applied map[string]bool) ([]Migration, error) {
	byName := make(map[string]Migration, len(migrations))
	for _, m := range migrations {
		byName[m.Name()] = m
	}

	var pending []Migration
	for _, m := range migrations {
		if !applied[m.Name()] {
			pending = append(pending, m)
		}
	}
	if len(pending) == 0 {
		return nil, nil
	}

	hasDeps := false
	for _, m := range pending {
		if _, ok := m.(DependencyDeclarer); ok {
			hasDeps = true
			break
		}
	}
	if !hasDeps {
		return pending, nil
	}

	pendingSet := make(map[string]bool, len(pending))
	for _, m := range pending {
		pendingSet[m.Name()] = true
	}

	inDegree := make(map[string]int, len(pending))
	dependents := make(map[string][]string, len(pending))
	for _, m := range pending {
		name := m.Name()
		if _, exists := inDegree[name]; !exists {
			inDegree[name] = 0
		}
		if dd, ok := m.(DependencyDeclarer); ok {
			for _, dep := range dd.Dependencies() {
				if applied[dep] {
					continue
				}
				if !pendingSet[dep] {
					if _, known := byName[dep]; !known {
						return nil, fmt.Errorf(
							"%w: migration %q depends on %q which does not exist",
							ErrMissingDependency, name, dep,
						)
					}
				}
				inDegree[name]++
				dependents[dep] = append(dependents[dep], name)
			}
		}
	}

	var queue []string
	for _, m := range pending {
		if inDegree[m.Name()] == 0 {
			queue = append(queue, m.Name())
		}
	}

	var sorted []Migration
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		sorted = append(sorted, byName[name])
		for _, dep := range dependents[name] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(sorted) != len(pending) {
		return nil, fmt.Errorf("%w: not all migrations could be ordered", ErrCyclicDependency)
	}
	return sorted, nil
}
