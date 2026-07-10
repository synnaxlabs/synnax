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

// tabResource is the ontology ID the tab helper derives from a tab key, so each
// tab in a fixture backs a distinct resource.
func tabResource(key uuid.UUID) ontology.ID {
	return ontology.ID{Type: "lineplot", Key: key.String()}
}

// tab constructs a resource Tab with a fixed UUID and a key-derived resource. Tests
// use the UUID directly to assert on tab identity.
func tab(key uuid.UUID) panel.Tab {
	return panel.Tab{Variant: panel.TabResource{
		TabBase:  panel.TabBase{Key: key},
		Resource: tabResource(key),
	}}
}

// viewTab constructs a view Tab with a fixed UUID and an inline view of the given type.
func viewTab(key uuid.UUID, viewType string) panel.Tab {
	return panel.Tab{Variant: panel.TabView{
		TabBase: panel.TabBase{Key: key},
		View:    panel.View{Type: viewType},
	}}
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

// tabByKey returns the tab with the given key found anywhere in n, and whether
// it was found.
func tabByKey(n panel.Node, key uuid.UUID) (panel.Tab, bool) {
	switch v := n.Variant.(type) {
	case panel.NodeLeaf:
		for _, t := range v.Leaf.Tabs {
			if t.Key() == key {
				return t, true
			}
		}
		return panel.Tab{}, false
	case panel.NodeSplit:
		if t, ok := tabByKey(v.Split.First, key); ok {
			return t, true
		}
		return tabByKey(v.Split.Last, key)
	default:
		return panel.Tab{}, false
	}
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
				TargetLeaf: new(int32(1)),
				Index:      new(int32(1)),
			}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1, tab2, tab3}))
		})

		It("Should append when index is absent", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:        tab(tab2),
				TargetLeaf: new(int32(1)),
			}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1, tab2}))
		})

		It("Should split the target leaf and insert into the new sibling when location is present", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:        tab(tab2),
				TargetLeaf: new(int32(1)),
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
				TargetLeaf: new(int32(1)),
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
				TargetLeaf: new(int32(1)),
				Location:   new(spatial.LocationCenter),
			}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1, tab2}))
		})

		It("Should degrade an edge insert into an empty leaf to a direct insert", func() {
			p := panel.Panel{Root: leafNode()}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:        tab(tab1),
				TargetLeaf: new(int32(1)),
				Location:   new(spatial.LocationRight),
			}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1}))
		})

		It("Should insert into the leaf holding TargetTab when set", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionX, 0.5,
				leafNode(tab(tab1)),
				leafNode(tab(tab2)),
			)}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:       tab(tab3),
				TargetTab: &tab2,
			}.Handle(p))
			split := MustBeOk(asSplit(next.Root))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab1}))
			Expect(tabKeys(split.Last)).To(Equal([]uuid.UUID{tab2, tab3}))
		})

		It("Should error when TargetTab is set but no tab matches it", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			missing := uuid.New()
			Expect(panel.InsertTabPayload{Tab: tab(tab2), TargetTab: &missing}.Handle(p)).
				Error().To(MatchError(ContainSubstring("tab not found in tree")))
		})

		It("Should default to the first leaf in traversal order when no target is set", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionX, 0.5,
				leafNode(tab(tab1)),
				leafNode(tab(tab2)),
			)}
			next := MustSucceed(panel.InsertTabPayload{Tab: tab(tab3)}.Handle(p))
			split := MustBeOk(asSplit(next.Root))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab1, tab3}))
			Expect(tabKeys(split.Last)).To(Equal([]uuid.UUID{tab2}))
		})

		It("Should default to the root leaf when no target is set on a single-leaf tree", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.InsertTabPayload{Tab: tab(tab2)}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1, tab2}))
		})

		It("Should refresh an existing tab's content in place when no placement is given", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionX, 0.5,
				leafNode(tab(tab1)),
				leafNode(viewTab(tab2, "selector")),
			)}
			next := MustSucceed(panel.InsertTabPayload{Tab: tab(tab2)}.Handle(p))
			split := MustBeOk(asSplit(next.Root))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab1}))
			Expect(tabKeys(split.Last)).To(Equal([]uuid.UUID{tab2}))
			refreshed := MustBeOk(tabByKey(next.Root, tab2))
			Expect(refreshed.Variant).To(Equal(panel.TabResource{
				TabBase:  panel.TabBase{Key: tab2},
				Resource: tabResource(tab2),
			}))
		})

		It("Should not duplicate an existing tab when no placement is given", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2))}
			next := MustSucceed(panel.InsertTabPayload{Tab: tab(tab1)}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1, tab2}))
		})

		It("Should be a no-op when the resource already backs a different tab", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2))}
			duplicate := panel.Tab{Variant: panel.TabResource{
				TabBase:  panel.TabBase{Key: tab3},
				Resource: tabResource(tab1),
			}}
			next := MustSucceed(panel.InsertTabPayload{Tab: duplicate}.Handle(p))
			Expect(next).To(Equal(p))
		})

		It("Should be a no-op even when the duplicate insert carries a placement", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2))}
			duplicate := panel.Tab{Variant: panel.TabResource{
				TabBase:  panel.TabBase{Key: tab3},
				Resource: tabResource(tab1),
			}}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:      duplicate,
				Location: new(spatial.LocationRight),
			}.Handle(p))
			Expect(next).To(Equal(p))
		})

		It("Should relocate an existing tab and refresh its content when a placement is given", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), viewTab(tab2, "selector"))}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:        tab(tab2),
				TargetLeaf: new(int32(1)),
				Location:   new(spatial.LocationRight),
			}.Handle(p))
			split := MustBeOk(asSplit(next.Root))
			Expect(split.Direction).To(Equal(spatial.DirectionX))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab1}))
			Expect(tabKeys(split.Last)).To(Equal([]uuid.UUID{tab2}))
			refreshed := MustBeOk(tabByKey(next.Root, tab2))
			Expect(refreshed.Variant).To(Equal(panel.TabResource{
				TabBase:  panel.TabBase{Key: tab2},
				Resource: tabResource(tab2),
			}))
		})

		It("Should reorder an existing tab within its leaf when only an index is given", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2), tab(tab3))}
			next := MustSucceed(panel.InsertTabPayload{
				Tab:   tab(tab3),
				Index: new(int32(0)),
			}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab3, tab1, tab2}))
		})

		It("Should move an existing tab into the leaf holding TargetTab without duplicating it", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionX, 0.5,
				leafNode(tab(tab1)),
				leafNode(tab(tab2)),
			)}
			next := MustSucceed(panel.InsertTabPayload{Tab: tab(tab1), TargetTab: &tab2}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab2, tab1}))
		})

		DescribeTable("Should error on bad inputs",
			func(p panel.Panel, payload panel.InsertTabPayload, expected string) {
				Expect(payload.Handle(p)).Error().To(MatchError(ContainSubstring(expected)))
			},
			Entry("path does not resolve",
				panel.Panel{Root: leafNode()},
				panel.InsertTabPayload{Tab: tab(uuid.New()), TargetLeaf: new(int32(7)), Index: new(int32(0))},
				"invalid node path",
			),
			Entry("path resolves to a split",
				panel.Panel{Root: splitNode(spatial.DirectionX, 0.5, leafNode(), leafNode())},
				panel.InsertTabPayload{Tab: tab(uuid.New()), TargetLeaf: new(int32(1)), Index: new(int32(0))},
				"node at path is not a leaf",
			),
			Entry("index exceeds tab count",
				panel.Panel{Root: leafNode()},
				panel.InsertTabPayload{Tab: tab(uuid.New()), TargetLeaf: new(int32(1)), Index: new(int32(5))},
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

	Describe("SplitTab", func() {
		It("Should split the tab off into a new sibling pane to the right for direction x", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2))}
			next := MustSucceed(panel.SplitTabPayload{
				Key:       tab2,
				Direction: spatial.DirectionX,
			}.Handle(p))
			split := MustBeOk(asSplit(next.Root))
			Expect(split.Direction).To(Equal(spatial.DirectionX))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab1}))
			Expect(tabKeys(split.Last)).To(Equal([]uuid.UUID{tab2}))
		})

		It("Should split the tab off into a new sibling pane below for direction y", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2))}
			next := MustSucceed(panel.SplitTabPayload{
				Key:       tab2,
				Direction: spatial.DirectionY,
			}.Handle(p))
			split := MustBeOk(asSplit(next.Root))
			Expect(split.Direction).To(Equal(spatial.DirectionY))
			Expect(tabKeys(split.First)).To(Equal([]uuid.UUID{tab1}))
			Expect(tabKeys(split.Last)).To(Equal([]uuid.UUID{tab2}))
		})

		It("Should resolve the tab's own leaf in a nested tree without disturbing siblings", func() {
			p := panel.Panel{Root: splitNode(
				spatial.DirectionX, 0.5,
				leafNode(tab(tab1), tab(tab2)),
				leafNode(tab(tab3)),
			)}
			next := MustSucceed(panel.SplitTabPayload{
				Key:       tab1,
				Direction: spatial.DirectionX,
			}.Handle(p))
			outer := MustBeOk(asSplit(next.Root))
			inner := MustBeOk(asSplit(outer.First))
			Expect(tabKeys(inner.First)).To(Equal([]uuid.UUID{tab2}))
			Expect(tabKeys(inner.Last)).To(Equal([]uuid.UUID{tab1}))
			Expect(tabKeys(outer.Last)).To(Equal([]uuid.UUID{tab3}))
		})

		It("Should no-op when the tab is the only tab in its leaf", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(panel.SplitTabPayload{
				Key:       tab1,
				Direction: spatial.DirectionX,
			}.Handle(p))
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1}))
		})

		It("Should return ErrTabNotFound when no tab matches the key", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), tab(tab2))}
			Expect(panel.SplitTabPayload{Key: uuid.New(), Direction: spatial.DirectionX}.Handle(p)).Error().
				To(MatchError(ContainSubstring("tab not found in tree")))
		})
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
		It("Should swap a view tab to the resource in place", func() {
			p := panel.Panel{Root: leafNode(viewTab(tab1, "selector"))}
			next := MustSucceed(
				panel.SetTabResourcePayload{Key: tab1, Resource: tabResource(tab2)}.Handle(p),
			)
			leaf := MustBeOk(asLeaf(next.Root))
			Expect(leaf.Tabs[0]).To(Equal(panel.Tab{Variant: panel.TabResource{
				TabBase:  panel.TabBase{Key: tab1},
				Resource: tabResource(tab2),
			}}))
		})

		It("Should replace an existing resource without changing the tab's identity", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			next := MustSucceed(
				panel.SetTabResourcePayload{Key: tab1, Resource: tabResource(tab3)}.Handle(p),
			)
			Expect(tabKeys(next.Root)).To(Equal([]uuid.UUID{tab1}))
			refreshed := MustBeOk(tabByKey(next.Root, tab1))
			Expect(refreshed.Variant).To(Equal(panel.TabResource{
				TabBase:  panel.TabBase{Key: tab1},
				Resource: tabResource(tab3),
			}))
		})

		It("Should be a no-op when the resource already backs another tab", func() {
			p := panel.Panel{Root: leafNode(tab(tab1), viewTab(tab2, "selector"))}
			next := MustSucceed(
				panel.SetTabResourcePayload{Key: tab2, Resource: tabResource(tab1)}.Handle(p),
			)
			Expect(next).To(Equal(p))
		})

		It("Should return ErrTabNotFound when no tab matches the key", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			Expect(
				panel.SetTabResourcePayload{Key: uuid.New(), Resource: tabResource(tab2)}.Handle(p),
			).Error().To(MatchError(ContainSubstring("tab not found in tree")))
		})
	})

	Describe("SetTabView", func() {
		It("Should swap a resource tab to the view in place", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			view := panel.View{Type: "docs"}
			next := MustSucceed(panel.SetTabViewPayload{Key: tab1, View: view}.Handle(p))
			leaf := MustBeOk(asLeaf(next.Root))
			Expect(leaf.Tabs[0]).To(Equal(panel.Tab{Variant: panel.TabView{
				TabBase: panel.TabBase{Key: tab1},
				View:    view,
			}}))
		})

		It("Should return ErrTabNotFound when no tab matches the key", func() {
			p := panel.Panel{Root: leafNode(tab(tab1))}
			view := panel.View{Type: "docs"}
			Expect(panel.SetTabViewPayload{Key: uuid.New(), View: view}.Handle(p)).Error().
				To(MatchError(ContainSubstring("tab not found in tree")))
		})
	})
})
