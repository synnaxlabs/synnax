// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"io"
)

type (
	Config       = core.Config
	Core         = core.Core
	Node         = core.Node
	NodeKey      = core.NodeKey
	NodeState    = core.NodeState
	Cluster      = core.Cluster
	Resolver     = aspen.Resolver
	ClusterState = aspen.ClusterState
)

var DefaultConfig = core.DefaultConfig

type Distribution struct {
	Core
	Channel  channel.Service
	Framer   *framer.Service
	Ontology *ontology.Ontology
	Signals  *signals.Provider
	Group    *group.Service
	Closers  []io.Closer
}

// Close closes the distribution layer.
func (d Distribution) Close() error {
	e := errors.NewCatcher(errors.WithAggregation())
	e.Exec(d.Ontology.Close)
	e.Exec(d.Framer.Close)
	for _, c := range d.Closers {
		e.Exec(c.Close)
	}
	e.Exec(d.Core.Close)
	return e.Error()
}

// Open opens the distribution layer for the node using the provided Config. The caller
// is responsible for closing the distribution layer when it is no longer in use.
func Open(ctx context.Context, cfg Config) (d Distribution, err error) {
	d.Core, err = core.Open(ctx, cfg)
	if err != nil {
		return d, err
	}

	gorpDB := d.Storage.Gorpify()

	if d.Ontology, err = ontology.Open(ctx, ontology.Config{
		Instrumentation: cfg.Instrumentation.Child("ontology"),
		DB:              gorpDB,
	}); err != nil {
		return d, err
	}
	if d.Group, err = group.OpenService(group.Config{
		DB:       gorpDB,
		Ontology: d.Ontology,
	}); err != nil {
		return d, err
	}

	nodeOntologySvc := &cluster.NodeOntologyService{
		Ontology: d.Ontology,
		Cluster:  d.Cluster,
	}
	clusterOntologySvc := &cluster.OntologyService{Cluster: d.Cluster}
	d.Ontology.RegisterService(clusterOntologySvc)
	d.Ontology.RegisterService(nodeOntologySvc)

	nodeOntologySvc.ListenForChanges(ctx)

	channelTransport := channeltransport.New(cfg.Pool)
	frameTransport := frametransport.New(cfg.Pool)
	*cfg.Transports = append(*cfg.Transports, channelTransport, frameTransport)

	ver, err := verification.OpenService(
		ctx,
		cfg.Verifier,
		verification.Config{
			DB:  d.Storage.KV,
			Ins: cfg.Instrumentation,
		})
	if err != nil {
		return d, err
	}
	d.Closers = append(d.Closers, ver)

	d.Channel, err = channel.New(ctx, channel.ServiceConfig{
		HostResolver:     d.Cluster,
		ClusterDB:        gorpDB,
		TSChannel:        d.Storage.TS,
		Transport:        channelTransport,
		Ontology:         d.Ontology,
		Group:            d.Group,
		IntOverflowCheck: ver.IsOverflowed,
	})
	if err != nil {
		return d, err
	}

	d.Framer, err = framer.Open(framer.Config{
		Instrumentation: cfg.Instrumentation.Child("framer"),
		ChannelReader:   d.Channel,
		TS:              d.Storage.TS,
		Transport:       frameTransport,
		HostResolver:    d.Cluster,
	})
	if err != nil {
		return d, err
	}

	if err = d.configureControlUpdates(ctx); err != nil {
		return d, err
	}

	d.Signals, err = signals.New(signals.Config{
		Channel:         d.Channel,
		Framer:          d.Framer,
		Instrumentation: cfg.Instrumentation.Child("cdc"),
	})
	if err != nil {
		return d, err
	}
	c, err := ontologycdc.Publish(ctx, d.Signals, d.Ontology)
	d.Closers = append(d.Closers, c)
	return d, err
}

func (d Distribution) configureControlUpdates(ctx context.Context) error {
	controlCh := []channel.Channel{{
		Name:        fmt.Sprintf("sy_node_%v_control", d.Cluster.HostKey()),
		Leaseholder: d.Cluster.HostKey(),
		Virtual:     true,
		DataType:    telem.StringT,
		Internal:    true,
	}}
	if err := d.Channel.CreateManyIfNamesDontExist(ctx, &controlCh); err != nil {
		return err
	}
	return d.Framer.ConfigureControlUpdateChannel(ctx, controlCh[0].Key())
}
