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
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Config is the configuration for opening a DB.
type Config struct {
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
	// Logger is the witness of it all.
	Logger *zap.Logger
}

var (
	_ config.Config[Config] = (*Config)(nil)
	// DefaultConfig is the default configuration for a DB.
	DefaultConfig = Config{
		MetaECD: &binary.JSONIdentEncoderDecoder{},
		Logger:  zap.NewNop(),
	}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("cesium.unary")
	validate.NotNil(v, "FS", c.FS)
	validate.NotNil(v, "MetaECD", c.MetaECD)
	validate.NotNil(v, "Logger", c.Logger)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.FS = override.Nil(c.FS, other.FS)
	c.MetaECD = override.Nil(c.MetaECD, other.MetaECD)
	if c.Channel.Key == "" {
		c.Channel = other.Channel
	}
	c.Logger = override.Nil(c.Logger, other.Logger)
	return c
}

func Open(configs ...Config) (*DB, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	cfg.Channel, err = readOrCreateMeta(cfg)
	if err != nil {
		return nil, err
	}
	rangerDB, err := ranger.Open(ranger.Config{FS: cfg.FS, Logger: cfg.Logger})

	db := &DB{Config: cfg, Ranger: rangerDB}
	if cfg.Channel.IsIndex {
		db._idx = &index.Ranger{DB: rangerDB, Logger: cfg.Logger}
	} else if cfg.Channel.Index == "" {
		db._idx = index.Rate{Rate: cfg.Channel.Rate, Logger: cfg.Logger}
	}
	return db, err
}
