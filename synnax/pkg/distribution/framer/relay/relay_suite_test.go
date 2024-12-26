// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay_test

import (
	"context"
	"testing"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	ctx = context.Background()
	ins alamos.Instrumentation
)

func TestRelay(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Relay Suite")
}

type serviceContainer struct {
	channel   channel.Service
	writer    *writer.Service
	relay     *relay.Relay
	transport struct {
		channel channel.Transport
		writer  writer.Transport
		relay   relay.Transport
	}
}

func provision(n int) (*mock.CoreBuilder, map[core.NodeKey]serviceContainer) {
	var (
		builder    = mock.NewCoreBuilder()
		service    = make(map[core.NodeKey]serviceContainer)
		channelNet = tmock.NewChannelNetwork()
		writerNet  = tmock.NewWriterNetwork()
		relayNet   = tmock.NewRelayNetwork()
	)
	for i := 0; i < n; i++ {
		var (
			c         = builder.New()
			container serviceContainer
		)
		container.channel = MustSucceed(channel.New(ctx, channel.ServiceConfig{
			HostResolver:     c.Cluster,
			ClusterDB:        c.Storage.Gorpify(),
			TSChannel:        c.Storage.TS,
			Transport:        channelNet.New(c.Config.AdvertiseAddress),
			IntOverflowCheck: func(ctx context.Context, count types.Uint20) error { return nil },
		}))
		freeWrites := confluence.NewStream[relay.Response](25)
		container.relay = MustSucceed(relay.Open(relay.Config{
			Instrumentation: ins,
			TS:              c.Storage.TS,
			Transport:       relayNet.New(c.Config.AdvertiseAddress),
			HostResolver:    c.Cluster,
			ChannelReader:   container.channel,
			FreeWrites:      freeWrites,
		}))
		container.writer = MustSucceed(writer.OpenService(writer.ServiceConfig{
			Instrumentation: ins,
			TS:              c.Storage.TS,
			ChannelReader:   container.channel,
			Transport:       writerNet.New(c.Config.AdvertiseAddress),
			HostResolver:    c.Cluster,
			FreeWrites:      freeWrites,
		}))
		service[c.Cluster.HostKey()] = container
	}
	builder.WaitForTopologyToStabilize()
	return builder, service
}
