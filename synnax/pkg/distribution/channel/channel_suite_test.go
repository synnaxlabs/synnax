// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

var ctx = context.Background()

func TestChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Channel Suite")
}

func provisionServices() (*mock.CoreBuilder, map[core.NodeKey]channel.Service, int) {
	var (
		services = make(map[aspen.NodeKey]channel.Service)
		net      = tmock.NewChannelNetwork()
		builder  = mock.NewCoreBuilder(core.Config{
			Storage: storage.Config{MemBacked: config.Bool(true)},
		})
		builder2 = mock.NewCoreBuilder(core.Config{
			Storage: storage.Config{MemBacked: config.Bool(true)},
		})
		core1 = builder.New(ctx)
		core2 = builder.New(ctx)
		core3 = builder2.New(ctx)
		limit = 5
	)
	otg1 := MustSucceed(ontology.Open(ctx, ontology.Config{
		DB: core1.Storage.Gorpify(),
	}))
	builder.AttachCloser(otg1)
	g1 := MustSucceed(group.OpenService(ctx, group.Config{
		DB:       core1.Storage.Gorpify(),
		Ontology: otg1,
	}))
	builder.AttachCloser(g1)
	otg2 := MustSucceed(ontology.Open(ctx, ontology.Config{
		DB: core2.Storage.Gorpify(),
	}))
	builder.AttachCloser(otg2)
	g2 := MustSucceed(group.OpenService(ctx, group.Config{
		DB:       core2.Storage.Gorpify(),
		Ontology: otg2,
	}))
	builder.AttachCloser(g2)
	otg3 := MustSucceed(ontology.Open(ctx, ontology.Config{
		DB: core3.Storage.Gorpify(),
	}))
	builder2.AttachCloser(otg3)
	g3 := MustSucceed(group.OpenService(ctx, group.Config{
		DB:       core3.Storage.Gorpify(),
		Ontology: otg3,
	}))
	builder2.AttachCloser(g3)
	services[1] = MustSucceed(channel.New(ctx, channel.ServiceConfig{
		HostResolver:     core1.Cluster,
		ClusterDB:        core1.Storage.Gorpify(),
		TSChannel:        core1.Storage.TS,
		Transport:        net.New(core1.Config.AdvertiseAddress),
		IntOverflowCheck: func(ctx context.Context, count types.Uint20) error { return nil },
		Ontology:         otg1,
		Group:            g1,
	}))
	services[2] = MustSucceed(channel.New(ctx, channel.ServiceConfig{
		HostResolver:     core2.Cluster,
		ClusterDB:        core2.Storage.Gorpify(),
		TSChannel:        core2.Storage.TS,
		Transport:        net.New(core2.Config.AdvertiseAddress),
		IntOverflowCheck: func(ctx context.Context, count types.Uint20) error { return nil },
		Ontology:         otg2,
		Group:            g2,
	}))
	services[3] = MustSucceed(channel.New(ctx, channel.ServiceConfig{
		HostResolver: core3.Cluster,
		ClusterDB:    core3.Storage.Gorpify(),
		TSChannel:    core3.Storage.TS,
		Transport:    net.New(core3.Config.AdvertiseAddress),
		IntOverflowCheck: func(ctx context.Context, count types.Uint20) error {
			if count > types.Uint20(limit) {
				return errors.New("channel limit exceeded")
			}
			return nil
		},
		Ontology: otg3,
		Group:    g3,
	}))
	Eventually(func(g Gomega) {
		g.Expect(core1.Cluster.Nodes()).To(HaveLen(2))
		g.Expect(core2.Cluster.Nodes()).To(HaveLen(2))
		g.Expect(core3.Cluster.Nodes()).To(HaveLen(1))
	}).Should(Succeed())
	return builder, services, limit

}
