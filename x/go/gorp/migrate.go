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
	"github.com/synnaxlabs/x/graph"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/types"
	"go.uber.org/zap"
)

// Migration is a versioned schema migration that transforms entries stored in gorp.
type Migration interface {
	// Name returns a human-readable identifier for this migration.
	Name() string
	// Run executes the migration within the provided kv.Tx.
	Run(ctx context.Context, tx Tx, ins alamos.Instrumentation) error
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

type entryMigration[IK Key, OK Key, I Entry[IK], O Entry[OK]] struct {
	name      string
	transform TransformFunc[I, O]
	entries   int
}

// NewEntryMigration creates a Migration that iterates over all entries with the
// configured prefix, decodes each as type I, transforms it to type O via the
// transform function, and encodes the result. Both decoding and encoding use
// the DB's codec from MigrationContext.
func NewEntryMigration[IK Key, OK Key, I Entry[IK], O Entry[OK]](
	name string,
	transform TransformFunc[I, O],
) Migration {
	return &entryMigration[IK, OK, I, O]{
		name:      name,
		transform: transform,
	}
}

func (m *entryMigration[IK, OK, I, O]) Name() string { return m.name }

func (m *entryMigration[IK, OK, I, O]) EntriesProcessed() int { return m.entries }

func (m *entryMigration[IK, OK, I, O]) Run(ctx context.Context, tx Tx, ins alamos.Instrumentation) (err error) {
	var (
		reader = WrapReader[IK, I](tx)
		writer = WrapWriter[OK, O](tx)
	)
	m.entries = 0
	iter, err := reader.OpenIterator(IterOptions{})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		m.entries++
		if shouldLogProgress(m.entries) {
			ins.L.Debug(
				"migration progress",
				zap.String("migration", m.name),
				zap.Int("entries", m.entries),
			)
		}
		old := iter.Value(ctx)
		if old == nil {
			if err := iter.Error(); err != nil {
				return err
			}
			continue
		}
		newEntry, err := m.transform(ctx, *old)
		if err != nil {
			return errors.Wrapf(err, "entry %v (transform)", (*old).GorpKey())
		}
		if err = writer.Set(ctx, newEntry); err != nil {
			return err
		}
	}
	return err
}

type migration struct {
	name string
	fn   func(ctx context.Context, tx Tx, ins alamos.Instrumentation) error
}

// NewMigration creates a Migration that receives a fully wrapped gorp.Tx,
// allowing arbitrary read/write operations on the store.
func NewMigration(
	name string,
	fn func(ctx context.Context, tx Tx, ins alamos.Instrumentation) error,
) Migration {
	return &migration{name: name, fn: fn}
}

func (m *migration) Name() string { return m.name }

func (m *migration) Run(ctx context.Context, tx Tx, ins alamos.Instrumentation) error {
	return m.fn(ctx, tx, ins)
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

// WithMigrationDep injects a dependency into the context for use during migrations.
// The dependency is keyed by its type, so each concrete type can be injected once.
// Use this before calling OpenTable to make services or other resources available
// to migration transform functions.
func WithMigrationDep[T any](ctx context.Context, dep T) context.Context {
	return context.WithValue(ctx, depKey[T]{}, dep)
}

// MigrationDep retrieves a dependency previously injected via WithMigrationDep.
// Returns ErrMissingMigrationDep if the dependency was not injected.
func MigrationDep[T any](ctx context.Context) (T, error) {
	v, ok := ctx.Value(depKey[T]{}).(T)
	if !ok {
		return v, errors.Wrapf(
			query.ErrNotFound,
			"%s not found in context",
			types.Name[T](),
		)
	}
	return v, nil
}

// topoSort filters out already-applied migrations, then produces a valid
// execution order. Dependencies that are already applied are considered
// satisfied and do not need to appear in the pending set.
func topoSort(migrations []Migration, applied set.Set[string]) ([]Migration, error) {
	byName := make(map[string]Migration, len(migrations))
	for _, m := range migrations {
		if _, dup := byName[m.Name()]; dup {
			return nil, errors.Newf("duplicate migration name %q", m.Name())
		}
		byName[m.Name()] = m
	}

	var pending []Migration
	for _, m := range migrations {
		if !applied.Contains(m.Name()) {
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
				if applied.Contains(dep) {
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
