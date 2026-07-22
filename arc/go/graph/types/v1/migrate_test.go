// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/arc/graph/types/v0"
	v1 "github.com/synnaxlabs/arc/graph/types/v1"
	irv0 "github.com/synnaxlabs/arc/ir/types/v0"
	irv1 "github.com/synnaxlabs/arc/ir/types/v1"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateGraph", func() {
	It("Should carry the viewport, functions, nodes, and edges", func(ctx SpecContext) {
		migrated := MustSucceed(v1.MigrateGraph(ctx, v0.Graph{
			Viewport:  v0.Viewport{Zoom: 2},
			Functions: irv0.Functions{{Key: "f"}},
			Edges:     irv0.Edges{{Kind: irv0.EdgeKindContinuous}},
			Nodes:     v0.Nodes{{Key: "n1", Type: "add"}},
		}))
		Expect(migrated.Viewport.Zoom).To(Equal(2.0))
		Expect(migrated.Functions).To(HaveLen(1))
		Expect(migrated.Functions[0].Key).To(Equal("f"))
		Expect(migrated.Edges).To(HaveLen(1))
		Expect(migrated.Edges[0].Kind).To(Equal(irv1.EdgeKindContinuous))
		Expect(migrated.Nodes).To(HaveLen(1))
		Expect(migrated.Nodes[0].Key).To(Equal("n1"))
	})
})
