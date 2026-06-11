// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/panel/pb"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

func TestPanelPB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Panel PB Suite")
}

// view builds a non-trivial view, including an opaque Args map whose string and
// bool values survive the structpb round-trip unchanged.
func view() panel.TabView {
	return panel.TabView{
		Type: "docs",
		Name: "Docs",
		Args: msgpack.EncodedJSON{"path": "/intro", "pinned": true},
	}
}

func resourceTab() panel.Tab {
	id := ontology.ID{Type: ontology.ResourceTypeSchematic, Key: uuid.New().String()}
	return panel.Tab{Key: uuid.New(), Resource: &id}
}

func viewTab() panel.Tab {
	v := view()
	return panel.Tab{Key: uuid.New(), View: &v}
}

// nestedPanel exercises every translator: a split whose sides are leaves holding
// a resource-backed tab and a view-backed tab respectively.
func nestedPanel() panel.Panel {
	return panel.Panel{
		Key:  uuid.New(),
		Name: "nested",
		Root: panel.Node{Split: &panel.Split{
			Direction: spatial.DirectionX,
			Size:      0.4,
			First:     &panel.Node{Leaf: &panel.Leaf{Tabs: []panel.Tab{resourceTab()}}},
			Last:      &panel.Node{Leaf: &panel.Leaf{Tabs: []panel.Tab{viewTab()}}},
		}},
	}
}

var _ = Describe("Translator", func() {
	Describe("Panel", func() {
		It("Should round-trip a deeply nested panel through protobuf", func() {
			p := nestedPanel()
			back := MustSucceed(pb.PanelFromPB(MustSucceed(pb.PanelToPB(p))))
			Expect(back).To(Equal(p))
		})

		It("Should round-trip a single empty-leaf panel", func() {
			p := panel.Panel{
				Key:  uuid.New(),
				Name: "leaf",
				Root: panel.Node{Leaf: &panel.Leaf{Tabs: []panel.Tab{}}},
			}
			back := MustSucceed(pb.PanelFromPB(MustSucceed(pb.PanelToPB(p))))
			Expect(back).To(Equal(p))
		})

		It("Should round-trip a slice of panels", func() {
			ps := []panel.Panel{nestedPanel(), nestedPanel()}
			back := MustSucceed(pb.PanelsFromPB(MustSucceed(pb.PanelsToPB(ps))))
			Expect(back).To(Equal(ps))
		})

		It("Should propagate the encoding error when a view's args are not structpb-encodable", func() {
			p := panel.Panel{
				Key:  uuid.New(),
				Name: "bad-args",
				Root: panel.Node{Leaf: &panel.Leaf{Tabs: []panel.Tab{{
					Key:  uuid.New(),
					View: &panel.TabView{Type: "x", Args: msgpack.EncodedJSON{"bad": make(chan int)}},
				}}}},
			}
			Expect(pb.PanelToPB(p)).Error().To(MatchError(ContainSubstring("invalid type")))
		})
	})

	Describe("Tab and TabView", func() {
		It("Should round-trip a resource-backed tab", func() {
			t := resourceTab()
			back := MustSucceed(pb.TabFromPB(MustSucceed(pb.TabToPB(t))))
			Expect(back).To(Equal(t))
		})

		It("Should round-trip a view-backed tab with opaque args", func() {
			t := viewTab()
			back := MustSucceed(pb.TabFromPB(MustSucceed(pb.TabToPB(t))))
			Expect(back).To(Equal(t))
		})

		It("Should round-trip a slice of tabs", func() {
			ts := []panel.Tab{resourceTab(), viewTab()}
			back := MustSucceed(pb.TabsFromPB(MustSucceed(pb.TabsToPB(ts))))
			Expect(back).To(Equal(ts))
		})

		It("Should round-trip a slice of views", func() {
			vs := []panel.TabView{view(), {Type: "about", Args: msgpack.EncodedJSON{}}}
			back := MustSucceed(pb.TabViewsFromPB(MustSucceed(pb.TabViewsToPB(vs))))
			Expect(back).To(Equal(vs))
		})

		It("Should return a zero TabView when decoding nil", func() {
			Expect(pb.TabViewFromPB(nil)).To(Equal(panel.TabView{}))
		})
	})

	Describe("Node, Leaf, and Split", func() {
		It("Should round-trip a leaf node", func() {
			n := panel.Node{Leaf: &panel.Leaf{Tabs: []panel.Tab{resourceTab()}}}
			back := MustSucceed(pb.NodeFromPB(MustSucceed(pb.NodeToPB(n))))
			Expect(back).To(Equal(n))
		})

		It("Should round-trip a slice of leaves", func() {
			ls := []panel.Leaf{{Tabs: []panel.Tab{resourceTab()}}, {Tabs: []panel.Tab{}}}
			back := MustSucceed(pb.LeavesFromPB(MustSucceed(pb.LeavesToPB(ls))))
			Expect(back).To(Equal(ls))
		})

		It("Should round-trip a slice of splits", func() {
			ss := []panel.Split{{
				Direction: spatial.DirectionY,
				Size:      0.5,
				First:     &panel.Node{Leaf: &panel.Leaf{Tabs: []panel.Tab{viewTab()}}},
				Last:      &panel.Node{Leaf: &panel.Leaf{Tabs: []panel.Tab{}}},
			}}
			back := MustSucceed(pb.SplitsFromPB(MustSucceed(pb.SplitsToPB(ss))))
			Expect(back).To(Equal(ss))
		})

		It("Should round-trip a slice of nodes", func() {
			ns := []panel.Node{
				{Leaf: &panel.Leaf{Tabs: []panel.Tab{resourceTab()}}},
				{Split: &panel.Split{
					Direction: spatial.DirectionX,
					Size:      0.3,
					First:     &panel.Node{Leaf: &panel.Leaf{Tabs: []panel.Tab{}}},
					Last:      &panel.Node{Leaf: &panel.Leaf{Tabs: []panel.Tab{}}},
				}},
			}
			back := MustSucceed(pb.NodesFromPB(MustSucceed(pb.NodesToPB(ns))))
			Expect(back).To(Equal(ns))
		})
	})
})
