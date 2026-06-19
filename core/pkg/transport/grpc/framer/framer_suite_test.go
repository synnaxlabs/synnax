// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelmock "github.com/synnaxlabs/synnax/pkg/service/channel/mock"
)

var (
	mockCluster *mock.Cluster
	dist        *distribution.Layer
	chSvc       *channel.Service
)

func TestFramer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transport gRPC Framer Suite")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	mockCluster = mock.NewCluster(ctx, 1)
	node := mockCluster.Nodes[1]
	dist = node.Layer
	chSvc = channelmock.ChannelService(node)
})
