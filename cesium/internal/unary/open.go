// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary

import (
	"context"
	"sync/atomic"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/alignment"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ErrVirtual is returned when the caller tried to open a unary database on a virtual
// channel.
var ErrVirtual = errors.New("cannot open a unary database on a virtual channel")

// Config is the configuration for opening a DB.
type Config struct {
	alamos.Instrumentation
	// Channel that the database will store data for. This only needs to be set when
	// creating a new database. If the database already exists, the Channel information
	// will be read from the DB's meta file.
	Channel channel.Channel
	// MetaCodec is used to encode and decode metadata about the channel.
	// [REQUIRED]
	MetaCodec binary.Codec
	// FS is the filesystem that the DB will use to store its data. DB will write to the
	// root of the filesystem, so this should probably be a subdirectory. DB should have
	// exclusive access, and it should be empty when the DB is first opened.
	// [REQUIRED]
	FS fs.FS
	// FileSize is the maximum size, in bytes, for a writer to be created on a file.
	// Note while that a file's size may still exceed this value, it is not likely to
	// exceed by much with frequent commits.
	// [OPTIONAL] Default: 1GB
	FileSize telem.Size
	// GCThreshold is the minimum tombstone proportion of the FileSize to trigger a GC.
	// Must be in (0, 1].
	// Note: Setting this value to 0 will have NO EFFECT as it is the default value.
	// instead, set it to a very small number greater than 0.
	// [OPTIONAL] Default: 0.2
	GCThreshold float32
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for a DB.
	DefaultConfig = Config{FileSize: 1 * telem.Gigabyte, GCThreshold: 0.2}
)

// Validate implements config.GateConfig.
func (cfg Config) Validate() error {
	v := validate.New("cesium.unary")
	validate.NotNil(v, "fs", cfg.FS)
	validate.NotNil(v, "meta_codec", cfg.MetaCodec)
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
	cfg.MetaCodec = override.Nil(cfg.MetaCodec, other.MetaCodec)
	return cfg
}

func Open(ctx context.Context, configs ...Config) (*DB, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	cfg.Channel, err = meta.Open(ctx, cfg.FS, cfg.Channel, cfg.MetaCodec)
	if err != nil {
		return nil, err
	}
	wrapError := channel.NewErrorWrapper(cfg.Channel)
	if cfg.Channel.Virtual {
		return nil, wrapError(ErrVirtual)
	}
	domainDB, err := domain.Open(domain.Config{
		FS:              cfg.FS,
		Instrumentation: cfg.Instrumentation,
		FileSize:        cfg.FileSize,
		GCThreshold:     cfg.GCThreshold,
	})
	if err != nil {
		return nil, err
	}
	c, err := control.New[*controlledWriter](control.Config{
		Concurrency:     cfg.Channel.Concurrency,
		Instrumentation: cfg.Instrumentation,
	})
	if err != nil {
		return nil, err
	}
	db := &DB{
		cfg:              cfg,
		domain:           domainDB,
		controller:       c,
		wrapError:        wrapError,
		closed:           &atomic.Bool{},
		leadingAlignment: &atomic.Uint32{},
	}
	db.leadingAlignment.Store(alignment.ZeroLeading)
	if cfg.Channel.IsIndex {
		db._idx = &index.Domain{DB: domainDB, Instrumentation: cfg.Instrumentation, Channel: cfg.Channel}
	}
	return db, nil
}
