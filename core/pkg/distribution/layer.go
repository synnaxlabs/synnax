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
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// Transport bundles the node-to-node transports the distribution layer requires.
type Transport interface {
	// Channel returns the transport for channel create, rename, and delete RPCs.
	Channel() channel.Transport
	// Framer returns the transport for frame write, iterate, relay, and delete
	// operations.
	Framer() framer.Transport
}

// LayerConfig is the configuration for opening the distribution layer.
type LayerConfig struct {
	// Transport bundles the network transports used for channel and framer node-to-node
	// RPCs.
	//
	// [REQUIRED]
	Transport Transport
	// GorpCodec sets the codec used to encode/decode data structures within the cluster
	// metadata DB.
	//
	// [OPTIONAL] - Defaults to orc.NewCodec(msgpack.Codec)
	GorpCodec encoding.Codec
	// AspenTransport is the network transport used for key-value gossip and cluster
	// topology information.
	//
	// [REQUIRED]
	AspenTransport aspen.Transport
	// Storage is the storage layer that the distribution layer will use for persisting
	// data across its various services.
	//
	// [REQUIRED]
	Storage *storage.Layer
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// AdvertiseAddress sets the network address that the distribution layer will
	// publish to other nodes in the cluster.
	//
	// [REQUIRED]
	AdvertiseAddress address.Address
	// AspenOptions are additional options to pass when opening the Aspen key-value
	// store.
	//
	// [OPTIONAL] - Defaults to []
	AspenOptions []aspen.Option
	// PeerAddresses sets the list of peer nodes in the cluster that the distribution
	// layer will reach out to join the cluster. If this slice is empty, the
	// distribution layer will bootstrap a new single node cluster.
	//
	// [OPTIONAL] - Defaults to []
	PeerAddresses []address.Address
}

var _ config.Config[LayerConfig] = LayerConfig{}

// Override implements config.Config.
func (c LayerConfig) Override(other LayerConfig) LayerConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Storage = override.Nil(c.Storage, other.Storage)
	c.AdvertiseAddress = override.String(c.AdvertiseAddress, other.AdvertiseAddress)
	c.PeerAddresses = override.Slice(c.PeerAddresses, other.PeerAddresses)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.AspenTransport = override.Nil(c.AspenTransport, other.AspenTransport)
	c.AspenOptions = override.Slice(c.AspenOptions, other.AspenOptions)
	c.GorpCodec = override.Nil(c.GorpCodec, other.GorpCodec)
	return c
}

// Validate implements config.Config.
func (c LayerConfig) Validate() error {
	v := validate.New("distribution")
	validate.NotNil(v, "storage", c.Storage)
	validate.NotEmptyString(v, "advertise_address", c.AdvertiseAddress)
	validate.NotNil(v, "transport", c.Transport)
	validate.NotNil(v, "aspen_transport", c.AspenTransport)
	validate.NotNil(v, "gorp_codec", c.GorpCodec)
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
	// DB is the database for storing cluster wide metadata.
	DB *gorp.DB
	// Cluster provides information about the cluster topology. Nodes, keys, addresses,
	// states, etc.
	Cluster cluster.Cluster
	// Channel is the distribution-layer channel allocator: it assigns local keys and
	// creates, renames, and deletes storage channels across the cluster.
	Channel *channel.Service
	// Framer is for reading, writing, and streaming frames of telemetry across the
	// cluster.
	Framer *framer.Service
	closer io.MultiCloser
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
	cfg, err := config.New(LayerConfig{GorpCodec: orc.NewCodec(msgpack.Codec)}, cfgs...)
	if err != nil {
		return nil, err
	}
	l = &Layer{}
	cleanup, ok := service.NewOpener(ctx, &l.closer)
	defer func() { err = cleanup(err) }()

	aspenOptions := append([]aspen.Option{
		aspen.WithEngine(cfg.Storage.KV),
		aspen.WithTransport(cfg.AspenTransport),
		aspen.WithInstrumentation(cfg.Child("aspen")),
	}, cfg.AspenOptions...)

	// Since we're using our own key-value engine, the value we use for 'dirname'
	// doesn't matter.
	var aspenDB *aspen.DB
	if aspenDB, err = aspen.Open(
		ctx,
		"",
		cfg.AdvertiseAddress,
		cfg.PeerAddresses,
		aspenOptions...,
	); !ok(err, aspenDB) {
		return nil, err
	}
	l.Cluster = aspenDB.Cluster
	l.DB = gorp.Wrap(
		aspenDB,
		gorp.WithCodec(cfg.GorpCodec),
		gorp.WithIndexObservable(aspenDB.NewObservable(aspen.IgnoreHostLeaseholder)),
	)

	if l.Channel, err = channel.NewService(ctx, channel.ServiceConfig{
		Instrumentation: cfg.Child("channel"),
		HostResolver:    l.Cluster,
		KV:              l.DB,
		TS:              cfg.Storage.TS,
		Transport:       cfg.Transport.Channel(),
	}); !ok(err, nil) {
		return nil, err
	}

	if l.Framer, err = framer.OpenService(ctx, framer.ServiceConfig{
		Instrumentation: cfg.Child("framer"),
		TS:              cfg.Storage.TS,
		Transport:       cfg.Transport.Framer(),
		HostResolver:    l.Cluster,
	}); !ok(err, l.Framer) {
		return nil, err
	}

	return l, nil
}

// Close closes the Layer. Close must be called when the Layer is no longer in use. It
// is the caller's responsibility to ensure that all routines interacting with the Layer
// have finished before calling Close.
func (l *Layer) Close() error { return l.closer.Close() }
