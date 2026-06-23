// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	. "github.com/synnaxlabs/x/testutil"
)

func TestWriter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Framer Writer Suite")
}

var (
	dist       mock.Node
	channelSvc *channel.Service
	writerSvc  *writer.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	dist = mock.MustOpenNode(ctx)
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      dist.Channel,
		DB:           dist.DB,
		HostResolver: dist.Cluster,
		Ontology:     dist.Ontology,
		Group:        dist.Group,
		Search:       dist.Search,
	}))
	writerSvc = MustSucceed(writer.NewService(writer.ServiceConfig{
		Framer:  dist.Framer,
		Channel: channelSvc,
	}))
})
