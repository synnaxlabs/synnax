// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package virtual

import (
	"context"
	"sync/atomic"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/alignment"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type controlResource struct {
	ck        channel.Key
	alignment telem.Alignment
}

func (r *controlResource) ChannelKey() channel.Key { return r.ck }

type DB struct {
	cfg              Config
	controller       *control.Controller[*controlResource]
	wrapError        func(error) error
	closed           *atomic.Bool
	leadingAlignment *atomic.Uint32
	openWriters      *atomic.Int32
}

var (
	// ErrNotVirtual is returned when the caller opens a DB on a non-virtual channel.
	ErrNotVirtual = errors.New("channel is not virtual")
	// ErrDBClosed is returned when an operation is attempted on a closed DB.
	ErrDBClosed = resource.NewErrClosed("virtual.db")
)

// Config is the configuration for opening a DB.
type Config struct {
	alamos.Instrumentation
	// Channel that the database will operate on. This only needs to be set when creating
	// a new database. If the database already exists, this field will be read from the
	// DB's meta file.
	Channel channel.Channel
	// MetaCodec is used to encode and decode the channel metadata.
	// [REQUIRED]
	MetaCodec binary.Codec
	// FS is the filesystem that the DB will use to store metadata about the channel.
	// [REQUIRED]
	FS fs.FS
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Config.
func (cfg Config) Validate() error {
	v := validate.New("cesium.virtual")
	validate.NotNil(v, "fs", cfg.FS)
	validate.NotNil(v, "meta_codec", cfg.MetaCodec)
	return v.Error()
}

// Override implements config.Config.
func (cfg Config) Override(other Config) Config {
	cfg.FS = override.Nil(cfg.FS, other.FS)
	if cfg.Channel.Key == 0 {
		cfg.Channel = other.Channel
	}
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
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
	wrapError := channel.NewErrWrapper(cfg.Channel)
	if !cfg.Channel.Virtual {
		return nil, wrapError(ErrNotVirtual)
	}
	c, err := control.New[*controlResource](control.Config{
		Concurrency:     xcontrol.Shared,
		Instrumentation: cfg.Instrumentation,
	})
	if err != nil {
		return nil, err
	}
	db := &DB{
		cfg:              cfg,
		controller:       c,
		wrapError:        wrapError,
		closed:           &atomic.Bool{},
		leadingAlignment: &atomic.Uint32{},
		openWriters:      &atomic.Int32{},
	}
	db.leadingAlignment.Store(alignment.ZeroLeading)
	return db, nil
}

func (db *DB) Channel() channel.Channel {
	return db.cfg.Channel
}

func (db *DB) LeadingControlState() *control.State {
	return db.controller.LeadingState()
}

func (db *DB) Close() error {
	if !db.closed.CompareAndSwap(false, true) {
		return nil
	}
	count := db.openWriters.Load()
	if count > 0 {
		err := db.wrapError(errors.Wrapf(resource.ErrOpen, "cannot close channel because there are %d unclosed writers accessing it", count))
		db.closed.Store(false)
		return err
	}
	return nil
}

// RenameChannel renames the DB's channel to the given name, and persists the change to
// the underlying DB.
func (db *DB) RenameChannel(ctx context.Context, newName string) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	if db.cfg.Channel.Name == newName {
		return nil
	}
	db.cfg.Channel.Name = newName
	return meta.Create(ctx, db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}

// SetChannelKeyInMeta sets the key of the channel for this DB, and persists that change
// to the DB's meta file in the underlying filesystem.
func (db *DB) SetChannelKeyInMeta(ctx context.Context, key channel.Key) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	if db.cfg.Channel.Key == key {
		return nil
	}
	db.cfg.Channel.Key = key
	return meta.Create(ctx, db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}
