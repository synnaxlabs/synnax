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

// tab constructs a Tab with a fixed UUID and a placeholder resource. Tests use
// the UUID directly to assert on tab identity.
func tab(key uuid.UUID) panel.Tab {
	return panel.Tab{
		Key:      key,
		Resource: &ontology.ID{Type: ontology.ResourceTypeLineplot, Key: key.String()},
	}
}

// viewTab constructs a Tab with a fixed UUID and an inline view of the given type.
func viewTab(key uuid.UUID, viewType string) panel.Tab {
	return panel.Tab{Key: key, View: &panel.TabView{Type: viewType}}
}

// leafNode wraps a tab list as a leaf node.
func leafNode(tabs ...panel.Tab) panel.Node {
	return panel.Node{Leaf: &panel.Leaf{Tabs: append([]panel.Tab{}, tabs...)}}
}

// splitNode wraps two child nodes as a split node.
func splitNode(dir spatial.Direction, size float64, first, last panel.Node) panel.Node {
	return panel.Node{Split: &panel.Split{
		Direction: dir,
		Size:      size,
		First:     &first,
		Last:      &last,
	}}
}

var _ = Describe("Actions", func() {
	var (
		tab1 uuid.UUID
		tab2 uuid.UUID
		tab3 uuid.UUID
	)
	BeforeEach(func() {
		tab1 = uuid.New()
		tab2 = uuid.New()
		tab3 = uuid.New()
	})

	Describe("Rename", func() {
		It("Should replace the panel's name", func() {
			p := panel.Panel{Name: "Before", Root: leafNode()}
			next := MustSucceed(panel.RenamePayload{Name: "After"}.Handle(p))
			Expect(next.Name).To(Equal("After"))
		})
	})

	Describe("InsertTab", func() {
		It("Should insert a tab into the root leaf at the given index", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab3))}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:        tab(tab2),
				TargetLeaf: 1,
				Index:      new(int32(1)),
			}.Handle(p))
			tabs := next.Root.Leaf.Tabs
			Expect(tabs).To(HaveLen(3))
			Expect(tabs[0].Key).To(Equal(tab1))
			Expect(tabs[1].Key).To(Equal(tab2))
			Expect(tabs[2].Key).To(Equal(tab3))
		})

		It("Should append when index is absent", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:        tab(tab2),
				TargetLeaf: 1,
			}.Handle(p))
			Expect(next.Root.Leaf.Tabs).To(HaveLen(2))
			Expect(next.Root.Leaf.Tabs[1].Key).To(Equal(tab2))
		})

		DescribeTable("Should error on bad inputs",
			func(p panel.Panel, payload panel.InsertTabPayload, expected error) {
				Expect(payload.Handle(p)).Error().To(MatchError(expected))
			},
			Entry("path does not resolve",
				panel.Panel{Root: leafNode()},
				panel.InsertTabPayload{Tab: panel.Tab{Key: uuid.New()}, TargetLeaf: 7, Index: new(int32(0))},
				panel.ErrInvalidPath,
			),
			Entry("path resolves to a split",
				panel.Panel{Root: splitNode(spatial.DirectionX, 0.5, leafNode(), leafNode())},
				panel.InsertTabPayload{Tab: panel.Tab{Key: uuid.New()}, TargetLeaf: 1, Index: new(int32(0))},
				panel.ErrNotALeaf,
			),
			Entry("index exceeds tab count",
				panel.Panel{Root: leafNode()},
				panel.InsertTabPayload{Tab: panel.Tab{Key: uuid.New()}, TargetLeaf: 1, Index: new(int32(5))},
				panel.ErrIndexOutOfRange,
			),
		)
	})

	Describe("RemoveTab", func() {
		It("Should remove the tab without collapsing when the leaf retains tabs", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2))}
			next := MustSucceed(panel.RemoveTabPayload{Key: tab1}.Handle(p))
			Expect(next.Root.Leaf.Tabs).To(HaveLen(1))
			Expect(next.Root.Leaf.Tabs[0].Key).To(Equal(tab2))
		})

		It("Should leave an empty leaf in place when there is no sibling to collapse into", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.RemoveTabPayload{Key: tab1}.Handle(p))
			Expect(next.Root.Leaf).ToNot(BeNil())
			Expect(next.Root.Leaf.Tabs).To(BeEmpty())
		})

		It("Should collapse the parent split when the leaf empties and the sibling is non-empty", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionX, 0.5,
				leafNode(tab(tab1)),
				leafNode(tab(tab2), tab(tab3)),
			)}
			next := MustSucceed(panel.RemoveTabPayload{Key: tab1}.Handle(p))
			Expect(next.Root.Split).To(BeNil())
			Expect(next.Root.Leaf).ToNot(BeNil())
			Expect(next.Root.Leaf.Tabs).To(HaveLen(2))
			Expect(next.Root.Leaf.Tabs[0].Key).To(Equal(tab2))
		})

		It("Should collapse a nested split toward the surviving sibling", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionY, 0.5,
				splitNode(
					spatial.DirectionX, 0.5,
					leafNode(tab(tab1)),
					leafNode(tab(tab2)),
				),
				leafNode(tab(tab3)),
			)}
			next := MustSucceed(panel.RemoveTabPayload{Key: tab1}.Handle(p))
			Expect(next.Root.Split).ToNot(BeNil())
			Expect(next.Root.Split.First.Leaf).ToNot(BeNil())
			Expect(next.Root.Split.First.Leaf.Tabs).To(HaveLen(1))
			Expect(next.Root.Split.First.Leaf.Tabs[0].Key).To(Equal(tab2))
		})

		It("Should return ErrTabNotFound when no tab matches the key", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			Expect(panel.RemoveTabPayload{Key: uuid.New()}.Handle(p)).Error().
				To(MatchError(panel.ErrTabNotFound))
		})
	})

	Describe("MoveTab", func() {
		It("Should move a tab within the same leaf to a new index", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2), tab(tab3))}
			next := MustSucceed(panel.MoveTabPayload{
				Key: tab1, TargetLeaf: 1, Index: new(int32(2)),
			}.Handle(p))
			Expect(next.Root.Leaf.Tabs).To(HaveLen(3))
			Expect(next.Root.Leaf.Tabs[0].Key).To(Equal(tab2))
			Expect(next.Root.Leaf.Tabs[1].Key).To(Equal(tab3))
			Expect(next.Root.Leaf.Tabs[2].Key).To(Equal(tab1))
		})

		It("Should move a tab across leaves of the same split", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionX, 0.5,
				leafNode(tab(tab1), tab(tab2)),
				leafNode(tab(tab3)),
			)}
			next := MustSucceed(panel.MoveTabPayload{
				Key: tab1, TargetLeaf: 3, Index: new(int32(0)),
			}.Handle(p))
			Expect(next.Root.Split.First.Leaf.Tabs).To(HaveLen(1))
			Expect(next.Root.Split.First.Leaf.Tabs[0].Key).To(Equal(tab2))
			Expect(next.Root.Split.Last.Leaf.Tabs).To(HaveLen(2))
			Expect(next.Root.Split.Last.Leaf.Tabs[0].Key).To(Equal(tab1))
			Expect(next.Root.Split.Last.Leaf.Tabs[1].Key).To(Equal(tab3))
		})

		It("Should move a tab into the empty side of a freshly split leaf", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2))}
			p = MustSucceed(panel.SplitLeafPayload{
				Leaf:     1,
				Location: spatial.LocationRight,
				Size:     new(float64(0.5)),
			}.Handle(p))
			next := MustSucceed(panel.MoveTabPayload{
				Key: tab2, TargetLeaf: 3, Index: new(int32(0)),
			}.Handle(p))
			Expect(next.Root.Split).ToNot(BeNil())
			Expect(next.Root.Split.First.Leaf.Tabs).To(HaveLen(1))
			Expect(next.Root.Split.First.Leaf.Tabs[0].Key).To(Equal(tab1))
			Expect(next.Root.Split.Last.Leaf.Tabs).To(HaveLen(1))
			Expect(next.Root.Split.Last.Leaf.Tabs[0].Key).To(Equal(tab2))
		})

		It("Should collapse the source split when moving the last tab out of a side", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionX, 0.5,
				leafNode(tab(tab1)),
				leafNode(tab(tab2)),
			)}
			next := MustSucceed(panel.MoveTabPayload{
				Key: tab1, TargetLeaf: 3, Index: new(int32(0)),
			}.Handle(p))
			Expect(next.Root.Split).To(BeNil())
			Expect(next.Root.Leaf).ToNot(BeNil())
			Expect(next.Root.Leaf.Tabs).To(HaveLen(2))
			Expect(next.Root.Leaf.Tabs[0].Key).To(Equal(tab1))
			Expect(next.Root.Leaf.Tabs[1].Key).To(Equal(tab2))
		})

		It("Should return ErrTabNotFound when no tab matches the key", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			Expect(panel.MoveTabPayload{Key: uuid.New(), TargetLeaf: 1, Index: new(int32(0))}.Handle(p)).Error().
				To(MatchError(panel.ErrTabNotFound))
		})
	})

	Describe("SplitLeaf", func() {
		It("Should split the root leaf into a split with the new empty leaf on the chosen side", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2))}
			next := MustSucceed(panel.SplitLeafPayload{
				Leaf:     1,
				Location: spatial.LocationRight,
				Size:     new(float64(0.4)),
			}.Handle(p))
			Expect(next.Root.Split).ToNot(BeNil())
			Expect(next.Root.Split.Direction).To(Equal(spatial.DirectionX))
			Expect(next.Root.Split.Size).To(Equal(0.4))
			Expect(next.Root.Split.First.Leaf).ToNot(BeNil())
			Expect(next.Root.Split.First.Leaf.Tabs).To(HaveLen(2))
			Expect(next.Root.Split.Last.Leaf).ToNot(BeNil())
			Expect(next.Root.Split.Last.Leaf.Tabs).To(BeEmpty())
		})

		It("Should split a nested leaf without disturbing siblings", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionY, 0.5,
				leafNode(tab(tab1)),
				leafNode(tab(tab2)),
			)}
			next := MustSucceed(panel.SplitLeafPayload{
				Leaf:     2,
				Location: spatial.LocationLeft,
				Size:     new(float64(0.5)),
			}.Handle(p))
			Expect(next.Root.Split.First.Split).ToNot(BeNil())
			Expect(next.Root.Split.First.Split.First.Leaf.Tabs).To(BeEmpty())
			Expect(next.Root.Split.First.Split.Last.Leaf.Tabs).To(HaveLen(1))
			Expect(next.Root.Split.Last.Leaf.Tabs).To(HaveLen(1))
		})

		It("Should default size to 0.5 when absent", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.SplitLeafPayload{
				Leaf:     1,
				Location: spatial.LocationBottom,
			}.Handle(p))
			Expect(next.Root.Split.Size).To(Equal(0.5))
			Expect(next.Root.Split.Direction).To(Equal(spatial.DirectionY))
		})

		DescribeTable("Should error on bad inputs",
			func(p panel.Panel, payload panel.SplitLeafPayload, expected error) {
				Expect(payload.Handle(p)).Error().To(MatchError(expected))
			},
			Entry("path does not resolve",
				panel.Panel{Root: leafNode()},
				panel.SplitLeafPayload{Leaf: 7, Location: spatial.LocationLeft},
				panel.ErrInvalidPath,
			),
			Entry("path resolves to a split",
				panel.Panel{Root: splitNode(spatial.DirectionX, 0.5, leafNode(), leafNode())},
				panel.SplitLeafPayload{Leaf: 1, Location: spatial.LocationLeft},
				panel.ErrNotALeaf,
			),
			Entry("location does not divide the area",
				panel.Panel{Root: leafNode()},
				panel.SplitLeafPayload{Leaf: 1, Location: spatial.LocationCenter},
				panel.ErrInvalidSplitLocation,
			),
		)
	})

	Describe("ResizeSplit", func() {
		It("Should update the size ratio of the split at the given path", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionX, 0.3,
				leafNode(tab(tab1)),
				leafNode(tab(tab2)),
			)}
			next := MustSucceed(panel.ResizeSplitPayload{Split: 1, Size: 0.7}.Handle(p))
			Expect(next.Root.Split.Size).To(Equal(0.7))
		})

		DescribeTable("Should error on bad inputs",
			func(p panel.Panel, payload panel.ResizeSplitPayload, expected error) {
				Expect(payload.Handle(p)).Error().To(MatchError(expected))
			},
			Entry("path does not resolve",
				panel.Panel{Root: leafNode()},
				panel.ResizeSplitPayload{Split: 7, Size: 0.5},
				panel.ErrInvalidPath,
			),
			Entry("path resolves to a leaf",
				panel.Panel{Root: leafNode(tab(uuid.New()))},
				panel.ResizeSplitPayload{Split: 1, Size: 0.5},
				panel.ErrNotASplit,
			),
		)
	})

	Describe("SetTabResource", func() {
		It("Should set the resource in place without changing identity", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			res := ontology.ID{Type: ontology.ResourceTypeSchematic, Key: tab2.String()}
			next := MustSucceed(panel.SetTabResourcePayload{Key: tab1, Resource: res}.Handle(p))
			tabs := next.Root.Leaf.Tabs
			Expect(tabs).To(HaveLen(1))
			Expect(tabs[0].Key).To(Equal(tab1))
			Expect(tabs[0].Resource).To(Equal(&res))
		})

		It("Should clear any view set on the tab", func() {
			p := panel.Panel{Root: leafNode(viewTab(tab1, "docs"))}
			res := ontology.ID{Type: ontology.ResourceTypeSchematic, Key: tab2.String()}
			next := MustSucceed(panel.SetTabResourcePayload{Key: tab1, Resource: res}.Handle(p))
			Expect(next.Root.Leaf.Tabs[0].Resource).To(Equal(&res))
			Expect(next.Root.Leaf.Tabs[0].View).To(BeNil())
		})

		It("Should return ErrTabNotFound when no tab matches the key", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			Expect(panel.SetTabResourcePayload{Key: uuid.New()}.Handle(p)).Error().
				To(MatchError(panel.ErrTabNotFound))
		})
	})

	Describe("SetTabView", func() {
		It("Should set the view in place without changing identity", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			view := panel.TabView{Type: "docs"}
			next := MustSucceed(panel.SetTabViewPayload{Key: tab1, View: view}.Handle(p))
			tabs := next.Root.Leaf.Tabs
			Expect(tabs).To(HaveLen(1))
			Expect(tabs[0].Key).To(Equal(tab1))
			Expect(tabs[0].View).To(Equal(&view))
		})

		It("Should clear any resource set on the tab", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.SetTabViewPayload{
				Key: tab1, View: panel.TabView{Type: "docs"},
			}.Handle(p))
			Expect(next.Root.Leaf.Tabs[0].View).ToNot(BeNil())
			Expect(next.Root.Leaf.Tabs[0].Resource).To(BeNil())
		})

		It("Should return ErrTabNotFound when no tab matches the key", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			Expect(panel.SetTabViewPayload{Key: uuid.New()}.Handle(p)).Error().
				To(MatchError(panel.ErrTabNotFound))
		})
	})
})
