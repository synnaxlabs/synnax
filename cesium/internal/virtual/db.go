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
	"fmt"
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
	"sync"
	"sync/atomic"
)

type controlEntity struct {
	ck        core.ChannelKey
	alignment telem.AlignmentPair
}

func (e *controlEntity) ChannelKey() core.ChannelKey { return e.ck }

type entityCount struct {
	sync.RWMutex
	openWriters int
}

func (s *entityCount) add(delta int) {
	s.Lock()
	s.openWriters += delta
	s.Unlock()
}

type DB struct {
	Config
	controller  *controller.Controller[*controlEntity]
	entityCount *entityCount
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
		entityCount: &entityCount{},
		closed:      &atomic.Bool{},
	}, nil
}

func (db *DB) CheckMigration(ecd binary.Codec) error {
	if db.Channel.Version != version.Current {
		db.Channel.Version = version.Current
		return meta.Create(db.FS, ecd, db.Channel)
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
	db.entityCount.RLock()
	defer db.entityCount.RUnlock()
	if db.entityCount.openWriters > 0 {
		return db.wrapError(errors.Newf(fmt.Sprintf("cannot close channel because there are %d unclosed writers accessing it", db.entityCount.openWriters)))
	}
	db.closed.Store(true)
	return nil
}
