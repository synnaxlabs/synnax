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

// resourceTab builds a core-backed tab referencing a backing document.
func resourceTab() panel.Tab {
	return panel.Tab{Variant: panel.TabResource{
		TabBase: panel.TabBase{Key: uuid.New()},
		Resource: ontology.ID{
			Type: ontology.ResourceTypeSchematic,
			Key:  uuid.New().String(),
		},
	}}
}

// viewTab builds an inline tab with an opaque args map whose string and bool values
// survive the structpb round-trip unchanged.
func viewTab() panel.Tab {
	return panel.Tab{Variant: panel.TabView{
		TabBase: panel.TabBase{Key: uuid.New()},
		View: panel.View{
			Type: "docs",
			Name: "Docs",
			Args: msgpack.EncodedJSON{"path": "/intro", "pinned": true},
		},
	}}
}

func leafNode(tabs ...panel.Tab) panel.Node {
	return panel.Node{Variant: panel.NodeLeaf{
		Leaf: panel.Leaf{Tabs: append([]panel.Tab{}, tabs...)},
	}}
}

// nestedPanel exercises every translator: a split whose sides are leaves holding
// a resource-backed tab and a view-backed tab respectively.
func nestedPanel() panel.Panel {
	return panel.Panel{
		Key:  uuid.New(),
		Name: "nested",
		Root: panel.Node{Variant: panel.NodeSplit{Split: panel.Split{
			Direction: spatial.DirectionX,
			Size:      0.4,
			First:     leafNode(resourceTab()),
			Last:      leafNode(viewTab()),
		}}},
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
				Root: leafNode(),
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
				Root: leafNode(panel.Tab{Variant: panel.TabView{
					TabBase: panel.TabBase{Key: uuid.New()},
					View: panel.View{
						Type: "x",
						Args: msgpack.EncodedJSON{"bad": make(chan int)},
					},
				}}),
			}
			Expect(pb.PanelToPB(p)).Error().To(MatchError(ContainSubstring("invalid type")))
		})
	})

	Describe("Tab", func() {
		It("Should round-trip a resource-backed tab", func() {
			t := resourceTab()
			back := MustSucceed(pb.TabFromPB(MustSucceed(pb.TabToPB(t))))
			Expect(back).To(Equal(t))
		})

		It("Should round-trip an inline tab with opaque args", func() {
			t := viewTab()
			back := MustSucceed(pb.TabFromPB(MustSucceed(pb.TabToPB(t))))
			Expect(back).To(Equal(t))
		})

		It("Should round-trip a slice of tabs", func() {
			ts := []panel.Tab{resourceTab(), viewTab()}
			back := MustSucceed(pb.TabsFromPB(MustSucceed(pb.TabsToPB(ts))))
			Expect(back).To(Equal(ts))
		})
	})

	Describe("Node, Leaf, and Split", func() {
		It("Should round-trip a leaf node", func() {
			n := leafNode(resourceTab())
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
				First:     leafNode(viewTab()),
				Last:      leafNode(),
			}}
			back := MustSucceed(pb.SplitsFromPB(MustSucceed(pb.SplitsToPB(ss))))
			Expect(back).To(Equal(ss))
		})

		It("Should round-trip a slice of nodes", func() {
			ns := []panel.Node{
				leafNode(resourceTab()),
				{Variant: panel.NodeSplit{Split: panel.Split{
					Direction: spatial.DirectionX,
					Size:      0.3,
					First:     leafNode(),
					Last:      leafNode(),
				}}},
			}
			back := MustSucceed(pb.NodesFromPB(MustSucceed(pb.NodesToPB(ns))))
			Expect(back).To(Equal(ns))
		})
	})
})

var _ = ShouldNotLeakGoroutinesPerSpec()
