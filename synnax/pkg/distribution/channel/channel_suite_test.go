// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	ctx = context.Background()
)

func TestChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Channel Suite")
}

func provisionServices() (*mock.CoreBuilder, map[core.NodeKey]channel.Service) {
	var (
		services = make(map[aspen.NodeKey]channel.Service)
		net      = tmock.NewChannelNetwork()
		builder  = mock.NewCoreBuilder(distribution.Config{
			Storage: storage.Config{MemBacked: config.Bool(true)},
		})
		core1 = builder.New()
		core2 = builder.New()
	)
	services[1] = MustSucceed(channel.New(ctx, channel.ServiceConfig{
		HostResolver:     core1.Cluster,
		ClusterDB:        core1.Storage.Gorpify(),
		TSChannel:        core1.Storage.TS,
		Transport:        net.New(core1.Config.AdvertiseAddress),
		IntOverflowCheck: func(count int64) error { return nil },
		GetChannelCount:  func() (int, error) { return 50, nil },
	}))
	services[2] = MustSucceed(channel.New(ctx, channel.ServiceConfig{
		HostResolver:     core2.Cluster,
		ClusterDB:        core2.Storage.Gorpify(),
		TSChannel:        core2.Storage.TS,
		Transport:        net.New(core2.Config.AdvertiseAddress),
		IntOverflowCheck: func(count int64) error { return nil },
		GetChannelCount:  func() (int, error) { return 50, nil },
	}))
	Eventually(func(g Gomega) {
		g.Expect(core1.Cluster.Nodes()).To(HaveLen(2))
	}).Should(Succeed())
	return builder, services

}
