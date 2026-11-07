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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// TableConfig is the configuration for opening the Arc table.
type TableConfig struct {
	// DB is the database that stores Arcs.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology holds the directory of each Arc and directory.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[TableConfig] = TableConfig{}

// Override implements config.Config.
func (c TableConfig) Override(other TableConfig) TableConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements config.Config.
func (c TableConfig) Validate() error {
	v := validate.New("arc.table")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	return v.Error()
}

// Table stores Arcs and their directories. Open it before the tables that reference
// Arcs, so their migrations can check that an Arc exists.
type Table struct {
	table       *gorp.Table[Key, Arc]
	indexes     indexes
	owner       *ontology.Owner
	directories directories
	root        directory.Key
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
		owner: cfg.Ontology.Own(ontology.Ownership{
			Children: []resource.Type{
				resource.TypeArc,
				resource.TypeArcDirectory,
			},
			Parents: []resource.Type{resource.TypeArcDirectory},
		}),
	}
	if t.directories, err = openDirectories(ctx, cfg); err != nil {
		return nil, err
	}
	if t.root, err = openRoot(ctx, cfg, t.directories, t.owner); err != nil {
		return nil, errors.Combine(err, t.directories.table.Close())
	}
	if t.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Arc]{
		DB:      cfg.DB,
		Name:    tableName,
		Indexes: t.indexes.all(),
		Migrations: versions.NewMigrations(versions.MigrationsConfig{
			Ontology: cfg.Ontology,
			Owner:    t.owner,
			Root:     directory.ResourceID(t.root),
		}),
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, errors.Combine(err, t.directories.table.Close())
	}
	return t, nil
}

// Close closes the table.
func (t *Table) Close() error {
	return errors.Combine(t.table.Close(), t.directories.table.Close())
}
