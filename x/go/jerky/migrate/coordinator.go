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

	"github.com/synnaxlabs/x/kv"
)

const (
	// MetadataPrefix is the key prefix for migration metadata.
	MetadataPrefix = "__jerky__/"
	// BatchSize is the number of records to migrate in a single batch.
	BatchSize = 1000
	// ProgressInterval is how often to log progress.
	ProgressInterval = 10000
)

// Metadata stores migration state for a type.
type Metadata struct {
	Version             int       `json:"version"`
	MigratedAt          time.Time `json:"migrated_at"`
	MigrationInProgress bool      `json:"migration_in_progress,omitempty"`
	LastMigratedKey     string    `json:"last_migrated_key,omitempty"`
	MigratedCount       int       `json:"migrated_count,omitempty"`
}

// Coordinator manages migrations for multiple types.
type Coordinator struct {
	registries map[string]*Registry
	db         kv.DB
}

// NewCoordinator creates a new migration coordinator.
func NewCoordinator(db kv.DB, registries ...*Registry) *Coordinator {
	c := &Coordinator{
		registries: make(map[string]*Registry),
		db:         db,
	}
	for _, r := range registries {
		c.registries[r.TypeName] = r
	}
	return c
}

// Run executes all pending migrations.
func (c *Coordinator) Run(ctx context.Context) error {
	for typeName, registry := range c.registries {
		if err := c.migrateType(ctx, typeName, registry); err != nil {
			return fmt.Errorf("migration failed for %s: %w", typeName, err)
		}
	}
	return nil
}

func (c *Coordinator) migrateType(ctx context.Context, typeName string, registry *Registry) error {
	// Read current metadata
	meta, err := c.readMetadata(ctx, typeName)
	if err != nil {
		return err
	}

	// Check if migration is needed
	if meta.Version >= registry.CurrentVersion {
		return nil
	}

	// Handle in-progress migration (resume)
	if meta.MigrationInProgress {
		fmt.Printf("jerky: resuming migration for %s from key %s\n", typeName, meta.LastMigratedKey)
	}

	// Run migrations sequentially
	for version := meta.Version; version < registry.CurrentVersion; version++ {
		migration := registry.GetMigration(version, version+1)
		if migration == nil {
			return fmt.Errorf("no migration found for %s v%d -> v%d", typeName, version, version+1)
		}

		if err := c.executeMigration(ctx, typeName, migration, meta); err != nil {
			return err
		}

		meta.Version = version + 1
	}

	// Mark complete
	meta.MigrationInProgress = false
	meta.LastMigratedKey = ""
	meta.MigratedAt = time.Now()

	return c.writeMetadata(ctx, typeName, meta)
}

func (c *Coordinator) executeMigration(ctx context.Context, typeName string, migration *Migration, meta *Metadata) error {
	// Mark migration as in progress
	meta.MigrationInProgress = true
	if err := c.writeMetadata(ctx, typeName, meta); err != nil {
		return err
	}

	// Open iterator with optional resume point
	iterOpts := kv.IterPrefix([]byte(typeName))

	iter, err := c.db.OpenIterator(iterOpts)
	if err != nil {
		return err
	}
	defer iter.Close()

	// Skip to resume point if needed
	if meta.LastMigratedKey != "" {
		iter.SeekGE([]byte(meta.LastMigratedKey))
		if iter.Valid() {
			iter.Next() // Skip the already-migrated key
		}
	} else {
		iter.First()
	}

	batch := make([]struct{ key, value []byte }, 0, BatchSize)
	count := 0

	for iter.Valid() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())

		oldValue := make([]byte, len(iter.Value()))
		copy(oldValue, iter.Value())

		newValue, err := migration.Migrate(oldValue)
		if err != nil {
			return fmt.Errorf("migration failed for key %x: %w", key, err)
		}

		batch = append(batch, struct{ key, value []byte }{key, newValue})
		count++

		if len(batch) >= BatchSize {
			if err := c.flushBatch(ctx, batch); err != nil {
				return err
			}

			// Checkpoint
			meta.LastMigratedKey = string(key)
			meta.MigratedCount += len(batch)
			if err := c.writeMetadata(ctx, typeName, meta); err != nil {
				return err
			}

			batch = batch[:0]

			if count%ProgressInterval == 0 {
				fmt.Printf("jerky: migrated %d records for %s\n", count, typeName)
			}
		}

		iter.Next()
	}

	// Flush remaining
	if len(batch) > 0 {
		if err := c.flushBatch(ctx, batch); err != nil {
			return err
		}
		meta.MigratedCount += len(batch)
	}

	if err := iter.Error(); err != nil {
		return err
	}

	fmt.Printf("jerky: completed migration for %s (v%d -> v%d, %d records)\n",
		typeName, migration.FromVersion, migration.ToVersion, meta.MigratedCount)

	return nil
}

func (c *Coordinator) flushBatch(ctx context.Context, batch []struct{ key, value []byte }) error {
	for _, item := range batch {
		if err := c.db.Set(ctx, item.key, item.value); err != nil {
			return err
		}
	}
	return nil
}

func (c *Coordinator) readMetadata(ctx context.Context, typeName string) (*Metadata, error) {
	key := []byte(MetadataPrefix + typeName)
	data, closer, err := c.db.Get(ctx, key)
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
	return c.db.Set(ctx, key, data)
}
