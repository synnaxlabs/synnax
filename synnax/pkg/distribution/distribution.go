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
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	channeltransport "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/channel"
	segmenttransport "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer"
)

type (
	Config       = core.Config
	Core         = core.Core
	Node         = core.Node
	NodeID       = core.NodeID
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
}

// Close closes the distribution layer.
func (d Distribution) Close() error { return d.Storage.Close() }

// Open opens the distribution layer for the node using the provided Config. The caller
// is responsible for closing the distribution layer when it is no longer in use.
func Open(ctx context.Context, cfg Config) (d Distribution, err error) {
	d.Core, err = core.Open(ctx, cfg)
	if err != nil {
		return d, err
	}

	gorpDB := d.Storage.Gorpify()

	d.Ontology, err = ontology.Open(ctx, gorpDB)
	if err != nil {
		return d, err
	}

	nodeOntologySvc := &core.NodeOntologyService{
		Ontology: d.Ontology,
		Cluster:  d.Cluster,
	}
	clusterOntologySvc := &core.ClusterOntologyService{Cluster: d.Cluster}
	d.Ontology.RegisterService(clusterOntologySvc)
	d.Ontology.RegisterService(nodeOntologySvc)
	nodeOntologySvc.ListenForChanges(ctx)

	channelTransport := channeltransport.New(cfg.Pool)
	segmentTransport := segmenttransport.New(cfg.Pool)
	*cfg.Transports = append(*cfg.Transports, channelTransport, segmentTransport)

	d.Channel, err = channel.New(channel.ServiceConfig{
		HostResolver: d.Cluster,
		ClusterDB:    gorpDB,
		TSChannel:    d.Storage.TS,
		Transport:    channelTransport,
		Ontology:     d.Ontology,
	})
	if err != nil {
		return d, err
	}
	d.Ontology.RegisterService(d.Channel)

	d.Framer, err = framer.Open(framer.ServiceConfig{
		Instrumentation: cfg.Instrumentation.Sub("framer"),
		ChannelReader:   d.Channel,
		TS:              d.Storage.TS,
		Transport:       segmentTransport,
		HostResolver:    d.Cluster,
	})

	return d, err
}
