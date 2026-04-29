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
)

var (
	mockCluster *mock.Cluster
	dist        *distribution.Layer
)

func TestFramer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gRPC Framer Suite")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	mockCluster = mock.ProvisionCluster(ctx, 1)
	dist = mockCluster.Nodes[1].Layer
	DeferCleanup(func() {
		Expect(dist.Close()).To(Succeed())
		Expect(mockCluster.Close()).To(Succeed())
	})
})
