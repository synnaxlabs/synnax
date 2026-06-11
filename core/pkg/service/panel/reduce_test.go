// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

// Reduce drives an action through its generated constructor and the generated
// dispatcher - the production Writer.Dispatch path - whereas the Actions specs
// invoke each payload's Handle directly. These specs cover the New*Action
// constructors and every Reduce branch.
var _ = Describe("Reduce", func() {
	It("Should route a Rename action through its constructor and handler", func() {
		next := MustSucceed(panel.Reduce(
			panel.Panel{Name: "old", Root: leafNode()},
			panel.NewRenameAction(panel.RenamePayload{Name: "new"}),
		))
		Expect(next.Name).To(Equal("new"))
	})

	It("Should route an InsertTab action", func() {
		k := uuid.New()
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: leafNode()},
			panel.NewInsertTabAction(panel.InsertTabPayload{Tab: tab(k), TargetLeaf: 1}),
		))
		Expect(tabKeys(&next.Root)).To(ContainElement(k))
	})

	It("Should route a RemoveTab action", func() {
		k := uuid.New()
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: leafNode(tab(k))},
			panel.NewRemoveTabAction(panel.RemoveTabPayload{Key: k}),
		))
		Expect(next.Root.Leaf.Tabs).To(BeEmpty())
	})

	It("Should route a MoveTab action", func() {
		k1, k2 := uuid.New(), uuid.New()
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: leafNode(tab(k1), tab(k2))},
			panel.NewMoveTabAction(panel.MoveTabPayload{Key: k1, TargetLeaf: 1, Index: new(int32(1))}),
		))
		Expect(tabKeys(&next.Root)).To(Equal([]uuid.UUID{k2, k1}))
	})

	It("Should route a SplitLeaf action", func() {
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: leafNode()},
			panel.NewSplitLeafAction(panel.SplitLeafPayload{Leaf: 1, Location: spatial.LocationLeft}),
		))
		Expect(next.Root.Split).ToNot(BeNil())
	})

	It("Should route a ResizeSplit action", func() {
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: splitNode(spatial.DirectionX, 0.5, leafNode(), leafNode())},
			panel.NewResizeSplitAction(panel.ResizeSplitPayload{Split: 1, Size: 0.7}),
		))
		Expect(next.Root.Split.Size).To(Equal(0.7))
	})

	It("Should route a SetTabResource action", func() {
		k := uuid.New()
		res := ontology.ID{Type: ontology.ResourceTypeSchematic, Key: uuid.New().String()}
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: leafNode(tab(k))},
			panel.NewSetTabResourceAction(panel.SetTabResourcePayload{Key: k, Resource: res}),
		))
		Expect(next.Root.Leaf.Tabs[0].Resource).To(Equal(&res))
	})

	It("Should route a SetTabView action", func() {
		k := uuid.New()
		view := panel.TabView{Type: "docs"}
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: leafNode(tab(k))},
			panel.NewSetTabViewAction(panel.SetTabViewPayload{Key: k, View: view}),
		))
		Expect(next.Root.Leaf.Tabs[0].View).To(Equal(&view))
	})

	It("Should apply a multi-action sequence in order", func() {
		k := uuid.New()
		next := MustSucceed(panel.Reduce(
			panel.Panel{Name: "old", Root: leafNode()},
			panel.NewRenameAction(panel.RenamePayload{Name: "new"}),
			panel.NewInsertTabAction(panel.InsertTabPayload{Tab: tab(k), TargetLeaf: 1}),
		))
		Expect(next.Name).To(Equal("new"))
		Expect(tabKeys(&next.Root)).To(ContainElement(k))
	})

	DescribeTable("Should return a missing-payload error and leave state unchanged when an action carries no payload",
		func(actionType string) {
			p := panel.Panel{Root: leafNode()}
			res, err := panel.Reduce(p, panel.Action{Type: actionType})
			Expect(err).To(MatchError(SatisfyAll(
				ContainSubstring("missing payload for declared variant"),
				ContainSubstring(actionType),
			)))
			Expect(res).To(Equal(p))
		},
		Entry("Rename", panel.ActionTypeRename),
		Entry("InsertTab", panel.ActionTypeInsertTab),
		Entry("RemoveTab", panel.ActionTypeRemoveTab),
		Entry("MoveTab", panel.ActionTypeMoveTab),
		Entry("SplitLeaf", panel.ActionTypeSplitLeaf),
		Entry("ResizeSplit", panel.ActionTypeResizeSplit),
		Entry("SetTabResource", panel.ActionTypeSetTabResource),
		Entry("SetTabView", panel.ActionTypeSetTabView),
	)
})
