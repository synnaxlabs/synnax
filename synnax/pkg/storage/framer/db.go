// Copyright 2023 Synnax Labs, Inc.
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/storage/control"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

const AutoSpan = cesium.AutoSpan

type (
	Channel        = cesium.Channel
	ChannelKey     = cesium.ChannelKey
	Frame          = cesium.Frame
	Iterator       = cesium.Iterator
	IteratorConfig = cesium.IteratorConfig
)

type Config struct {
	alamos.Instrumentation
	// Dirname is the directory in which the DB will store its files. DB should have
	// exclusive access to this directory, and sharing with another process/component
	// may result in data corruption. Defaults to an empty string.
	Dirname string
	// FS is the file system interface that the DB will use to read and write data.
	// [REQUIRED]
	FS xfs.FS
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for a DB.
	DefaultConfig = Config{FS: xfs.Default}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("ts")
	validate.NotNil(v, "FS", c.FS)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Dirname = override.String(c.Dirname, other.Dirname)
	c.FS = override.Nil(c.FS, other.FS)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

type DB struct {
	internal *cesium.DB
	control  *control.Service[ChannelKey]
	relay    *relay
}

func Open(configs ...Config) (*DB, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	db := &DB{relay: openRelay(cfg), control: control.NewService[ChannelKey]()}
	db.internal, err = cesium.Open(
		cfg.Dirname,
		cesium.WithFS(cfg.FS),
		cesium.WithInstrumentation(cfg.Instrumentation),
	)
	return db, err
}

func (db *DB) CreateChannel(ctx context.Context, ch ...Channel) error {
	return db.internal.CreateChannel(ctx, ch...)
}

func (db *DB) OpenIterator(cfg IteratorConfig) (*Iterator, error) {
	return db.internal.OpenIterator(cfg)
}

func (db *DB) Close() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	c.Exec(db.internal.Close)
	c.Exec(db.relay.close)
	return c.Error()
}
