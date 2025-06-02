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

	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel/verification"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	ontologycdc "github.com/synnaxlabs/synnax/pkg/distribution/ontology/signals"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	channeltransport "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/channel"
	frametransport "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer"
	"github.com/synnaxlabs/synnax/pkg/layer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
)

type (
	Core         = core.Core
	Node         = core.Node
	NodeKey      = core.NodeKey
	NodeState    = core.NodeState
	Cluster      = core.Cluster
	Resolver     = aspen.Resolver
	ClusterState = aspen.ClusterState
)

// Config is the configuration for opening the distribution layer. It extends
// core.Config with additional fields.
type Config struct {
	core.Config
	// Verifier is for verifying. Magic.
	// [REQUIRED}
	Verifier string
	// Ontology is an optional ontology configuration used to override properties on the
	// internally build ontology configuration.
	// [OPTIONAL]
	Ontology ontology.Config
	// Channel is an optional channel configuration used to override properties on the
	// internally build channel configuration.
	// [OPTIONAL]
	Channel channel.ServiceConfig
	// Group is an optional group configuration used to override properties on the
	// internally build group configuration.
	// [OPTIONAL]
	Group group.Config
	// Signals is an optional signals configuration used to override properties on the
	// internally build signals configuration.
	// [OPTIONAL]
	Signals signals.Config
	// Framer is an optional framer configuration used to override properties on the
	// internally build framer configuration.
	// [OPTIONAL]
	Framer framer.Config
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{Config: core.DefaultConfig}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Config = c.Config.Override(other.Config)
	c.Channel = c.Channel.Override(other.Channel)
	c.Ontology = c.Ontology.Override(other.Ontology)
	c.Group = c.Group.Override(other.Group)
	c.Signals = c.Signals.Override(other.Signals)
	c.Framer = c.Framer.Override(other.Framer)
	c.Verifier = override.String(c.Verifier, other.Verifier)
	return c
}

// Validate implements config.Config. It does nothing, and leaves
// validation to the individual components.
func (c Config) Validate() error { return nil }

type Layer struct {
	*Core
	Channel      channel.Service
	Framer       *framer.Service
	Ontology     *ontology.Ontology
	Signals      *signals.Provider
	Group        *group.Service
	Verification *verification.Service
	closer       xio.MultiCloser
}

// Close closes the distribution layer.
func (l *Layer) Close() error {
	e := errors.NewCatcher(errors.WithAggregation())
	e.Exec(l.closer.Close)
	return e.Error()
}

// Open opens the distribution layer for the node using the provided Config. The caller
// is responsible for closing the distribution layer when it is no longer in use.
func Open(ctx context.Context, cfg Config) (*Layer, error) {
	var (
		l   = &Layer{}
		err error
	)
	cleanup, ok := layer.NewOpener(ctx, &err, &l.closer)
	defer cleanup()
	if l.Core, err = core.Open(ctx, cfg.Config); !ok(l.Core) {
		return nil, err
	}
	gorpDB := l.Storage.Gorpify()
	if l.Ontology, err = ontology.Open(ctx,
		cfg.Ontology,
		ontology.Config{
			Instrumentation: cfg.Instrumentation.Child("ontology"),
			DB:              gorpDB,
		},
	); !ok(l.Ontology) {
		return nil, err
	}
	if l.Group, err = group.OpenService(
		ctx,
		cfg.Group,
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

	channelTransport := channeltransport.New(cfg.Pool)
	frameTransport := frametransport.New(cfg.Pool)
	*cfg.Transports = append(*cfg.Transports, channelTransport, frameTransport)

	if l.Verification, err = verification.OpenService(ctx, verification.Config{
		Verifier: cfg.Verifier,
		DB:       l.Storage.KV,
		Ins:      cfg.Instrumentation,
	}); !ok(l.Verification) {
		return nil, err
	}

	if l.Channel, err = channel.New(ctx, cfg.Channel, channel.ServiceConfig{
		HostResolver:     l.Cluster,
		ClusterDB:        gorpDB,
		TSChannel:        l.Storage.TS,
		Transport:        channelTransport,
		Ontology:         l.Ontology,
		Group:            l.Group,
		IntOverflowCheck: l.Verification.IsOverflowed,
	}); !ok(nil) {
		return nil, err
	}

	if l.Framer, err = framer.Open(cfg.Framer, framer.Config{
		Instrumentation: cfg.Instrumentation.Child("framer"),
		ChannelReader:   l.Channel,
		TS:              l.Storage.TS,
		Transport:       frameTransport,
		HostResolver:    l.Cluster,
	}); !ok(nil) {
		return nil, err
	}

	if err = l.configureControlUpdates(ctx); !ok(nil) {
		return nil, err
	}

	if l.Signals, err = signals.New(cfg.Signals, signals.Config{
		Channel:         l.Channel,
		Framer:          l.Framer,
		Instrumentation: cfg.Instrumentation.Child("signals"),
	}); !ok(nil) {
		return nil, err
	}
	c, err := ontologycdc.Publish(ctx, l.Signals, l.Ontology)
	if err != nil {
		return nil, err
	}
	l.closer = append(l.closer, c)
	return l, err
}

func (l Layer) configureControlUpdates(ctx context.Context) error {
	controlCh := []channel.Channel{{
		Name:        fmt.Sprintf("sy_node_%v_control", l.Cluster.HostKey()),
		Leaseholder: l.Cluster.HostKey(),
		Virtual:     true,
		DataType:    telem.StringT,
		Internal:    true,
	}}
	if err := l.Channel.CreateMany(ctx, &controlCh, channel.RetrieveIfNameExists(true)); err != nil {
		return err
	}
	return l.Framer.ConfigureControlUpdateChannel(ctx, controlCh[0].Key(), controlCh[0].Name)
}
