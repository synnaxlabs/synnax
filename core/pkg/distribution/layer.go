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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// LayerConfig is the configuration for opening the distribution layer.  See fields for
// details on defining the configuration.
type LayerConfig struct {
	// ChannelTransport is the network transport used for channel-related RPCs.
	//
	// [REQUIRED]
	ChannelTransport channel.Transport
	// FramerTransport is the network transport used for moving telemetry frames.
	//
	// [REQUIRED]
	FrameTransport framer.Transport
	// GorpCodec sets the codec used to encode/decode data structures within the
	// cluster meta-data DB (gorp).
	//
	// [OPTIONAL] - Defaults to &binary.MsgPackCodec
	GorpCodec encoding.Codec
	// AspenTransport is the network transport used for key-value gossip and cluster
	// topology information.
	//
	// [REQUIRED]
	AspenTransport aspen.Transport
	// Instrumentation is for logging, tracing, and metrics.
	//
	// Storage is the storage layer that the distribution layer will use for persisting
	// data across its various services.
	//
	// [REQUIRED]
	Storage *storage.Layer
	// ValidateChannelNames disables channel name validation when true.
	// This allows channels with special characters, spaces, etc.
	//
	// [OPTIONAL] - Defaults to true (validation enabled)
	ValidateChannelNames *bool
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// AdvertiseAddress sets the network address that the distribution layer will publish
	// to other nodes in the cluster.
	//
	// [REQUIRED]
	AdvertiseAddress address.Address
	// AspenOptions are additional options to pass when opening the aspen key-value
	// store.
	//
	// [OPTIONAL] - Defaults to []
	AspenOptions []aspen.Option
	// PeerAddresses sets the list of peer nodes in the cluster that the distribution
	// layer will reach out to join the cluster. If this slice is empty, the distribution
	// layer will bootstrap a new single node cluster.
	//
	// [OPTIONAL] - Defaults to []
	PeerAddresses []address.Address
}

var (
	_ config.Config[LayerConfig] = LayerConfig{}
	// DefaultLayerConfig is the default configuration for opening the distribution layer.
	// This configuration is not valid on its own and must be overridden by the
	// required fields specific in Config.
	DefaultLayerConfig = LayerConfig{
		GorpCodec:            orc.NewCodec(msgpack.Codec),
		ValidateChannelNames: new(true),
	}
)

// Override implements config.Config.
func (c LayerConfig) Override(other LayerConfig) LayerConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Storage = override.Nil(c.Storage, other.Storage)
	c.AdvertiseAddress = override.String(c.AdvertiseAddress, other.AdvertiseAddress)
	c.PeerAddresses = override.Slice(c.PeerAddresses, other.PeerAddresses)
	c.ChannelTransport = override.Nil(c.ChannelTransport, other.ChannelTransport)
	c.FrameTransport = override.Nil(c.FrameTransport, other.FrameTransport)
	c.AspenTransport = override.Nil(c.AspenTransport, other.AspenTransport)
	c.AspenOptions = override.Slice(c.AspenOptions, other.AspenOptions)
	c.GorpCodec = override.Nil(c.GorpCodec, other.GorpCodec)
	c.ValidateChannelNames = override.Nil(c.ValidateChannelNames, other.ValidateChannelNames)
	return c
}

// Validate implements config.Config. It does nothing and leaves
// validation to the individual components.
func (c LayerConfig) Validate() error {
	v := validate.New("distribution")
	validate.NotNil(v, "storage", c.Storage)
	validate.NotEmptyString(v, "advertise_address", c.AdvertiseAddress)
	validate.NotNil(v, "channel_transport", c.ChannelTransport)
	validate.NotNil(v, "frame_transport", c.FrameTransport)
	validate.NotNil(v, "aspen_transport", c.AspenTransport)
	validate.NotNil(v, "codec", c.GorpCodec)
	validate.NotNil(v, "disable_channel_name_validation", c.ValidateChannelNames)
	return v.Error()
}

