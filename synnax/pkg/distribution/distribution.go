package distribution

import (
	"context"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment"
	channeltransport "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/channel"
	segmenttransport "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/segment"
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
	Channel  *channel.Service
	Segment  *segment.Service
	Ontology *ontology.Ontology
}

// Close closes the distribution layer.
func (d Distribution) Close() error { return d.Storage.Close() }

// Open opens the distribution layer for the node using the provided Config. The caller is responsible for closing the
// distribution layer when it is no longer in use.
func Open(ctx context.Context, cfg Config) (d Distribution, err error) {
	d.Core, err = core.Open(ctx, cfg)
	if err != nil {
		return d, err
	}

	gorpDB := d.Storage.Gorpify()

	d.Ontology, err = ontology.Open(gorpDB)
	if err != nil {
		return d, err
	}

	channelClient, channelServer := channeltransport.New(cfg.Pool)
	segmentTransport := segmenttransport.New(cfg.Pool)
	*cfg.Transports = append(*cfg.Transports, channelServer, segmentTransport)
	d.Channel = channel.New(d.Cluster, gorpDB, d.Storage.TS, channelClient, channelServer)
	d.Segment = segment.New(d.Channel, d.Storage.TS, segmentTransport, d.Cluster, cfg.Logger)

	return d, nil
}
