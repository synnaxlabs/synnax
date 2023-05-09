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
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
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
	// MetaECD is the encoder/decoder used to encode Channel information into the
	// meta file.
	MetaECD binary.EncoderDecoder
}

var (
	_ config.Config[Config] = (*Config)(nil)
	// DefaultConfig is the default configuration for a DB.
	DefaultConfig = Config{
		MetaECD: &binary.JSONEncoderDecoder{Pretty: true},
	}
)

// Validate implements config.Config.
func (cfg Config) Validate() error {
	v := validate.New("cesium.unary")
	validate.NotNil(v, "FS", cfg.FS)
	validate.NotNil(v, "MetaECD", cfg.MetaECD)
	return v.Error()
}

// Override implements config.Config.
func (cfg Config) Override(other Config) Config {
	cfg.FS = override.Nil(cfg.FS, other.FS)
	cfg.MetaECD = override.Nil(cfg.MetaECD, other.MetaECD)
	if cfg.Channel.Key == 0 {
		cfg.Channel = other.Channel
	}
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	return cfg
}

func Open(configs ...Config) (*DB, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	cfg.Channel, err = readOrCreateMeta(cfg)
	if err != nil {
		return nil, err
	}
	rangerDB, err := ranger.Open(ranger.Config{FS: cfg.FS, Instrumentation: cfg.Instrumentation})

	db := &DB{Config: cfg, Ranger: rangerDB}
	if cfg.Channel.IsIndex {
		db._idx = &index.Ranger{DB: rangerDB, Instrumentation: cfg.Instrumentation}
	} else if cfg.Channel.Index == 0 {
		db._idx = index.Rate{Rate: cfg.Channel.Rate}
	}
	return db, err
}