// Layer contains all relevant services within the Synnax distribution layer.
// The distribution layer wraps the storage layer to provide a monolithic data space
// for working with core data structures across Synnax.
//
// The Layer must be closed when it is no longer in use. It is not safe to modify any
// of the public fields in this struct, or to access these fields after Close has
// been called.
type Layer struct {
	// DB is the database for storing cluster wide meta-data.
	DB *gorp.DB
	// Cluster provides information about the cluster topology. Nodes, keys, addresses,
	// states, etc.
	Cluster node.Cluster
	// Channel is the distribution-layer channel allocator: it assigns local keys and
	// creates, renames, and deletes storage channels across the cluster. Rich channel
	// metadata and retrieval live in the service layer.
	Channel *channel.Service
	// ChannelRetriever is a late-bound hole for reading channel metadata, filled by the
	// service layer once its channel service opens. The framer reads channels through
	// it.
	ChannelRetriever *channel.RetrieverHolder
	// ValidateChannelNames reports whether channel name validation is enabled.
	ValidateChannelNames *bool
	// Framer is for reading, writing, and streaming frames of telemetry across the
	// cluster.
	Framer *framer.Service
	// Ontology manages relationships between arbitrary data structures in a directed
	// acyclic graph. It is the main method for defining relationships between resources
	// in Synnax.
	Ontology *ontology.Ontology
	// Search is the full-text search index for ontology resources.
	// [REQUIRED]
	Search *search.Index
	// Group is for grouping related resources in the cluster.
	Group *group.Service
	// closer is for properly shutting down the distribution layer.
	closer xio.MultiCloser
}

// Open opens the distribution Layer using the provided configuration(s). Later
// configurations override the fields set in previous configurations. If the configuration is
// invalid, or any services fail to open, Open returns a nil layer and an error.
//
// If the returned error is nil, the Layer must be closed by calling Close after use.
// None of the services in the Layer should be used after Close is called. It is the
// caller's responsibility to ensure that the Layer is not accessed after it is closed.
func OpenLayer(ctx context.Context, cfgs ...LayerConfig) (l *Layer, err error) {
	cfg, err := config.New(DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	l = &Layer{}
	cleanup, ok := service.NewOpener(ctx, &l.closer)
	defer func() {
		err = cleanup(err)
	}()

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

	if l.Ontology, err = ontology.Open(
		ctx,
		ontology.Config{
			Instrumentation: cfg.Child("ontology"),
			DB:              l.DB,
		},
	); !ok(err, l.Ontology) {
		return nil, err
	}

	if l.Search, err = search.Open(search.Config{
		Instrumentation: cfg.Child("search"),
	}); err != nil {
		return nil, err
	}

	if l.Group, err = group.OpenService(
		ctx,
		group.ServiceConfig{
			Instrumentation: cfg.Child("group"),
			DB:              l.DB,
			Ontology:        l.Ontology,
			Search:          l.Search,
		},
	); !ok(err, l.Group) {
		return nil, err
	}

	l.ValidateChannelNames = cfg.ValidateChannelNames
	l.ChannelRetriever = channel.NewRetrieverHolder()

	if l.Channel, err = channel.OpenService(ctx, channel.ServiceConfig{
		Instrumentation: cfg.Child("channel"),
		HostResolver:    l.Cluster,
		ClusterDB:       l.DB,
		TSChannel:       cfg.Storage.TS,
		Transport:       cfg.ChannelTransport,
	}); !ok(err, l.Channel) {
		return nil, err
	}

	if l.Framer, err = framer.OpenService(ctx, framer.ServiceConfig{
		Instrumentation: cfg.Child("framer"),
		Channel:         l.ChannelRetriever,
		TS:              cfg.Storage.TS,
		Transport:       cfg.FrameTransport,
		HostResolver:    l.Cluster,
	}); !ok(err, l.Framer) {
		return nil, err
	}

	return l, err
}

// Close closes the Layer. Close must be called when the Layer is no longer in use.
// the caller must ensure that all routines interacting with the Layer have finished
// before calling Close.
func (l *Layer) Close() error { return l.closer.Close() }
