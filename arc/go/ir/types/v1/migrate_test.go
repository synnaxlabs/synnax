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
	v0 "github.com/synnaxlabs/arc/ir/types/v0"
	v1 "github.com/synnaxlabs/arc/ir/types/v1"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migrate", func() {
	It("Should carry a Function's fields to the next version", func(ctx SpecContext) {
		migrated := MustSucceed(v1.MigrateFunction(ctx, v0.Function{
			Key:  "scale",
			Body: v0.Body{Raw: "x * 2"},
		}))
		Expect(migrated.Key).To(Equal("scale"))
		Expect(migrated.Body.Raw).To(Equal("x * 2"))
	})
	It("Should carry an Edge's endpoints and kind", func(ctx SpecContext) {
		migrated := MustSucceed(v1.MigrateEdge(ctx, v0.Edge{
			Source: v0.Handle{Node: "a", Param: "out"},
			Target: v0.Handle{Node: "b", Param: "in"},
			Kind:   v0.EdgeKindContinuous,
		}))
		Expect(migrated.Source).To(Equal(v1.Handle{Node: "a", Param: "out"}))
		Expect(migrated.Target).To(Equal(v1.Handle{Node: "b", Param: "in"}))
		Expect(migrated.Kind).To(Equal(v1.EdgeKindContinuous))
	})
	It("Should carry a Node's key and type", func(ctx SpecContext) {
		migrated := MustSucceed(v1.MigrateNode(ctx, v0.Node{Key: "n1", Type: "add"}))
		Expect(migrated.Key).To(Equal("n1"))
		Expect(migrated.Type).To(Equal("add"))
	})
	It("Should carry an IR's functions, nodes, and edges", func(ctx SpecContext) {
		migrated := MustSucceed(v1.MigrateIR(ctx, v0.IR{
			Functions: v0.Functions{{Key: "f"}},
			Nodes:     v0.Nodes{{Key: "n"}},
			Edges:     v0.Edges{{Kind: v0.EdgeKindContinuous}},
		}))
		Expect(migrated.Functions).To(HaveLen(1))
		Expect(migrated.Functions[0].Key).To(Equal("f"))
		Expect(migrated.Nodes).To(HaveLen(1))
		Expect(migrated.Edges).To(HaveLen(1))
	})
	It("Should carry Authorities' default and channel overrides", func(ctx SpecContext) {
		def := uint8(5)
		migrated := MustSucceed(v1.MigrateAuthorities(ctx, v0.Authorities{
			Default:  &def,
			Channels: map[uint32]uint8{1: 2},
		}))
		Expect(*migrated.Default).To(Equal(uint8(5)))
		Expect(migrated.Channels).To(HaveKeyWithValue(uint32(1), uint8(2)))
	})
})
