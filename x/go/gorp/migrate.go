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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/graph"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"go.uber.org/zap"
)

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
	L       *alamos.Logger
}

// EntryCounter is optionally implemented by Migration types that track how many
// entries they processed. OpenTable checks for this after each migration to include
// entry counts in log output.
type EntryCounter interface {
	EntriesProcessed() int
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

// TransformFunc transforms an old entry of type I into a new entry of type O.
type TransformFunc[I, O any] func(ctx context.Context, old I) (O, error)

type typedMigration[IK Key, OK Key, I Entry[IK], O Entry[OK]] struct {
	name        string
	inputCodec  binary.Codec
	outputCodec binary.Codec
	transform   TransformFunc[I, O]
	entries     int
}

// NewTypedMigration creates a Migration that iterates over all entries with the
// configured prefix, decodes each as type I using inputCodec, transforms it to
// type O via the transform function, and encodes the result using outputCodec.
// If inputCodec is nil, MigrationConfig.DBCodec is used for decoding.
// If outputCodec is nil, MigrationConfig.DBCodec is used for encoding.
func NewTypedMigration[IK Key, OK Key, I Entry[IK], O Entry[OK]](
	name string,
	inputCodec binary.Codec,
	outputCodec binary.Codec,
	transform TransformFunc[I, O],
) Migration {
	return &typedMigration[IK, OK, I, O]{
		name:        name,
		inputCodec:  inputCodec,
		outputCodec: outputCodec,
		transform:   transform,
	}
}

func (m *typedMigration[IK, OK, I, O]) Name() string { return m.name }

func (m *typedMigration[IK, OK, I, O]) EntriesProcessed() int { return m.entries }

func (m *typedMigration[IK, OK, I, O]) Run(
	ctx context.Context,
	kvTx kv.Tx,
	cfg MigrationConfig,
) (err error) {
	m.entries = 0
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
		m.entries++
		if shouldLogProgress(m.entries) {
			cfg.L.Debug(
				"migration progress",
				zap.String("migration", m.name),
				zap.Int("entries", m.entries),
			)
		}
		var old I
		if err = inCodec.Decode(ctx, iter.Value(), &old); err != nil {
			return errors.Wrapf(err, "entry %q (decode)", iter.Key())
		}
		newEntry, err := m.transform(ctx, old)
		if err != nil {
			return errors.Wrapf(err, "entry %v (transform)", old.GorpKey())
		}
		var data []byte
		if data, err = outCodec.Encode(ctx, newEntry); err != nil {
			return errors.Wrapf(err, "entry %v (encode)", newEntry.GorpKey())
		}
		if err = kvTx.Set(ctx, iter.Key(), data); err != nil {
			return err
		}
	}
	return err
}

type codecTransitionMigration[K Key, E Entry[K]] struct {
	name    string
	codec   binary.Codec
	entries int
}

// NewCodecTransition creates a Migration that re-encodes all entries from the DB's
// default codec (e.g. msgpack) to the provided target codec (e.g. protobuf).
func NewCodecTransition[K Key, E Entry[K]](name string, codec binary.Codec) Migration {
	return &codecTransitionMigration[K, E]{name: name, codec: codec}
}

func (m *codecTransitionMigration[K, E]) Name() string { return m.name }

func (m *codecTransitionMigration[K, E]) EntriesProcessed() int { return m.entries }

func (m *codecTransitionMigration[K, E]) Run(
	ctx context.Context,
	kvTx kv.Tx,
	cfg MigrationConfig,
) (err error) {
	m.entries = 0
	iter, err := kvTx.OpenIterator(kv.IterPrefix(cfg.Prefix))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		m.entries++
		if shouldLogProgress(m.entries) {
			cfg.L.Debug(
				"migration progress",
				zap.String("migration", m.name),
				zap.Int("entries", m.entries),
			)
		}
		var entry E
		if err = cfg.DBCodec.Decode(ctx, iter.Value(), &entry); err != nil {
			return errors.Wrapf(err, "entry %q (decode)", iter.Key())
		}
		var data []byte
		if data, err = m.codec.Encode(ctx, entry); err != nil {
			return errors.Wrapf(err, "entry %v (encode)", entry.GorpKey())
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

type depKey[T any] struct{}

// ErrMissingMigrationDep is returned when MigrationDep is called for a type
// that was not injected into the context via WithMigrationDep.
var ErrMissingMigrationDep = errors.New("missing migration dependency")

// WithMigrationDep injects a dependency into the context for use during migrations.
// The dependency is keyed by its type, so each concrete type can be injected once.
// Use this before calling OpenTable to make services or other resources available
// to migration transform functions.
func WithMigrationDep[T any](ctx context.Context, dep T) context.Context {
	return context.WithValue(ctx, depKey[T]{}, dep)
}

// MigrationDep retrieves a dependency previously injected via WithMigrationDep.
// Panics if the dependency was not injected, since this indicates a programming
// error (the service forgot to inject the dependency before calling OpenTable).
func MigrationDep[T any](ctx context.Context) T {
	v, ok := ctx.Value(depKey[T]{}).(T)
	if !ok {
		panic(fmt.Sprintf(
			"%s: %T not found in context",
			ErrMissingMigrationDep,
			*new(T),
		))
	}
	return v
}

// MigrationDepOpt retrieves a dependency previously injected via WithMigrationDep.
// Returns the dependency and true if found, or the zero value and false if not.
func MigrationDepOpt[T any](ctx context.Context) (T, bool) {
	v, ok := ctx.Value(depKey[T]{}).(T)
	return v, ok
}

// topoSort filters out already-applied migrations, then produces a valid
// execution order. Dependencies that are already applied are considered
// satisfied and do not need to appear in the pending set.
func topoSort(migrations []Migration, applied map[string]bool) ([]Migration, error) {
	byName := make(map[string]Migration, len(migrations))
	for _, m := range migrations {
		if _, dup := byName[m.Name()]; dup {
			return nil, fmt.Errorf("duplicate migration name %q", m.Name())
		}
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

	adj := make(map[string][]string, len(pending))
	for _, m := range pending {
		name := m.Name()
		adj[name] = nil
		if dd, ok := m.(DependencyDeclarer); ok {
			for _, dep := range dd.Dependencies() {
				if applied[dep] {
					continue
				}
				adj[name] = append(adj[name], dep)
			}
		}
	}

	order, err := graph.TopoSort(adj)
	if err != nil {
		return nil, err
	}

	sorted := make([]Migration, len(order))
	for i, name := range order {
		sorted[i] = byName[name]
	}
	return sorted, nil
}

// Deprecated: Use the Migration interface with OpenTable instead.
var ErrMigrationCountExceeded = errors.New(
	"migration count is greater than the maximum of 255",
)

// MigrationSpec defines a single migration that should be run with a transaction.
//
// Deprecated: Use the Migration interface with OpenTable instead.
type MigrationSpec struct {
	Migrate func(context.Context, Tx) error
	Name    string
}

// Migrator executes a series of migrations in order, tracking progress with
// incrementing versions.
//
// Deprecated: Use the Migration interface with OpenTable instead.
type Migrator struct {
	Key        string
	Migrations []MigrationSpec
	Force      bool
}

// Run executes all migrations that haven't been completed yet.
//
// Deprecated: Use the Migration interface with OpenTable instead.
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
