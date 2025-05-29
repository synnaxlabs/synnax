// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/x/confluence"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

var (
	ctx = context.Background()
	ins alamos.Instrumentation
)

func TestWriter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Writer Suite")
}

type serviceContainer struct {
	channel   channel.Service
	writer    *writer.Service
	transport struct {
		channel channel.Transport
		writer  writer.Transport
	}
}

func provision(n int) (*mock.CoreBuilder, map[core.NodeKey]serviceContainer) {
	var (
		builder    = mock.NewCoreBuilder()
		services   = make(map[core.NodeKey]serviceContainer)
		channelNet = tmock.NewChannelNetwork()
		writerNet  = tmock.NewWriterNetwork()
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
		container.writer = MustSucceed(writer.OpenService(writer.ServiceConfig{
			Instrumentation: ins,
			TS:              c.Storage.TS,
			ChannelReader:   container.channel,
			HostResolver:    c.Cluster,
			Transport:       writerNet.New(c.Config.AdvertiseAddress /*buffer*/, 10),
			FreeWrites:      confluence.NewStream[relay.Response](1000),
		}))
		services[c.Cluster.HostKey()] = container
	}
	builder.WaitForTopologyToStabilize()
	return builder, services
}
