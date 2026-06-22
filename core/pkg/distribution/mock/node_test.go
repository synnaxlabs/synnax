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
	ShouldNotLeakGoroutinesPerSpec()
	It("Should open a single-node cluster whose host is the bootstrapper", func(ctx SpecContext) {
		n := mock.OpenNode(ctx)
		Expect(n.Cluster.HostKey()).To(Equal(node.KeyBootstrapper))
		Expect(n.Close()).To(Succeed())
	})
})
