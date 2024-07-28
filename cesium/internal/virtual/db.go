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
	"github.com/synnaxlabs/x/validate"
	"sync/atomic"
)

type controlEntity struct {
	ck              core.ChannelKey
	sampleAlignment uint32
}

func (e *controlEntity) ChannelKey() core.ChannelKey { return e.ck }

type DB struct {
	Config
	controller  *controller.Controller[*controlEntity]
	entityCount *core.EntityCount
	wrapError   func(error) error
	closed      *atomic.Bool
}

var dbClosed = core.EntityClosed("virtual.db")

type Config struct {
	alamos.Instrumentation
	FS      xfs.FS
	Channel core.Channel
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Config.
func (cfg Config) Validate() error {
	v := validate.New("cesium.virtual")
	validate.NotNil(v, "FS", cfg.FS)
	return v.Error()
}

// Override implements config.Config.
func (cfg Config) Override(other Config) Config {
	cfg.FS = override.Nil(cfg.FS, other.FS)
	if cfg.Channel.Key == 0 {
		cfg.Channel = other.Channel
	}
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	return cfg
}

func Open(configs ...Config) (db *DB, err error) {
	cfg, err := config.New(DefaultConfig, configs...)
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
	return &DB{
		Config:      cfg,
		controller:  c,
		wrapError:   core.NewErrorWrapper(cfg.Channel),
		entityCount: &core.EntityCount{},
		closed:      &atomic.Bool{},
	}, nil
}

func (db *DB) CheckMigration(codec binary.Codec) error {
	if db.Channel.Version != version.Current {
		db.Channel.Version = version.Current
		return meta.Create(db.FS, codec, db.Channel)
	}
	return nil
}

func (db *DB) LeadingControlState() *controller.State {
	return db.controller.LeadingState()
}

func (db *DB) Close() error {
	if db.closed.Load() {
		return nil
	}
	total, unlock := db.entityCount.LockAndCountOpen()
	defer unlock()
	if total > 0 {
		return db.wrapError(errors.Newf("cannot close channel because there are %d unclosed writers accessing it", total))
	}
	db.closed.Store(true)
	return nil
}
