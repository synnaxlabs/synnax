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
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// LayerConfig is the configuration for opening the distribution layer.
type LayerConfig struct {
	// GorpCodec sets the codec used to encode/decode data structures within the
	// metadata DB.
	//
	// [OPTIONAL] - Defaults to encoding.NewDecodeFallbackCodec(orc.Codec,
	// msgpack.Codec)
	GorpCodec encoding.Codec
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
	c.GorpCodec = override.Nil(c.GorpCodec, other.GorpCodec)
	return c
}

// Validate implements config.Config.
func (c LayerConfig) Validate() error {
	v := validate.New("distribution")
	v.NotNil("storage", c.Storage)
	v.NotEmptyString("advertise_address", c.AdvertiseAddress)
	v.NotNil("gorp_codec", c.GorpCodec)
	return v.Error()
}

// Layer contains all relevant services within the Synnax distribution layer. The
// distribution layer wraps the storage layer to provide a monolithic data space for
// working with core data structures across Synnax.
//
// The Layer must be closed when it is no longer in use. It is not safe to modify any of
// the public fields in this struct, or to access these fields after Close has been
// called.
type Layer struct {
	// DB stores the metadata of the Core.
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
}

// Open opens the distribution Layer using the provided configuration(s). Later
// configurations override the fields set in previous configurations. If the
// configuration is invalid, or any services fail to open, Open returns a nil layer and
// an error.
//
// If the returned error is nil, the Layer must be closed by calling Close after use.
// None of the services in the Layer should be used after Close is called. It is the
// caller's responsibility to ensure that the Layer is not accessed after it is closed.
func OpenLayer(ctx context.Context, cfgs ...LayerConfig) (l *Layer, err error) {
	cfg, err := config.New(LayerConfig{
		GorpCodec: encoding.NewDecodeFallbackCodec(orc.Codec, msgpack.Codec),
	}, cfgs...)
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
	l.DB = gorp.Wrap(cfg.Storage.KV, gorp.WithCodec(cfg.GorpCodec))

	if l.Channel, err = channel.NewService(ctx, channel.ServiceConfig{
		Instrumentation: cfg.Child("channel"),
		HostKey:         l.Identity.HostKey,
		KV:              cfg.Storage.KV,
		TS:              cfg.Storage.TS,
	}); !ok(err, nil) {
		return nil, err
	}

	if l.Framer, err = framer.OpenService(ctx, framer.ServiceConfig{
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

// Close closes the Layer. Close must be called when the Layer is no longer in use. It
// is the caller's responsibility to ensure that all routines interacting with the Layer
// have finished before calling Close.
func (l *Layer) Close() error { return l.closer.Close() }
