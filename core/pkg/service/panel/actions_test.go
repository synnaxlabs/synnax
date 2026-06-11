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

// tab constructs a resource Tab with a fixed UUID and a placeholder resource. Tests
// use the UUID directly to assert on tab identity.
func tab(key uuid.UUID) panel.Tab {
	return panel.Tab{Variant: panel.TabResource{ResourceTab: panel.ResourceTab{
		Key:      key,
		Resource: ontology.ID{Type: ontology.ResourceTypeLineplot, Key: key.String()},
	}}}
}

// viewTab constructs a view Tab with a fixed UUID and an inline view of the given type.
func viewTab(key uuid.UUID, viewType string) panel.Tab {
	return panel.Tab{Variant: panel.TabView{ViewTab: panel.ViewTab{
		Key:  key,
		View: panel.View{Type: viewType},
	}}}
}

// leafNode wraps a tab list as a leaf node.
func leafNode(tabs ...panel.Tab) panel.Node {
	return panel.Node{Variant: panel.NodeLeaf{
		Leaf: panel.Leaf{Tabs: append([]panel.Tab{}, tabs...)},
	}}
}

// splitNode wraps two child nodes as a split node.
func splitNode(dir spatial.Direction, size float64, first, last panel.Node) panel.Node {
	return panel.Node{Variant: panel.NodeSplit{Split: panel.Split{
		Direction: dir,
		Size:      size,
		First:     first,
		Last:      last,
	}}}
}

// asLeaf returns the leaf variant of n and whether n is a leaf.
func asLeaf(n panel.Node) (panel.Leaf, bool) {
	v, ok := n.Variant.(panel.NodeLeaf)
	return v.Leaf, ok
}

// asSplit returns the split variant of n and whether n is a split.
func asSplit(n panel.Node) (panel.Split, bool) {
	v, ok := n.Variant.(panel.NodeSplit)
	return v.Split, ok
}

