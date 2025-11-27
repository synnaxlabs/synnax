// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
)

const (
	// MetadataPrefix is the key prefix for migration metadata.
	MetadataPrefix = "__jerky__/"
)

// Metadata stores migration state for a type.
type Metadata struct {
	Version    int       `json:"version"`
	MigratedAt time.Time `json:"migrated_at"`
}

// Coordinator manages migrations for multiple types using TypedMigrators.
type Coordinator struct {
	migrators map[string]TypedMigrator
	db        *gorp.DB
	kvDB      kv.DB
}

// NewCoordinator creates a new migration coordinator.
func NewCoordinator(db *gorp.DB, kvDB kv.DB, migrators ...TypedMigrator) *Coordinator {
	c := &Coordinator{
		migrators: make(map[string]TypedMigrator),
		db:        db,
		kvDB:      kvDB,
	}
	for _, m := range migrators {
		c.migrators[m.TypeName()] = m
	}
	return c
}

// Run executes all pending migrations.
func (c *Coordinator) Run(ctx context.Context) error {
	for typeName, migrator := range c.migrators {
		if err := c.migrateType(ctx, typeName, migrator); err != nil {
			return fmt.Errorf("migration failed for %s: %w", typeName, err)
		}
	}
	return nil
}

func (c *Coordinator) migrateType(ctx context.Context, typeName string, migrator TypedMigrator) error {
	// Read current metadata
	meta, err := c.readMetadata(ctx, typeName)
	if err != nil {
		return err
	}

	// Check if migration is needed
	if meta.Version >= migrator.CurrentVersion() {
		return nil
	}

	fmt.Printf("jerky: migrating %s from v%d to v%d\n", typeName, meta.Version, migrator.CurrentVersion())

	// Delegate migration to the TypedMigrator which uses gorp's iterator
	// and version-aware proto unmarshaling
	if err := migrator.MigrateAll(ctx, c.db, meta.Version); err != nil {
		return err
	}

	// Update metadata
	meta.Version = migrator.CurrentVersion()
	meta.MigratedAt = time.Now()

	fmt.Printf("jerky: completed migration for %s to v%d\n", typeName, meta.Version)

	return c.writeMetadata(ctx, typeName, meta)
}

func (c *Coordinator) readMetadata(ctx context.Context, typeName string) (*Metadata, error) {
	key := []byte(MetadataPrefix + typeName)
	data, closer, err := c.kvDB.Get(ctx, key)
	if err != nil {
		if err == kv.NotFound {
			return &Metadata{Version: 0}, nil
		}
		return nil, err
	}
	defer closer.Close()

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

func (c *Coordinator) writeMetadata(ctx context.Context, typeName string, meta *Metadata) error {
	key := []byte(MetadataPrefix + typeName)
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return c.kvDB.Set(ctx, key, data)
}
