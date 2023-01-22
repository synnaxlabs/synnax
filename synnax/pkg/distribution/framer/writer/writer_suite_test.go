// Copyright 2023 Synnax Labs, Inc.
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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"testing"
)

var ctx = context.Background()

func TestWriter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Write Suite")
}

type serviceContainer struct {
	channel   channel.Service
	writer    *writer.Service
	transport struct {
		channel channel.Transport
		writer  writer.Transport
	}
}

func provision(n int, logger *zap.Logger) (*mock.CoreBuilder, map[core.NodeID]serviceContainer) {
	var (
		builder    = mock.NewCoreBuilder(core.Config{Logger: logger})
		services   = make(map[core.NodeID]serviceContainer)
		channelNet = tmock.NewChannelNetwork()
		writerNet  = tmock.NewFramerWriterNetwork()
	)
	for i := 0; i < n; i++ {
		var (
			c         = builder.New()
			container serviceContainer
		)
		container.channel = MustSucceed(channel.New(channel.ServiceConfig{
			HostResolver: c.Cluster,
			ClusterDB:    c.Storage.Gorpify(),
			TSChannel:    c.Storage.TS,
			Transport:    channelNet.New(c.Config.AdvertiseAddress),
		}))
		container.writer = MustSucceed(writer.OpenService(writer.ServiceConfig{
			TS:            c.Storage.TS,
			ChannelReader: container.channel,
			HostResolver:  c.Cluster,
			Transport:     writerNet.New(c.Config.AdvertiseAddress /*buffer*/, 10),
			Logger:        logger,
		}))
		services[c.Cluster.HostID()] = container
	}
	builder.WaitForTopologyToStabilize()
	return builder, services
}
