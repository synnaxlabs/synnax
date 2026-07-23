// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/arc/ir/types/v1"
	v2 "github.com/synnaxlabs/arc/ir/types/v2"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateFunction", func() {
	It("Should carry a Function's fields to the next version", func(ctx SpecContext) {
		migrated := MustSucceed(v2.MigrateFunction(ctx, v1.Function{
			Key:  "scale",
			Body: v1.Body{Raw: "x * 2"},
		}))
		Expect(migrated.Key).To(Equal("scale"))
		Expect(migrated.Body.Raw).To(Equal("x * 2"))
	})
})

var _ = Describe("MigrateIR", func() {
	It("Should carry an IR's functions, nodes, edges, and root", func(ctx SpecContext) {
		migrated := MustSucceed(v2.MigrateIR(ctx, v1.IR{
			Functions: v1.Functions{{Key: "f"}},
			Nodes:     v1.Nodes{{Key: "n", Type: "add"}},
			Edges:     v1.Edges{{Kind: v1.EdgeKindContinuous}},
			Root:      v1.Scope{Key: "root"},
		}))
		Expect(migrated.Functions).To(HaveLen(1))
		Expect(migrated.Functions[0].Key).To(Equal("f"))
		Expect(migrated.Nodes).To(HaveLen(1))
		Expect(migrated.Nodes[0].Key).To(Equal("n"))
		Expect(migrated.Nodes[0].Type).To(Equal("add"))
		Expect(migrated.Edges).To(HaveLen(1))
		Expect(migrated.Root.Key).To(Equal("root"))
	})
})
