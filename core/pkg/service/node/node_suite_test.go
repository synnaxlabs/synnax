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
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestNode(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Node Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// openTestService spins up a fresh ontology against the supplied cluster and returns a
// node Service registered with it. Test fixtures use this instead of the mock cluster's
// pre-registered services so that the test owns the ontology lifecycle and avoids
// duplicate-registration panics.
func openTestService(
	ctx context.Context,
	c cluster.Cluster,
) (*node.Service, *ontology.Ontology) {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	svc := MustSucceed(node.NewService(ctx, node.ServiceConfig{
		Cluster:  c,
		Ontology: otg,
	}))
	return svc, otg
}

var (
	testCluster *testutil.Cluster
	testSvc     *node.Service
	testOtg     *ontology.Ontology
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	testCluster = testutil.NewCluster(ctx, 2)
	testSvc, testOtg = openTestService(ctx, testCluster.Nodes[1].Cluster)
})
