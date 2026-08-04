// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Node", func() {
	Describe("OpenNode", func() {
		It(
			"Should open a single-node cluster bootstrapped as the first node",
			func(ctx SpecContext) {
				n := DeferClose(mock.OpenNode(ctx))
				Expect(n.Cluster.HostKey()).To(Equal(node.KeyBootstrapper))
				Expect(n.Layer).ToNot(BeNil())
				Expect(n.Storage).ToNot(BeNil())
			},
		)

		It(
			"Should tear down the underlying cluster and storage on Close",
			func(ctx SpecContext) {
				n := mock.OpenNode(ctx)
				Expect(n.Close()).To(Succeed())
			},
		)
	})

	Describe("NewNode", func() {
		It(
			"Should open a node and register its teardown with the spec",
			func(ctx SpecContext) {
				n := mock.NewNode(ctx)
				Expect(n.Cluster.HostKey()).To(Equal(node.KeyBootstrapper))
				Expect(n.Storage).ToNot(BeNil())
			},
		)
	})
})
