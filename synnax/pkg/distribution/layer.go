// Copyright 2025 Synnax Labs, Inc.
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
	"fmt"
	"io"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel/verification"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	ontologycdc "github.com/synnaxlabs/synnax/pkg/distribution/ontology/signals"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/layer"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening the distribution layer.  See fields for
// details on defining the configuration.
type Config struct {
	alamos.Instrumentation
	// Verifier is for verifying. Magic.
	//
	// [REQUIRED]
	Verifier string
	// TestingIntOverflowCheck is used for overriding default verifier behavior
	// for testing purposes only.
	TestingIntOverflowCheck channel.IntOverflowChecker
	// EnableSearch sets whether search indexing is enabled for cluster resources.
	//
	// [OPTIONAL] - Defaults to true
	EnableSearch          *bool
	ChannelTransport      channel.Transport
	FrameTransport        framer.Transport
	AspenTransport        aspen.Transport
	AdvertiseAddress      address.Address
	PeerAddresses         []address.Address
	Storage               *storage.Layer
	AspenOptions          []aspen.Option
	EnableOntologySignals *bool
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening the distribution layer.
	// This configuration is not valid on its own and must be overridden by the
	// required fields specific in Config.
	DefaultConfig = Config{EnableSearch: config.True(), EnableOntologySignals: config.True()}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.EnableSearch = override.Nil(c.EnableSearch, other.EnableSearch)
	c.Verifier = override.String(c.Verifier, other.Verifier)
	c.ChannelTransport = override.Nil(c.ChannelTransport, other.ChannelTransport)
	c.FrameTransport = override.Nil(c.FrameTransport, other.FrameTransport)
	c.AspenTransport = override.Nil(c.AspenTransport, other.AspenTransport)
	c.Storage = override.Nil(c.Storage, other.Storage)
	c.AdvertiseAddress = override.String(c.AdvertiseAddress, other.AdvertiseAddress)
	c.PeerAddresses = override.Slice(c.PeerAddresses, other.PeerAddresses)
	c.AspenOptions = override.Slice(c.AspenOptions, other.AspenOptions)
	c.EnableOntologySignals = override.Nil(c.EnableOntologySignals, other.EnableOntologySignals)
	c.TestingIntOverflowCheck = override.Nil(c.TestingIntOverflowCheck, other.TestingIntOverflowCheck)
	return c
}

// Validate implements config.Config. It does nothing and leaves
// validation to the individual components.
func (c Config) Validate() error {
	v := validate.New("distribution")
	validate.NotNil(v, "frame_transport", c.FrameTransport)
	validate.NotNil(v, "channel_transport", c.ChannelTransport)
	validate.NotNil(v, "storage", c.Storage)
	validate.NotNil(v, "aspen_transport", c.AspenTransport)
	validate.NotNil(v, "enable_search", c.EnableSearch)
	validate.NotNil(v, "enable_ontology_signals", c.EnableOntologySignals)
	return v.Error()
}

// Layer contains all relevant services within the Synnax distribution layer.
// The distribution layer wraps the storage layer to provide a monolithic data space
// for working with core data structures across Synnax.
type Layer struct {
	cfg Config
	// KV is an eventually consistent, distributed key-value store that synchronizes key-value pairs across the
	// cluster.
	KV kvx.DB
	// Cluster provides information about the cluster topology. Nodes, keys, addresses, states, etc.
	Cluster cluster.Cluster
	// Channels is for creating, deleting, and retrieving channels across the cluster.
	Channels channel.Service
	// Framer is for reading, writing, and streaming frames of telemetry across the
	// cluster.
	Framer *framer.Service
	// Ontology manages relationships between arbitrary data structures in a directed
	// acyclic graph. It is the main method for defining relationships between resources
	// in Synnax.
	Ontology *ontology.Ontology
	// Signals are for propagating changes to data structures through channels in
	// Synnax.
	Signals *signals.Provider
	// Group is for grouping related resources in the cluster.
	Group *group.Service
	// Verification verifies that the universe remains as it is.
	Verification *verification.Service
	// closer is for properly shutting down the distribution layer.
	closer xio.MultiCloser
}

// GorpDB returns a gorp.DB that can be used to interact with the storage key-value store.
func (l *Layer) GorpDB() *gorp.DB {
	return gorp.Wrap(
		l.KV,
		gorp.WithCodec(&binary.TracingCodec{
			Level:           alamos.Bench,
			Instrumentation: l.cfg.Instrumentation,
			Codec:           &binary.MsgPackCodec{},
		}),
	)
}

// Close closes the distribution layer, returning any error encountered.
func (l *Layer) Close() error { return l.closer.Close() }

