// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package distribution

import (
	"context"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/control"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// LayerConfig is the configuration for opening the distribution layer.
type LayerConfig struct {
	// Storage is the storage layer that the distribution layer will use for persisting
	// data across its various services.
	//
	// [REQUIRED]
	Storage *storage.Layer
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// AdvertiseAddress is the address at which the host node is reached.
	//
	// [REQUIRED]
	AdvertiseAddress address.Address
}

var _ config.Config[LayerConfig] = LayerConfig{}

// Override implements config.Config.
func (c LayerConfig) Override(other LayerConfig) LayerConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Storage = override.Nil(c.Storage, other.Storage)
	c.AdvertiseAddress = override.String(c.AdvertiseAddress, other.AdvertiseAddress)
	return c
}

// Validate implements config.Config.
func (c LayerConfig) Validate() error {
	v := validate.New("distribution")
	v.NotNil("storage", c.Storage)
	if c.Storage != nil {
		v.NotNil("storage.gorp", c.Storage.Gorp)
	}
	v.NotEmptyString("advertise_address", c.AdvertiseAddress)
	return v.Error()
}

// Layer contains all relevant services within the Synnax distribution layer. The
// distribution layer wraps the storage layer to provide a monolithic data space for
// working with core data structures across Synnax.
//
// Streamers need Run to run. The Layer must be closed after Run returns. It is not safe
// to modify any of the public fields in this struct, or to access these fields after
// Close has been called.
type Layer struct {
	// DB stores the metadata of the Core. It is the Gorp DB of the storage layer.
	DB *gorp.DB
	// Identity names the Core: its cluster key and host key.
	Identity cluster.Identity
	// Host is the host node of the Core.
	Host node.Node
	// Channel is the distribution-layer channel allocator: it assigns local keys and
	// creates, renames, and deletes storage channels.
	Channel *channel.Service
	// Framer is for reading, writing, and streaming frames of telemetry.
	Framer *framer.Service
	// Control is for reading and observing the control state of channels.
	Control *control.Service
	closer  io.MultiCloser
	// lifecycle orders Run against Close.
	lifecycle struct {
		sync.Mutex
		running bool
		ran     bool
		closed  bool
	}
}

// OpenLayer opens the distribution Layer using the provided configuration(s). Later
// configurations override the fields set in previous configurations. If the
// configuration is invalid, or any services fail to open, OpenLayer returns a nil layer
// and an error.
//
// OpenLayer starts no goroutines, but the control service subscribes to storage. If
// the returned error is nil, the Layer must be closed by calling Close after use. None
// of the services in the Layer should be used after Close is called. It is the
// caller's responsibility to ensure that the Layer is not accessed after it is closed.
func OpenLayer(ctx context.Context, cfgs ...LayerConfig) (l *Layer, err error) {
	cfg, err := config.New(LayerConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	l = &Layer{}
	cleanup, ok := service.NewOpener(ctx, &l.closer)
	defer func() { err = cleanup(err) }()

	if l.Identity, err = cluster.OpenIdentity(
		ctx,
		cfg.Storage.KV,
		cfg.AdvertiseAddress,
	); !ok(err, nil) {
		return nil, err
	}
	l.Host = node.Node{Key: l.Identity.HostKey, Address: cfg.AdvertiseAddress}
	l.DB = cfg.Storage.Gorp

	if l.Channel, err = channel.NewService(ctx, channel.ServiceConfig{
		Instrumentation: cfg.Child("channel"),
		HostKey:         l.Identity.HostKey,
		KV:              cfg.Storage.KV,
		TS:              cfg.Storage.TS,
	}); !ok(err, nil) {
		return nil, err
	}

	if l.Framer, err = framer.OpenService(framer.ServiceConfig{
		Instrumentation: cfg.Child("framer"),
		TS:              cfg.Storage.TS,
	}); !ok(err, l.Framer) {
		return nil, err
	}

	if l.Control, err = control.OpenService(ctx, control.ServiceConfig{
		Instrumentation: cfg.Child("control"),
		TS:              cfg.Storage.TS,
	}); !ok(err, l.Control) {
		return nil, err
	}

	return l, nil
}

// Run runs the goroutines of the Layer until ctx ends, then stops every streamer and
// returns once the goroutines exit. Streamers wait for Run to start and fail after it
// returns. A second Run, or a Run after Close, returns an error.
func (l *Layer) Run(ctx context.Context) error {
	l.lifecycle.Lock()
	if l.lifecycle.closed {
		l.lifecycle.Unlock()
		return errors.New("run called after close")
	}
	if l.lifecycle.ran {
		l.lifecycle.Unlock()
		return errors.New("run called twice")
	}
	l.lifecycle.ran, l.lifecycle.running = true, true
	l.lifecycle.Unlock()
	defer func() {
		l.lifecycle.Lock()
		l.lifecycle.running = false
		l.lifecycle.Unlock()
	}()
	return l.Framer.Run(ctx)
}

// Close closes the Layer. Call it only after Run returns: while Run runs, Close returns
// an error and closes nothing. It is the caller's responsibility to ensure that all
// routines interacting with the Layer have finished before calling Close.
func (l *Layer) Close() error {
	l.lifecycle.Lock()
	defer l.lifecycle.Unlock()
	if l.lifecycle.running {
		return errors.New("close called while Run runs")
	}
	if l.lifecycle.closed {
		return nil
	}
	l.lifecycle.closed = true
	return l.closer.Close()
}
