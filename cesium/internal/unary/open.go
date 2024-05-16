// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/config"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening a DB.
type Config struct {
	alamos.Instrumentation
	// FS is where the database stores its files. This FS is assumed to be a directory
	// where DB has exclusive read and write access.
	FS xfs.FS
	// Channel is the Channel for the database. This only needs to be set when
	// creating a new database. If the database already exists, the Channel information
	// will be read from the databases meta file.
	Channel core.Channel
	// FileSize is the maximum file before which no more writes can be made to a file.
	// [OPTIONAL] Default: 1GB
	FileSize telem.Size
	// GCThreshold is the minimum tombstone proportion of the Filesize to trigger a GC.
	// Must be in (0, 1].
	// Note: Setting this value to 0 will have NO EFFECT as it is the default value.
	// instead, set it to a very small number greater than 0.
	// [OPTIONAL] Default: 0.5
	GCThreshold float32
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for a DB.
	DefaultConfig = Config{FileSize: 1 * telem.Gigabyte, GCThreshold: 0.5}
)

// Validate implements config.GateConfig.
func (cfg Config) Validate() error {
	v := validate.New("cesium.unary")
	validate.NotNil(v, "FS", cfg.FS)
	return v.Error()
}

// Override implements config.GateConfig.
func (cfg Config) Override(other Config) Config {
	cfg.FS = override.Nil(cfg.FS, other.FS)
	if cfg.Channel.Key == 0 {
		cfg.Channel = other.Channel
	}
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	cfg.FileSize = override.Numeric(cfg.FileSize, other.FileSize)
	cfg.GCThreshold = override.Numeric(cfg.GCThreshold, other.GCThreshold)
	return cfg
}

func Open(configs ...Config) (*DB, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	domainDB, err := domain.Open(domain.Config{
		FS:              cfg.FS,
		Instrumentation: cfg.Instrumentation,
		FileSize:        cfg.FileSize,
		GCThreshold:     cfg.GCThreshold,
	})
	db := &DB{
		Config:     cfg,
		Domain:     domainDB,
		Controller: controller.New[controlledWriter](cfg.Channel.Concurrency),
		mu:         &openEntityCount{},
	}
	if cfg.Channel.IsIndex {
		db._idx = &index.Domain{DB: domainDB, Instrumentation: cfg.Instrumentation}
	} else if cfg.Channel.Index == 0 {
		db._idx = index.Rate{Rate: cfg.Channel.Rate}
	}
	return db, err
}
