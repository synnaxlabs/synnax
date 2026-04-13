// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package variable

import (
	"context"
	"sync/atomic"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/alignment"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var ErrNotVariable = errors.New("channel does not have a variable-density data type")

type Config struct {
	alamos.Instrumentation
	MetaCodec   encoding.Codec
	FS          fs.FS
	Channel     channel.Channel
	FileSize    telem.Size
	GCThreshold float32
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{FileSize: 1 * telem.Gigabyte, GCThreshold: 0.2}
)

func (cfg Config) Validate() error {
	v := validate.New("cesium.variable")
	validate.NotNil(v, "fs", cfg.FS)
	validate.NotNil(v, "meta_codec", cfg.MetaCodec)
	return v.Error()
}

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
		return nil, wrapError(errors.New("cannot open a variable database on a virtual channel"))
	}
	if !cfg.Channel.DataType.IsVariable() {
		return nil, wrapError(ErrNotVariable)
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
		offsets:          newOffsetCache(),
	}
	db.leadingAlignment.Store(alignment.ZeroLeading)
	return db, nil
}
