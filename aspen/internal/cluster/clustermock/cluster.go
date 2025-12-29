// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package clustermock

import (
	"context"

	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
)

type Builder struct {
	Configs     []cluster.Config
	GossipNet   *fmock.Network[gossip.Message, gossip.Message]
	PledgeNet   *fmock.Network[pledge.Request, pledge.Response]
	ClusterAPIs map[node.Key]*cluster.Cluster
}

func NewBuilder(cfgs ...cluster.Config) *Builder {
	return &Builder{
		Configs:     cfgs,
		GossipNet:   fmock.NewNetwork[gossip.Message, gossip.Message](),
		PledgeNet:   fmock.NewNetwork[pledge.Request, pledge.Response](),
		ClusterAPIs: make(map[node.Key]*cluster.Cluster),
	}
}

func (b *Builder) New(ctx context.Context, cfgs ...cluster.Config) (*cluster.Cluster, error) {
	gossipServer := b.GossipNet.UnaryServer("")
	pledgeServer := b.PledgeNet.UnaryServer(gossipServer.Address)
	cfgs = append(b.Configs, cfgs...)
	cfgs = append(cfgs, cluster.Config{
		HostAddress: gossipServer.Address,
		Gossip:      gossip.Config{TransportClient: b.GossipNet.UnaryClient(), TransportServer: gossipServer},
		Pledge: pledge.Config{
			TransportClient: b.PledgeNet.UnaryClient(),
			TransportServer: pledgeServer,
			Peers:           b.memberAddresses(),
		},
	})
	c, err := cluster.Open(ctx, cfgs...)
	if err != nil {
		return nil, err
	}
	b.ClusterAPIs[c.Host().Key] = c
	return c, nil
}

func (b *Builder) memberAddresses() []address.Address {
	memberAddresses := make([]address.Address, 0, len(b.ClusterAPIs))
	for _, api := range b.ClusterAPIs {
		memberAddresses = append(memberAddresses, api.Host().Address)
	}
	return memberAddresses
}
