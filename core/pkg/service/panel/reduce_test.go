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
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/union"
)

// Reduce drives an action through its generated constructor and the generated
// dispatcher - the production Writer.Apply path - whereas the Actions specs
// invoke each payload's Handle directly. These specs cover the New*Action
// constructors and every Reduce branch.
var _ = Describe("Reduce", func() {
	It("Should route a Set action through its constructor and handler", func() {
		next := MustSucceed(panel.Reduce(
			panel.Panel{Name: "old", Root: leafNode()},
			panel.NewSetAction(panel.SetPayload{Path: []string{"name"}, Value: "new"}),
		))
		Expect(next.Name).To(Equal("new"))
	})

	It("Should route an InsertTabs action", func() {
		k := uuid.New()
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: leafNode()},
			panel.NewInsertTabsAction(
				panel.InsertTabsPayload{
					Tabs:       []panel.Tab{tab(k)},
					TargetLeaf: new(int32(1)),
				},
			),
		))
		Expect(tabKeys(next.Root)).To(ContainElement(k))
	})

	It("Should route a RemoveTab action", func() {
		k := uuid.New()
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: leafNode(tab(k))},
			panel.NewRemoveTabAction(panel.RemoveTabPayload{Key: k}),
		))
		Expect(tabKeys(next.Root)).To(BeEmpty())
	})

	It("Should route a MoveTab action", func() {
		k1, k2 := uuid.New(), uuid.New()
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: leafNode(tab(k1), tab(k2))},
			panel.NewMoveTabAction(
				panel.MoveTabPayload{Key: k1, TargetLeaf: 1, Index: new(int32(2))},
			),
		))
		Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{k2, k1}))
	})

	It("Should route a SplitTab action", func() {
		k1, k2 := uuid.New(), uuid.New()
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: leafNode(tab(k1), tab(k2))},
			panel.NewSplitTabAction(
				panel.SplitTabPayload{Key: k2, Direction: spatial.DirectionX},
			),
		))
		MustBeOk(asSplit(next.Root))
	})

	It("Should route a ResizeSplit action", func() {
		next := MustSucceed(panel.Reduce(
			panel.Panel{
				Root: splitNode(spatial.DirectionX, 0.5, leafNode(), leafNode()),
			},
			panel.NewResizeSplitAction(panel.ResizeSplitPayload{Split: 1, Size: 0.7}),
		))
		split := MustBeOk(asSplit(next.Root))
		Expect(split.Size).To(Equal(0.7))
	})

	It("Should route a SetResourceTab action", func() {
		k, other := uuid.New(), uuid.New()
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: leafNode(viewTab(k, "selector"))},
			panel.NewSetTabResourceAction(
				panel.SetTabResourcePayload{Key: k, Resource: tabResource(other)},
			),
		))
		leaf := MustBeOk(asLeaf(next.Root))
		Expect(leaf.Tabs[0]).To(Equal(panel.Tab{Variant: panel.ResourceTab{
			Key:      k,
			Resource: tabResource(other),
		}}))
	})

	It("Should route a SetViewTab action", func() {
		k := uuid.New()
		view := panel.View{Type: "docs"}
		next := MustSucceed(panel.Reduce(
			panel.Panel{Root: leafNode(tab(k))},
			panel.NewSetTabViewAction(panel.SetTabViewPayload{Key: k, View: view}),
		))
		leaf := MustBeOk(asLeaf(next.Root))
		Expect(leaf.Tabs[0]).To(Equal(panel.Tab{Variant: panel.ViewTab{
			Key:  k,
			View: view,
		}}))
	})

	It("Should apply a multi-action sequence in order", func() {
		k := uuid.New()
		next := MustSucceed(panel.Reduce(
			panel.Panel{Name: "old", Root: leafNode()},
			panel.NewSetAction(panel.SetPayload{Path: []string{"name"}, Value: "new"}),
			panel.NewInsertTabsAction(
				panel.InsertTabsPayload{
					Tabs:       []panel.Tab{tab(k)},
					TargetLeaf: new(int32(1)),
				},
			),
		))
		Expect(next.Name).To(Equal("new"))
		Expect(tabKeys(next.Root)).To(ContainElement(k))
	})

	It("Should leave state unchanged on an action with no variant", func() {
		p := panel.Panel{Root: leafNode()}
		res, err := panel.Reduce(p, panel.Action{})
		Expect(err).To(MatchError(union.ErrMissingPayload))
		Expect(res).To(Equal(p))
	})

	DescribeTable(
		"Should reject an action that carries no payload",
		func(actionType panel.ActionType) {
			var a panel.Action
			Expect(json.Unmarshal(
				[]byte(`{"type":"`+string(actionType)+`"}`),
				&a,
			)).To(MatchError(SatisfyAll(
				ContainSubstring("missing payload for declared variant"),
				ContainSubstring(string(actionType)),
			)))
		},
		Entry("Set", panel.SetActionType),
		Entry("InsertTabs", panel.InsertTabsActionType),
		Entry("RemoveTab", panel.RemoveTabActionType),
		Entry("MoveTab", panel.MoveTabActionType),
		Entry("SplitTab", panel.SplitTabActionType),
		Entry("ResizeSplit", panel.ResizeSplitActionType),
		Entry("SetResourceTab", panel.SetTabResourceActionType),
		Entry("SetViewTab", panel.SetTabViewActionType),
	)
})
