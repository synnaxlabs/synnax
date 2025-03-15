// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	ontologycdc "github.com/synnaxlabs/synnax/pkg/distribution/ontology/signals"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
)

type Builder struct {
	cfg        distribution.Config
	core       mock.CoreBuilder
	Nodes      map[dcore.NodeKey]distribution.Distribution
	writerNet  *tmock.FramerWriterNetwork
	iterNet    *tmock.FramerIteratorNetwork
	channelNet *tmock.ChannelNetwork
	relayNet   *tmock.FramerRelayNetwork
	deleteNet  *tmock.FramerDeleterNetwork
}

func NewBuilder(cfgs ...distribution.Config) *Builder {
	cfg := lo.Must(config.New[distribution.Config](distribution.Config{}, cfgs...))
	coreBuilder := mock.NewCoreBuilder(cfg.Config)
	return &Builder{
		cfg:        cfg,
		core:       *coreBuilder,
		writerNet:  tmock.NewWriterNetwork(),
		iterNet:    tmock.NewIteratorNetwork(),
		channelNet: tmock.NewChannelNetwork(),
		relayNet:   tmock.NewRelayNetwork(),
		deleteNet:  tmock.NewDeleterNetwork(),
		Nodes:      make(map[dcore.NodeKey]distribution.Distribution),
	}
}

func (b *Builder) New(ctx context.Context) distribution.Distribution {
	core := b.core.New()
	d := distribution.Distribution{Core: core}

	trans := mockFramerTransport{
		iter:    b.iterNet.New(core.Config.AdvertiseAddress, 1),
		writer:  b.writerNet.New(core.Config.AdvertiseAddress, 1),
		relay:   b.relayNet.New(core.Config.AdvertiseAddress, 1),
		deleter: b.deleteNet.New(core.Config.AdvertiseAddress),
	}

	d.Ontology = lo.Must(ontology.Open(ctx, b.cfg.Ontology, ontology.Config{DB: d.Storage.Gorpify()}))
	d.Group = lo.Must(group.OpenService(b.cfg.Group, group.Config{Ontology: d.Ontology, DB: d.Storage.Gorpify()}))

	nodeOntologySvc := &cluster.NodeOntologyService{
		Cluster:  d.Cluster,
		Ontology: d.Ontology,
	}
	clusterOntologySvc := &cluster.OntologyService{Cluster: d.Cluster}
	d.Ontology.RegisterService(nodeOntologySvc)
	d.Ontology.RegisterService(clusterOntologySvc)
	nodeOntologySvc.ListenForChanges(ctx)

	d.Channel = lo.Must(channel.New(ctx, b.cfg.Channel, channel.ServiceConfig{
		HostResolver:     d.Cluster,
		ClusterDB:        d.Storage.Gorpify(),
		TSChannel:        d.Storage.TS,
		Transport:        b.channelNet.New(d.Config.AdvertiseAddress),
		Ontology:         d.Ontology,
		Group:            d.Group,
		IntOverflowCheck: func(ctx context.Context, count types.Uint20) error { return nil },
	}))

	d.Framer = lo.Must(framer.Open(b.cfg.Framer, framer.Config{
		Instrumentation: d.Instrumentation,
		ChannelReader:   d.Channel,
		TS:              d.Storage.TS,
		HostResolver:    d.Cluster,
		Transport:       trans,
	}))

	d.Signals = lo.Must(signals.New(b.cfg.Signals, signals.Config{
		Instrumentation: d.Instrumentation,
		Channel:         d.Channel,
		Framer:          d.Framer,
	}))

	// If we're not the bootstrapper, don't propagate changes to prevent issues when
	// trying to find free channels. We're going to resolve this issue in #105:
	// https://github.com/synnaxlabs/synnax/issues/105
	if d.Cluster.HostKey().IsBootstrapper() {
		d.Closers = append(d.Closers, lo.Must(ontologycdc.Publish(ctx, d.Signals, d.Ontology)))
	}

	b.Nodes[core.Cluster.HostKey()] = d

	return d
}

func (b *Builder) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, node := range b.Nodes {
		c.Exec(node.Close)
	}
	return c.Error()
}

func (b *Builder) Cleanup() error {
	return b.core.Cleanup()
}

type mockFramerTransport struct {
	iter    iterator.Transport
	writer  writer.Transport
	relay   relay.Transport
	deleter deleter.Transport
}

var _ framer.Transport = (*mockFramerTransport)(nil)

func (m mockFramerTransport) Iterator() iterator.Transport {
	return m.iter
}

func (m mockFramerTransport) Writer() writer.Transport {
	return m.writer
}

func (m mockFramerTransport) Relay() relay.Transport {
	return m.relay
}

func (m mockFramerTransport) Deleter() deleter.Transport {
	return m.deleter
}
