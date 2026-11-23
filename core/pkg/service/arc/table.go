// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"sync"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/arc/drift"
	driftv1 "github.com/synnaxlabs/synnax/pkg/service/arc/drift/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// rootName is the name of the directory that holds every Arc.
const rootName = "Arcs"

// DriftTable is the name of the table of drift records. A record holds the channels of
// its task's deployed program, so a reference it holds stands for the task.
const DriftTable = "arc_drift"

// TableConfig is the configuration for opening the Arc table.
type TableConfig struct {
	// DB is the database that stores Arcs.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[TableConfig] = TableConfig{}

// Override implements config.Config.
func (c TableConfig) Override(other TableConfig) TableConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements config.Config.
func (c TableConfig) Validate() error {
	v := validate.New("arc.table")
	v.NotNil("db", c.DB)
	return v.Error()
}

// Table stores Arcs, their directories, and their drift records. Open it before the
// tables that reference Arcs, so their migrations can check that an Arc exists.
type Table struct {
	table       *gorp.Table[Key, Arc]
	indexes     indexes
	directories *directory.Table
	root        directory.Key
	drifts      *gorp.Table[drift.Key, drift.Record]
	// params returns the channel params of each standard flow function, built at its
	// first call, since only a graph Arc reads them.
	params func() channelParams
}

// OpenTable opens the Arc table and migrates the stored Arcs. Close the table after the
// Service that uses it.
func OpenTable(ctx context.Context, cfgs ...TableConfig) (*Table, error) {
	cfg, err := config.New(TableConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	t := &Table{
		indexes: newIndexes(),
		params: sync.OnceValue(func() channelParams {
			return newChannelParams(standardSymbols())
		}),
	}
	if t.directories, err = directory.OpenTable(ctx, directory.TableConfig{
		DB:              cfg.DB,
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	if t.root, err = openRoot(ctx, cfg.DB, t.directories); err != nil {
		return nil, errors.Combine(err, t.directories.Close())
	}
	if t.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Arc]{
		DB:      cfg.DB,
		Name:    tableName,
		Indexes: t.indexes.all(),
		References: append(
			[]gorp.Reference[Key, Arc]{newDirectoryReference()},
			newChannelReferences(t.params)...,
		),
		Migrations:      versions.NewMigrations(t.root),
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, errors.Combine(err, t.directories.Close())
	}
	if t.drifts, err = gorp.OpenTable(ctx, gorp.TableConfig[drift.Key, drift.Record]{
		DB:   cfg.DB,
		Name: DriftTable,
		References: append(
			[]gorp.Reference[drift.Key, drift.Record]{newTaskReference()},
			newDeployedReferences()...,
		),
		Migrations:      []gorp.Migration{driftv1.Migration},
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, errors.Combine(
			err,
			errors.Combine(t.table.Close(), t.directories.Close()),
		)
	}
	return t, nil
}

// newTaskReference returns the reference from each drift record to its task, so
// deleting the task deletes the record.
func newTaskReference() *gorp.KeyReference[drift.Key, drift.Record, drift.Key] {
	return gorp.NewKeyReference(
		gorp.KeyReferenceConfig[drift.Key, drift.Record, drift.Key]{
			Name:     "task",
			Table:    string(resource.TypeTask),
			OnDelete: gorp.Cascade,
			Kind:     gorp.Place,
			Targets:  func(r *drift.Record) []drift.Key { return []drift.Key{r.Task} },
		},
	)
}

// openRoot returns the key of the directory that no directory holds, creating it when
// it does not exist.
func openRoot(
	ctx context.Context,
	db *gorp.DB,
	dirs *directory.Table,
) (directory.Key, error) {
	var roots []directory.Directory
	if err := db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return dirs.NewRetrieve().
			Where(directory.MatchNested(false)).
			Entries(&roots).
			Exec(ctx, snap)
	}); err != nil {
		return uuid.Nil(), err
	}
	if len(roots) > 0 {
		return roots[0].Key, nil
	}
	root := directory.Directory{Name: rootName}
	if err := db.WithTx(ctx, func(tx gorp.Tx) error {
		return dirs.NewWriter(tx).Create(ctx, &root)
	}); err != nil {
		return uuid.Nil(), err
	}
	return root.Key, nil
}

// newDirectoryReference returns the reference from each Arc to the directory that
// holds it.
func newDirectoryReference() *gorp.KeyReference[Key, Arc, directory.Key] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[Key, Arc, directory.Key]{
		Name:     "directory",
		Table:    string(resource.TypeArcDirectory),
		OnDelete: gorp.Restrict,
		Kind:     gorp.Place,
		Targets: func(a *Arc) []directory.Key {
			if d := a.Base().Directory; d != nil {
				return []directory.Key{*d}
			}
			return nil
		},
	})
}

// Close closes the table.
func (t *Table) Close() error {
	return errors.Combine(
		t.drifts.Close(),
		errors.Combine(t.table.Close(), t.directories.Close()),
	)
}
