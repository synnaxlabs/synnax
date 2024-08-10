// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package virtual

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/version"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"sync/atomic"
)

type controlEntity struct {
	ck        core.ChannelKey
	alignment telem.AlignmentPair
}

func (e *controlEntity) ChannelKey() core.ChannelKey { return e.ck }

type DB struct {
	cfg              Config
	controller       *controller.Controller[*controlEntity]
	wrapError        func(error) error
	closed           *atomic.Bool
	leadingAlignment *atomic.Uint32
	openWriters      *atomic.Int32
}

var dbClosed = core.EntityClosed("virtual.db")

// Config is the configuration for opening a DB.
type Config struct {
	alamos.Instrumentation
	// Channel that the database will operate on. This only needs to be set when creating
	// a new database. If the database already exists, this field will be read from the
	// DB's meta file.
	Channel core.Channel
	// MetaCodec is used to encode and decode the channel metadata.
	// [REQUIRED]
	MetaCodec binary.Codec
	// FS is the filesystem that the DB will use to store meta-data about the channel.
	// [REQUIRED]
	FS xfs.FS
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Config.
func (cfg Config) Validate() error {
	v := validate.New("cesium.virtual")
	validate.NotNil(v, "FS", cfg.FS)
	validate.NotNil(v, "MetaCodec", cfg.MetaCodec)
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

func Open(configs ...Config) (db *DB, err error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	cfg.Channel, err = meta.ReadOrCreate(cfg.FS, cfg.Channel, cfg.MetaCodec)
	if err != nil {
		return nil, err
	}
	c, err := controller.New[*controlEntity](controller.Config{
		Concurrency:     control.Shared,
		Instrumentation: cfg.Instrumentation,
	})
	if err != nil {
		return nil, err
	}
	db = &DB{
		cfg:              cfg,
		controller:       c,
		wrapError:        core.NewErrorWrapper(cfg.Channel),
		closed:           &atomic.Bool{},
		leadingAlignment: &atomic.Uint32{},
		openWriters:      &atomic.Int32{},
	}
	db.leadingAlignment.Store(telem.ZeroLeadingAlignment)
	return db, nil
}

func (db *DB) CheckMigration(codec binary.Codec) error {
	if db.cfg.Channel.Version != version.Current {
		db.cfg.Channel.Version = version.Current
		return meta.Create(db.cfg.FS, codec, db.cfg.Channel)
	}
	return nil
}

func (db *DB) Channel() core.Channel {
	return db.cfg.Channel
}

func (db *DB) LeadingControlState() *controller.State {
	return db.controller.LeadingState()
}

func (db *DB) Close() error {
	if !db.closed.CompareAndSwap(false, true) {
		return nil
	}
	count := db.openWriters.Load()
	if count > 0 {
		err := db.wrapError(errors.Newf("cannot close channel because there are %d unclosed writers accessing it", count))
		db.closed.Store(false)
		return err
	}
	return nil
}

// RenameChannel renames the DB's channel to the given name, and persists the change to
// the underlying DB.
func (db *DB) RenameChannel(newName string) error {
	if db.closed.Load() {
		return dbClosed
	}
	if db.cfg.Channel.Name == newName {
		return nil
	}
	db.cfg.Channel.Name = newName
	return meta.Create(db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}

// SetChannelKeyInMeta sets the key of the channel for this DB, and persists that change
// to the DB's meta file in the underlying filesystem.
func (db *DB) SetChannelKeyInMeta(key core.ChannelKey) error {
	if db.closed.Load() {
		return dbClosed
	}
	if db.cfg.Channel.Key == key {
		return nil
	}
	db.cfg.Channel.Key = key
	return meta.Create(db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}
