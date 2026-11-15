// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package directory

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

const (
	// SetChannelName is the channel that carries each created or changed directory.
	SetChannelName = "sy_arc_directory_set"
	// DeleteChannelName is the channel that carries the key of each deleted directory.
	DeleteChannelName = "sy_arc_directory_delete"
)

// TableConfig is the configuration for opening the directory table.
type TableConfig struct {
	// DB is the database that stores directories.
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
	v := validate.New("arc.directory.table")
	v.NotNil("db", c.DB)
	return v.Error()
}

// Table stores Arc directories. Open it before the tables that reference directories.
type Table struct {
	table   *gorp.Table[Key, Directory]
	indexes indexes
}

// OpenTable opens the directory table.
func OpenTable(ctx context.Context, cfgs ...TableConfig) (*Table, error) {
	cfg, err := config.New(TableConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	t := &Table{indexes: newIndexes()}
	if t.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Directory]{
		DB:              cfg.DB,
		Name:            tableName,
		Indexes:         t.indexes.all(),
		References:      t.indexes.references(),
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	return t, nil
}

// NewRetrieve opens a query for directories.
func (t *Table) NewRetrieve() Retrieve {
	return Retrieve{gorp: t.table.NewRetrieve(), indexes: t.indexes}
}

// NewWriter opens a writer that runs on tx.
func (t *Table) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: tx, table: t.table}
}

// Changes returns the changes that c made to directories.
func (t *Table) Changes(
	ctx context.Context,
	c gorp.Commit,
) gorp.TxReader[Key, Directory] {
	return t.table.Changes(ctx, c)
}

// Observe returns an observable of the changes to directories.
func (t *Table) Observe() observe.Observable[gorp.TxReader[Key, Directory]] {
	return t.table.Observe()
}

// Resources returns the resource.Provider of directories.
func (t *Table) Resources() *resource.TableProvider[Key, Directory] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Directory]{
		Table:  t.table,
		Search: t.indexes.search,
		ID:     ResourceID,
		Key:    KeyFromResourceID,
		Name:   func(d *Directory) string { return d.Name },
	})
}

// Close closes the table.
func (t *Table) Close() error { return t.table.Close() }

// Writer creates, changes, and deletes directories. It checks no rule that involves
// the Arcs in a directory.
type Writer struct {
	tx    gorp.Tx
	table *gorp.Table[Key, Directory]
}

// Create creates d, giving d a key when it has none. It returns an error wrapping
// validate.ErrValidation when a directory with the key exists.
func (w Writer) Create(ctx context.Context, d *Directory) error {
	if d.Key == uuid.Nil() {
		d.Key = uuid.New()
	}
	if err := d.Validate(); err != nil {
		return err
	}
	return w.table.NewCreate().Exclusive().Entry(d).Exec(ctx, w.tx)
}

// Update replaces the directory with key with the result of change, which receives the
// stored directory. An invalid result is a validation error.
func (w Writer) Update(
	ctx context.Context,
	key Key,
	change func(Directory) (Directory, error),
) error {
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Directory](key)).
		ChangeErr(func(_ gorp.Context, d Directory) (Directory, error) {
			d, err := change(d)
			if err != nil {
				return Directory{}, err
			}
			if err = d.Validate(); err != nil {
				return Directory{}, err
			}
			return d, nil
		}).
		Exec(ctx, w.tx)
}

// Delete deletes the directories with keys. A directory that holds an Arc or a
// directory is a validation error.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Directory](keys...)).
		Guard(func(ctx gorp.Context, d Directory) error {
			deps, err := w.table.Dependents(ctx, ctx.Snapshot, d.Key)
			if err != nil {
				return err
			}
			if len(deps) > 0 {
				return errors.Wrapf(
					validate.ErrValidation, "directory %q is not empty", d.Name,
				)
			}
			return nil
		}).
		Exec(ctx, w.tx)
}