// Open opens the distribution layer using the provided configurations. Later
// configurations override the fields set in previous ones. If the configuration is
// invalid, or any services fail to open, Open returns a nil layer and an error. If the
// return error is nil, the Layer must be closed by calling Close after use.
func Open(ctx context.Context, cfgs ...Config) (*Layer, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	l := &Layer{cfg: cfg}
	cleanup, ok := layer.NewOpener(ctx, &err, &l.closer)
	defer cleanup()
	aspenOptions := append([]aspen.Option{
		aspen.WithEngine(cfg.Storage.KV),
		aspen.WithTransport(cfg.AspenTransport),
		aspen.WithInstrumentation(cfg.Instrumentation.Child("aspen")),
	}, cfg.AspenOptions...)

	var aspenDB *aspen.DB
	// Since we're using our own key-value engine, the value we use for 'dirname'
	// doesn't matter.
	if aspenDB, err = aspen.Open(
		ctx,
		/* dirname */ "",
		cfg.AdvertiseAddress,
		cfg.PeerAddresses,
		aspenOptions...,
	); !ok(aspenDB) {
		return nil, err
	}
	l.Cluster = aspenDB.Cluster
	l.KV = aspenDB.DB
	gorpDB := l.GorpDB()

	if l.Ontology, err = ontology.Open(
		ctx,
		ontology.Config{
			Instrumentation: cfg.Instrumentation.Child("ontology"),
			DB:              gorpDB,
		},
	); !ok(l.Ontology) {
		return nil, err
	}
	if l.Group, err = group.OpenService(
		ctx,
		group.Config{
			DB:       gorpDB,
			Ontology: l.Ontology,
		},
	); !ok(l.Group) {
		return nil, err
	}

	nodeOntologySvc := &cluster.NodeOntologyService{
		Ontology: l.Ontology,
		Cluster:  l.Cluster,
	}
	clusterOntologySvc := &cluster.OntologyService{Cluster: l.Cluster}
	l.Ontology.RegisterService(ctx, clusterOntologySvc)
	l.Ontology.RegisterService(ctx, nodeOntologySvc)

	nodeOntologySvc.ListenForChanges(ctx)

	if l.Verification, err = verification.OpenService(ctx, verification.Config{
		Verifier: cfg.Verifier,
		DB:       l.KV,
		Ins:      cfg.Instrumentation,
	}); !ok(l.Verification) {
		return nil, err
	}

	if l.Channels, err = channel.New(ctx, channel.ServiceConfig{
		HostResolver: l.Cluster,
		ClusterDB:    gorpDB,
		TSChannel:    cfg.Storage.TS,
		Transport:    cfg.ChannelTransport,
		Ontology:     l.Ontology,
		Group:        l.Group,
		IntOverflowCheck: lo.Ternary(
			cfg.TestingIntOverflowCheck != nil,
			cfg.TestingIntOverflowCheck,
			l.Verification.IsOverflowed,
		),
	}); !ok(nil) {
		return nil, err
	}

	if l.Framer, err = framer.Open(framer.Config{
		Instrumentation: cfg.Instrumentation.Child("framer"),
		ChannelReader:   l.Channels,
		TS:              cfg.Storage.TS,
		Transport:       cfg.FrameTransport,
		HostResolver:    l.Cluster,
	}); !ok(l.Framer) {
		return nil, err
	}

	if err = l.configureControlUpdates(ctx); !ok(nil) {
		return nil, err
	}

	if l.Signals, err = signals.New(signals.Config{
		Channel:         l.Channels,
		Framer:          l.Framer,
		Instrumentation: cfg.Instrumentation.Child("signals"),
	}); !ok(nil) {
		return nil, err
	}
	if *cfg.EnableOntologySignals {
		var ontologyCDCCloser io.Closer
		if ontologyCDCCloser, err = ontologycdc.Publish(
			ctx,
			l.Signals,
			l.Ontology,
		); !ok(ontologyCDCCloser) {
			return nil, err
		}
	}
	return l, err
}

func (l Layer) configureControlUpdates(ctx context.Context) error {
	controlCh := channel.Channel{
		Name:        fmt.Sprintf("sy_node_%v_control", l.Cluster.HostKey()),
		Leaseholder: l.Cluster.HostKey(),
		Virtual:     true,
		DataType:    telem.StringT,
		Internal:    true,
	}
	if err := l.Channels.Create(ctx, &controlCh, channel.RetrieveIfNameExists(true)); err != nil {
		return err
	}
	return l.Framer.ConfigureControlUpdateChannel(ctx, controlCh.Key(), controlCh.Name)
}
