// Copyright 2023 Synnax Labs, Inc.
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
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
)

type Builder struct {
	mock.CoreBuilder
	Nodes      []distribution.Distribution
	writerNet  *tmock.FramerWriterNetwork
	iterNet    *tmock.FramerIteratorNetwork
	channelNet *tmock.ChannelNetwork
}

func NewBuilder(cfg ...distribution.Config) *Builder {
	coreBuilder := mock.NewCoreBuilder(cfg...)

	return &Builder{
		CoreBuilder: *coreBuilder,
		writerNet:   tmock.NewFramerWriterNetwork(),
		iterNet:     tmock.NewFramerIteratorNetwork(),
		channelNet:  tmock.NewChannelNetwork(),
	}
}

func (b *Builder) New(ctx context.Context) distribution.Distribution {
	core := b.CoreBuilder.New()
	d := distribution.Distribution{Core: core}

	trans := mockFramerTransport{
		iter:   b.iterNet.New(core.Config.AdvertiseAddress, 1),
		writer: b.writerNet.New(core.Config.AdvertiseAddress, 1),
	}

	d.Ontology = lo.Must(ontology.Open(ctx, ontology.Config{DB: d.Storage.Gorpify()}))

	nodeOntologySvc := &dcore.NodeOntologyService{
		Cluster:  d.Cluster,
		Ontology: d.Ontology,
	}
	clusterOntologySvc := &dcore.ClusterOntologyService{Cluster: d.Cluster}
	d.Ontology.RegisterService(nodeOntologySvc)
	d.Ontology.RegisterService(clusterOntologySvc)
	nodeOntologySvc.ListenForChanges(ctx)

	d.Channel = lo.Must(channel.New(channel.ServiceConfig{
		HostResolver: d.Cluster,
		ClusterDB:    d.Storage.Gorpify(),
		TSChannel:    d.Storage.TS,
		Transport:    b.channelNet.New(d.Config.AdvertiseAddress),
		Ontology:     d.Ontology,
	}))

	d.Framer = lo.Must(framer.Open(framer.ServiceConfig{
		ChannelReader: d.Channel,
		TS:            d.Storage.TS,
		HostResolver:  d.Cluster,
		Transport:     trans,
	}))
	return d
}

type mockFramerTransport struct {
	iter   iterator.Transport
	writer writer.Transport
}

var _ framer.Transport = (*mockFramerTransport)(nil)

func (m mockFramerTransport) Iterator() iterator.Transport {
	return m.iter
}

func (m mockFramerTransport) Writer() writer.Transport {
	return m.writer
}
