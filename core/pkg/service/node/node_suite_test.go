// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	testCluster *testutil.Cluster
	testSvc     *node.Service
)

func TestNode(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Node Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	testCluster = testutil.NewCluster(ctx, 2)
	testSvc = MustSucceed(node.NewService(node.ServiceConfig{
		Cluster: testCluster.Nodes[1].Cluster,
	}))
})