// tabKeys returns the keys of the tabs in n's leaf, in order, or nil when n is
// not a leaf.
func tabKeys(n panel.Node) []uuid.UUID {
	leaf, ok := asLeaf(n)
	if !ok {
		return nil
	}
	keys := make([]uuid.UUID, len(leaf.Tabs))
	for i, t := range leaf.Tabs {
		keys[i] = t.Key()
	}
	return keys
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
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1, tab2, tab3}))
		})

		It("Should append when index is absent", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:        tab(tab2),
				TargetLeaf: 1,
			}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1, tab2}))
		})

		It("Should split the target leaf and insert into the new sibling when location is present", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:        tab(tab2),
				TargetLeaf: 1,
				Location:   new(spatial.LocationBottom),
			}.Handle(p))
			split := MustBeOk(asSplit(next.Root))
			Expect(split.Direction).To(Equal(spatial.DirectionY))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab1}))
			Expect(tabKeys(split.Last)).To(Equal([]uuid.UUID{tab2}))
		})

		It("Should place the new sibling first for a left location", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:        tab(tab2),
				TargetLeaf: 1,
				Location:   new(spatial.LocationLeft),
			}.Handle(p))
			split := MustBeOk(asSplit(next.Root))
			Expect(split.Direction).To(Equal(spatial.DirectionX))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab2}))
			Expect(tabKeys(split.Last)).To(Equal([]uuid.UUID{tab1}))
		})

		It("Should place the tab directly in the target leaf for a center location", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:        tab(tab2),
				TargetLeaf: 1,
				Location:   new(spatial.LocationCenter),
			}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1, tab2}))
		})

		It("Should degrade an edge insert into an empty leaf to a direct insert", func() {
			p := panel.Panel{Root: leafNode()}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:        tab(tab1),
				TargetLeaf: 1,
				Location:   new(spatial.LocationRight),
			}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1}))
		})

		DescribeTable("Should error on bad inputs",
			func(p panel.Panel, payload panel.InsertTabPayload, expected string) {
				Expect(payload.Handle(p)).Error().To(MatchError(ContainSubstring(expected)))
			},
			Entry("path does not resolve",
				panel.Panel{Root: leafNode()},
				panel.InsertTabPayload{Tab: tab(uuid.New()), TargetLeaf: 7, Index: new(int32(0))},
				"invalid node path",
			),
			Entry("path resolves to a split",
				panel.Panel{Root: splitNode(spatial.DirectionX, 0.5, leafNode(), leafNode())},
				panel.InsertTabPayload{Tab: tab(uuid.New()), TargetLeaf: 1, Index: new(int32(0))},
				"node at path is not a leaf",
			),
			Entry("index exceeds tab count",
				panel.Panel{Root: leafNode()},
				panel.InsertTabPayload{Tab: tab(uuid.New()), TargetLeaf: 1, Index: new(int32(5))},
				"index out of range",
			),
		)
	})

	Describe("RemoveTab", func() {
		It("Should remove the tab without collapsing when the leaf retains tabs", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2))}
			next := MustSucceed(panel.RemoveTabPayload{Key: tab1}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab2}))
		})

		It("Should leave an empty leaf in place when there is no sibling to collapse into", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.RemoveTabPayload{Key: tab1}.Handle(p))
			leaf := MustBeOk(asLeaf(next.Root))
			Expect(leaf.Tabs).To(BeEmpty())
		})

		It("Should collapse the parent split when the leaf empties and the sibling is non-empty", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionX, 0.5,
				leafNode(tab(tab1)),
				leafNode(tab(tab2), tab(tab3)),
			)}
			next := MustSucceed(panel.RemoveTabPayload{Key: tab1}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab2, tab3}))
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
			split := MustBeOk(asSplit(next.Root))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab2}))
		})

		It("Should return ErrTabNotFound when no tab matches the key", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			Expect(panel.RemoveTabPayload{Key: uuid.New()}.Handle(p)).Error().
				To(MatchError(ContainSubstring("tab not found in tree")))
		})
	})

	Describe("MoveTab", func() {
		It("Should move a tab within the same leaf to a new index", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2), tab(tab3))}
			next := MustSucceed(panel.MoveTabPayload{
				Key: tab1, TargetLeaf: 1, Index: new(int32(2)),
			}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab2, tab3, tab1}))
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
			split := MustBeOk(asSplit(next.Root))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab2}))
			Expect(tabKeys(split.Last)).To(Equal([]uuid.UUID{tab1, tab3}))
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
			split := MustBeOk(asSplit(next.Root))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab1}))
			Expect(tabKeys(split.Last)).To(Equal([]uuid.UUID{tab2}))
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
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1, tab2}))
		})

		It("Should split the target leaf and move the tab into the new sibling when location is present", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2))}
			next := MustSucceed(panel.MoveTabPayload{
				Key:        tab2,
				TargetLeaf: 1,
				Location:   new(spatial.LocationRight),
			}.Handle(p))
			split := MustBeOk(asSplit(next.Root))
			Expect(split.Direction).To(Equal(spatial.DirectionX))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab1}))
			Expect(tabKeys(split.Last)).To(Equal([]uuid.UUID{tab2}))
		})

		It("Should no-op when moving a leaf's only tab to an edge of its own leaf", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.MoveTabPayload{
				Key:        tab1,
				TargetLeaf: 1,
				Location:   new(spatial.LocationLeft),
			}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1}))
		})

		It("Should split when the target leaf's only tab is a different tab", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionX, 0.5,
				leafNode(tab(tab1)),
				leafNode(tab(tab2)),
			)}
			next := MustSucceed(panel.MoveTabPayload{
				Key:        tab1,
				TargetLeaf: 3,
				Location:   new(spatial.LocationTop),
			}.Handle(p))
			split := MustBeOk(asSplit(next.Root))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab1}))
			Expect(tabKeys(split.Last)).To(Equal([]uuid.UUID{tab2}))
			Expect(split.Direction).To(Equal(spatial.DirectionY))
		})

		It("Should return ErrTabNotFound when no tab matches the key", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			Expect(panel.MoveTabPayload{Key: uuid.New(), TargetLeaf: 1, Index: new(int32(0))}.Handle(p)).Error().
				To(MatchError(ContainSubstring("tab not found in tree")))
		})

		It("Should place the tab directly in the target leaf for a center location", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionX, 0.5,
				leafNode(tab(tab1)),
				leafNode(tab(tab2)),
			)}
			next := MustSucceed(panel.MoveTabPayload{
				Key:        tab1,
				TargetLeaf: 3,
				Location:   new(spatial.LocationCenter),
			}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab2, tab1}))
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
			split := MustBeOk(asSplit(next.Root))
			Expect(split.Direction).To(Equal(spatial.DirectionX))
			Expect(split.Size).To(Equal(0.4))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab1, tab2}))
			Expect(tabKeys(split.Last)).To(BeEmpty())
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
			outer := MustBeOk(asSplit(next.Root))
			inner := MustBeOk(asSplit(outer.First))
			Expect(tabKeys(inner.First)).To(BeEmpty())
			Expect(tabKeys(inner.Last)).To(Equal([]uuid.UUID{tab1}))
			Expect(tabKeys(outer.Last)).To(Equal([]uuid.UUID{tab2}))
		})

		It("Should default size to 0.5 when absent", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.SplitLeafPayload{
				Leaf:     1,
				Location: spatial.LocationBottom,
			}.Handle(p))
			split := MustBeOk(asSplit(next.Root))
			Expect(split.Size).To(Equal(0.5))
			Expect(split.Direction).To(Equal(spatial.DirectionY))
		})

		DescribeTable("Should error on bad inputs",
			func(p panel.Panel, payload panel.SplitLeafPayload, expected string) {
				Expect(payload.Handle(p)).Error().To(MatchError(ContainSubstring(expected)))
			},
			Entry("path does not resolve",
				panel.Panel{Root: leafNode()},
				panel.SplitLeafPayload{Leaf: 7, Location: spatial.LocationLeft},
				"invalid node path",
			),
			Entry("path resolves to a split",
				panel.Panel{Root: splitNode(spatial.DirectionX, 0.5, leafNode(), leafNode())},
				panel.SplitLeafPayload{Leaf: 1, Location: spatial.LocationLeft},
				"node at path is not a leaf",
			),
			Entry("location does not divide the area",
				panel.Panel{Root: leafNode()},
				panel.SplitLeafPayload{Leaf: 1, Location: spatial.LocationCenter},
				"invalid split location",
			),
			Entry("size above 1",
				panel.Panel{Root: leafNode(tab(uuid.New()))},
				panel.SplitLeafPayload{Leaf: 1, Location: spatial.LocationLeft, Size: new(float64(1.5))},
				"split size must be in [0, 1]",
			),
			Entry("size below 0",
				panel.Panel{Root: leafNode(tab(uuid.New()))},
				panel.SplitLeafPayload{Leaf: 1, Location: spatial.LocationLeft, Size: new(float64(-0.1))},
				"split size must be in [0, 1]",
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
			split := MustBeOk(asSplit(next.Root))
			Expect(split.Size).To(Equal(0.7))
		})

		DescribeTable("Should error on bad inputs",
			func(p panel.Panel, payload panel.ResizeSplitPayload, expected string) {
				Expect(payload.Handle(p)).Error().To(MatchError(ContainSubstring(expected)))
			},
			Entry("path does not resolve",
				panel.Panel{Root: leafNode()},
				panel.ResizeSplitPayload{Split: 7, Size: 0.5},
				"invalid node path",
			),
			Entry("path resolves to a leaf",
				panel.Panel{Root: leafNode(tab(uuid.New()))},
				panel.ResizeSplitPayload{Split: 1, Size: 0.5},
				"node at path is not a split",
			),
			Entry("size above 1",
				panel.Panel{Root: splitNode(spatial.DirectionX, 0.5, leafNode(), leafNode())},
				panel.ResizeSplitPayload{Split: 1, Size: 1.5},
				"split size must be in [0, 1]",
			),
			Entry("size below 0",
				panel.Panel{Root: splitNode(spatial.DirectionX, 0.5, leafNode(), leafNode())},
				panel.ResizeSplitPayload{Split: 1, Size: -0.1},
				"split size must be in [0, 1]",
			),
		)
	})

	Describe("SetTabResource", func() {
		It("Should set the resource in place without changing identity", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			res := ontology.ID{Type: ontology.ResourceTypeSchematic, Key: tab2.String()}
			next := MustSucceed(panel.SetTabResourcePayload{Key: tab1, Resource: res}.Handle(p))
			leaf := MustBeOk(asLeaf(next.Root))
			Expect(leaf.Tabs[0].Variant).To(Equal(panel.TabResource{
				ResourceTab: panel.ResourceTab{Key: tab1, Resource: res},
			}))
		})

		It("Should replace a view set on the tab", func() {
			p := panel.Panel{Root: leafNode(viewTab(tab1, "docs"))}
			res := ontology.ID{Type: ontology.ResourceTypeSchematic, Key: tab2.String()}
			next := MustSucceed(panel.SetTabResourcePayload{Key: tab1, Resource: res}.Handle(p))
			leaf := MustBeOk(asLeaf(next.Root))
			Expect(leaf.Tabs[0].Variant).To(Equal(panel.TabResource{
				ResourceTab: panel.ResourceTab{Key: tab1, Resource: res},
			}))
		})

		It("Should return ErrTabNotFound when no tab matches the key", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			Expect(panel.SetTabResourcePayload{Key: uuid.New()}.Handle(p)).Error().
				To(MatchError(ContainSubstring("tab not found in tree")))
		})
	})

	Describe("SetTabView", func() {
		It("Should set the view in place without changing identity", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			view := panel.View{Type: "docs"}
			next := MustSucceed(panel.SetTabViewPayload{Key: tab1, View: view}.Handle(p))
			leaf := MustBeOk(asLeaf(next.Root))
			Expect(leaf.Tabs[0].Variant).To(Equal(panel.TabView{
				ViewTab: panel.ViewTab{Key: tab1, View: view},
			}))
		})

		It("Should replace a resource set on the tab", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			view := panel.View{Type: "docs"}
			next := MustSucceed(panel.SetTabViewPayload{Key: tab1, View: view}.Handle(p))
			leaf := MustBeOk(asLeaf(next.Root))
			Expect(leaf.Tabs[0].Variant).To(Equal(panel.TabView{
				ViewTab: panel.ViewTab{Key: tab1, View: view},
			}))
		})

		It("Should return ErrTabNotFound when no tab matches the key", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			Expect(panel.SetTabViewPayload{Key: uuid.New()}.Handle(p)).Error().
				To(MatchError(ContainSubstring("tab not found in tree")))
		})
	})
})
