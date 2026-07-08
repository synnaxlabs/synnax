// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package virtual

import (
	"sync/atomic"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/alignment"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/config"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type controlResource struct {
	ck channel.Key
	// alignment tracks the current write position as a packed domain index (upper 32
	// bits) and sample index (lower 32 bits). This field is accessed atomically because
	// Gate.Authorize and Gate.PeekResource return a shared pointer to this struct, and
	// the region's RWMutex is released before the caller accesses the field. This means
	// one goroutine may write alignment through Authorize while another reads it
	// through PeekResource concurrently.
	alignment atomic.Uint64
}

var _ control.Resource = &controlResource{}

func (r *controlResource) ChannelKey() channel.Key { return r.ck }

func (r *controlResource) loadAlignment() telem.Alignment {
	return telem.Alignment(r.alignment.Load())
}

func (r *controlResource) storeAlignment(a telem.Alignment) {
	r.alignment.Store(uint64(a))
}

// DB is a purely in-memory engine for a single virtual channel: it registers the
// channel and coordinates control handoff and write alignment between writers opened
// on it. Nothing about the channel is ever written to the file system.
type DB struct {
	controller       *control.Controller[*controlResource]
	wrapError        func(error) error
	closed           *atomic.Bool
	leadingAlignment *atomic.Uint32
	openWriters      *atomic.Int32
	cfg              Config
}

// ErrDBClosed is returned when an operation is attempted on a closed DB.
var ErrDBClosed = resource.NewClosedError("virtual.db")

// Config is the configuration for opening a DB. The DB is a purely in-memory engine:
// nothing about the channel is ever written to the file system.
type Config struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// Channel that the database will operate on. All fields must be fully resolved by
	// the caller before opening the DB.
	//
	// [REQUIRED]
	Channel channel.Channel
}

var _ config.Config[Config] = Config{}

// Validate implements config.Config.
func (cfg Config) Validate() error {
	v := validate.New("cesium.virtual")
	validate.Positive(v, "channel.key", cfg.Channel.Key)
	return v.Error()
}

// Override implements config.Config.
func (cfg Config) Override(other Config) Config {
	if cfg.Channel.Key == 0 {
		cfg.Channel = other.Channel
	}
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	return cfg
}

// Open opens a DB on the virtual channel in the given configuration. It returns a
// validation error if the configuration is invalid or the channel is not virtual.
func Open(configs ...Config) (*DB, error) {
	cfg, err := config.New(Config{}, configs...)
	if err != nil {
		return nil, err
	}
	wrapError := channel.NewErrorWrapper(cfg.Channel)
	if !cfg.Channel.Virtual {
		return nil, wrapError(errors.New("channel is not virtual"))
	}
	c, err := control.New[*controlResource](control.Config{
		Concurrency:     xcontrol.ConcurrencyShared,
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

// Channel returns the channel the DB operates on.
func (db *DB) Channel() channel.Channel {
	return db.cfg.Channel
}

// AllocateLeadingAlignment reserves and returns a fresh leading alignment domain for
// the channel. Writers on an index group allocate one domain per group from the group's
// index channel so that alignments correlate across the group's members.
func (db *DB) AllocateLeadingAlignment() telem.Alignment {
	return telem.NewAlignment(db.leadingAlignment.Add(1), 0)
}

// LeadingControlState returns the control state of the subject currently in control of
// the channel, or nil if no writers are open on the DB.
func (db *DB) LeadingControlState() *control.State {
	return db.controller.LeadingState()
}

// Close closes the DB. It returns an error wrapping resource.ErrOpen if any writers
// are still open on the DB, in which case the DB remains usable. Closing an
// already-closed DB is a no-op.
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

// RenameChannel renames the DB's channel to the given name. It returns ErrDBClosed if
// the DB is closed.
func (db *DB) RenameChannel(newName string) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	db.cfg.Channel.Name = newName
	return nil
}

// SetChannelKey sets the key of the channel for this DB. It returns ErrDBClosed if the
// DB is closed.
func (db *DB) SetChannelKey(key channel.Key) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	db.cfg.Channel.Key = key
	return nil
}
