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
	"time"

	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"go.uber.org/zap"
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

// Run executes migrations for the configured TypedMigrator. It should be called once
// at service bootup for each type that needs migration support.
func Run(ctx context.Context, cfgs ...Config) error {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return err
	}
	typeName := cfg.Migrator.TypeName()
	cfg.L.Debug("checking migration status", zap.String("type", typeName))

	meta, err := readMetadata(ctx, cfg, typeName)
	if err != nil {
		return err
	}

	if meta.Version >= cfg.Migrator.CurrentVersion() {
		cfg.L.Debug("already at current version",
			zap.String("type", typeName),
			zap.Int("version", meta.Version))
		return nil
	}

	cfg.L.Info("migrating",
		zap.String("type", typeName),
		zap.Int("from_version", meta.Version),
		zap.Int("to_version", cfg.Migrator.CurrentVersion()))

	if err := cfg.Migrator.MigrateAll(ctx, cfg.DB, meta.Version); err != nil {
		cfg.L.Error("migration failed",
			zap.String("type", typeName),
			zap.Error(err))
		return errors.Wrapf(err, "migration failed for %s", typeName)
	}

	meta.Version = cfg.Migrator.CurrentVersion()
	meta.MigratedAt = time.Now()

	cfg.L.Info("migration complete",
		zap.String("type", typeName),
		zap.Int("version", meta.Version))

	return writeMetadata(ctx, cfg, typeName, meta)
}

func readMetadata(ctx context.Context, cfg Config, typeName string) (*Metadata, error) {
	key := []byte(MetadataPrefix + typeName)
	data, closer, err := cfg.DB.KV().Get(ctx, key)
	if err != nil {
		if errors.Is(err, kv.NotFound) {
			return &Metadata{Version: 0}, nil
		}
		return nil, err
	}
	defer func() {
		err = errors.Combine(err, closer.Close())
	}()

	var meta Metadata
	if err = json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

func writeMetadata(ctx context.Context, cfg Config, typeName string, meta *Metadata) error {
	key := []byte(MetadataPrefix + typeName)
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return cfg.DB.KV().Set(ctx, key, data)
}
