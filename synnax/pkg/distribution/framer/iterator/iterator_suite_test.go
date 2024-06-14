// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

var (
	ctx = context.Background()
)

func TestIterator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IteratorServer Suite")
}

type serviceContainer struct {
	channel channel.Service
	iter    *iterator.Service
}

func provision(n int) (*mock.CoreBuilder, map[core.NodeKey]serviceContainer) {
	var (
		builder    = mock.NewCoreBuilder(core.Config{})
		services   = make(map[core.NodeKey]serviceContainer)
		channelNet = tmock.NewChannelNetwork()
		iterNet    = tmock.NewIteratorNetwork()
	)
	for i := 0; i < n; i++ {
		var (
			c    = builder.New()
			cont serviceContainer
		)
		cont.channel = MustSucceed(channel.New(ctx, channel.ServiceConfig{
			HostResolver:     c.Cluster,
			ClusterDB:        c.Storage.Gorpify(),
			Transport:        channelNet.New(c.Config.AdvertiseAddress),
			TSChannel:        c.Storage.TS,
			IntOverflowCheck: func(ctx context.Context, count types.Uint20) error { return nil },
		}))
		cont.iter = MustSucceed(iterator.OpenService(iterator.ServiceConfig{
			TS:            c.Storage.TS,
			ChannelReader: cont.channel,
			HostResolver:  c.Cluster,
			Transport:     iterNet.New(c.Config.AdvertiseAddress),
		}))
		services[c.Cluster.HostKey()] = cont
	}
	return builder, services
}
