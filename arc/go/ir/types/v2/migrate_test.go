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

var _ = Describe("Migrate", func() {
	It("Should carry a Function's fields to the next version", func(ctx SpecContext) {
		migrated := MustSucceed(v2.MigrateFunction(ctx, v1.Function{
			Key:  "scale",
			Body: v1.Body{Raw: "x * 2"},
		}))
		Expect(migrated.Key).To(Equal("scale"))
		Expect(migrated.Body.Raw).To(Equal("x * 2"))
	})
	It("Should carry a Node's key and type", func(ctx SpecContext) {
		migrated := MustSucceed(v2.MigrateNode(ctx, v1.Node{Key: "n1", Type: "add"}))
		Expect(migrated.Key).To(Equal("n1"))
		Expect(migrated.Type).To(Equal("add"))
	})
	It("Should carry an Edge's endpoints and kind", func(ctx SpecContext) {
		migrated := MustSucceed(v2.MigrateEdge(ctx, v1.Edge{
			Source: v1.Handle{Node: "a", Param: "out"},
			Target: v1.Handle{Node: "b", Param: "in"},
			Kind:   v1.EdgeKindContinuous,
		}))
		Expect(migrated.Source).To(Equal(v2.Handle{Node: "a", Param: "out"}))
		Expect(migrated.Target).To(Equal(v2.Handle{Node: "b", Param: "in"}))
		Expect(migrated.Kind).To(Equal(v2.EdgeKindContinuous))
	})
	It("Should carry a Scope's mode, steps, and transitions", func(ctx SpecContext) {
		nodeKey := "n1"
		targetKey := "s2"
		migrated := MustSucceed(v2.MigrateScope(ctx, v1.Scope{
			Key:      "root",
			Mode:     v1.ScopeModeSequential,
			Liveness: v1.LivenessGated,
			Steps:    v1.Members{{NodeKey: &nodeKey}},
			Transitions: []v1.Transition{{
				On:        v1.Handle{Node: "n1", Param: "done"},
				TargetKey: &targetKey,
			}},
		}))
		Expect(migrated.Key).To(Equal("root"))
		Expect(migrated.Mode).To(Equal(v2.ScopeModeSequential))
		Expect(migrated.Liveness).To(Equal(v2.LivenessGated))
		Expect(migrated.Steps).To(HaveLen(1))
		Expect(*migrated.Steps[0].NodeKey).To(Equal("n1"))
		Expect(migrated.Transitions).To(HaveLen(1))
		Expect(*migrated.Transitions[0].TargetKey).To(Equal("s2"))
	})
	It("Should carry a Member's node key and nested scope", func(ctx SpecContext) {
		nodeKey := "n1"
		migrated := MustSucceed(v2.MigrateMember(ctx, v1.Member{
			Scope: &v1.Scope{Key: "nested", Steps: v1.Members{{NodeKey: &nodeKey}}},
		}))
		Expect(migrated.NodeKey).To(BeNil())
		Expect(migrated.Scope.Key).To(Equal("nested"))
		Expect(migrated.Scope.Steps).To(HaveLen(1))
	})
	It("Should carry a Transition's trigger and target", func(ctx SpecContext) {
		targetKey := "s2"
		migrated := MustSucceed(v2.MigrateTransition(ctx, v1.Transition{
			On:        v1.Handle{Node: "n1", Param: "done"},
			TargetKey: &targetKey,
		}))
		Expect(migrated.On).To(Equal(v2.Handle{Node: "n1", Param: "done"}))
		Expect(*migrated.TargetKey).To(Equal("s2"))
	})
	It("Should carry an IR's functions, nodes, edges, and root", func(ctx SpecContext) {
		migrated := MustSucceed(v2.MigrateIR(ctx, v1.IR{
			Functions: v1.Functions{{Key: "f"}},
			Nodes:     v1.Nodes{{Key: "n"}},
			Edges:     v1.Edges{{Kind: v1.EdgeKindContinuous}},
			Root:      v1.Scope{Key: "root"},
		}))
		Expect(migrated.Functions).To(HaveLen(1))
		Expect(migrated.Functions[0].Key).To(Equal("f"))
		Expect(migrated.Nodes).To(HaveLen(1))
		Expect(migrated.Edges).To(HaveLen(1))
		Expect(migrated.Root.Key).To(Equal("root"))
	})
})
