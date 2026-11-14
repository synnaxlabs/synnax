// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v12_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	v12 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v12"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateSchematic", func() {
	var (
		valve = v11.ValveElementConfig{Orientation: "left", Scale: 2}
		pipe  = v11.PipeElementConfig{
			SegmentedEdgeConfig: v11.SegmentedEdgeConfig{Segments: []v11.Segment{}},
		}
	)
	node := func(key string) v11.Node {
		return v11.Node{Key: key, Position: spatial.XY{X: 1, Y: 2}, ZIndex: 3}
	}
	edge := func(key, source, target string) v11.Edge {
		return v11.Edge{
			Key:    key,
			Source: v11.Handle{Node: source, Param: "out"},
			Target: v11.Handle{Node: target, Param: "in"},
		}
	}
	migrate := func(ctx SpecContext, old v11.Schematic) v12.Schematic {
		GinkgoHelper()
		return MustSucceed(v12.MigrateSchematic(ctx, old))
	}

	It("Should move each config onto its node or edge", func(ctx SpecContext) {
		key := uuid.New()
		s := migrate(ctx, v11.Schematic{
			Key:      key,
			Name:     "Engine",
			Snapshot: true,
			Nodes:    []v11.Node{node("n1"), node("n2")},
			Edges:    []v11.Edge{edge("e1", "n1", "n2")},
			Configs: map[string]v11.ElementConfig{
				"n1": {Variant: valve},
				"n2": {Variant: valve},
				"e1": {Variant: pipe},
			},
		})
		Expect(s.Key).To(Equal(key))
		Expect(s.Name).To(Equal("Engine"))
		Expect(s.Snapshot).To(BeTrue())
		Expect(s.Nodes).To(Equal([]v12.Node{
			{
				Key:      "n1",
				Position: spatial.XY{X: 1, Y: 2},
				ZIndex:   3,
				Config:   v12.NodeConfig{Variant: v12.ValveNodeConfig(valve)},
			},
			{
				Key:      "n2",
				Position: spatial.XY{X: 1, Y: 2},
				ZIndex:   3,
				Config:   v12.NodeConfig{Variant: v12.ValveNodeConfig(valve)},
			},
		}))
		Expect(s.Edges).To(Equal([]v12.Edge{{
			Key:    "e1",
			Source: v11.Handle{Node: "n1", Param: "out"},
			Target: v11.Handle{Node: "n2", Param: "in"},
			Config: v12.EdgeConfig{Variant: v12.PipeEdgeConfig(pipe)},
		}}))
	})

	It("Should drop a config that no node or edge has", func(ctx SpecContext) {
		s := migrate(ctx, v11.Schematic{
			Nodes: []v11.Node{node("n1")},
			Configs: map[string]v11.ElementConfig{
				"n1":     {Variant: valve},
				"orphan": {Variant: valve},
			},
		})
		Expect(s.Nodes).To(HaveLen(1))
		Expect(s.Nodes[0].Key).To(Equal("n1"))
	})

	DescribeTable("Should drop a node",
		func(ctx SpecContext, configs map[string]v11.ElementConfig) {
			s := migrate(ctx, v11.Schematic{
				Nodes:   []v11.Node{node("n1"), node("n2")},
				Configs: configs,
			})
			Expect(s.Nodes).To(HaveLen(1))
			Expect(s.Nodes[0].Key).To(Equal("n2"))
		},
		Entry("with no config", map[string]v11.ElementConfig{
			"n2": {Variant: valve},
		}),
		Entry("whose config names no variant", map[string]v11.ElementConfig{
			"n1": {},
			"n2": {Variant: valve},
		}),
		Entry("whose config is an edge config", map[string]v11.ElementConfig{
			"n1": {Variant: pipe},
			"n2": {Variant: valve},
		}),
	)

	DescribeTable("Should give the default pipe to an edge",
		func(ctx SpecContext, configs map[string]v11.ElementConfig) {
			configs["n1"] = v11.ElementConfig{Variant: valve}
			configs["n2"] = v11.ElementConfig{Variant: valve}
			s := migrate(ctx, v11.Schematic{
				Nodes:   []v11.Node{node("n1"), node("n2")},
				Edges:   []v11.Edge{edge("e1", "n1", "n2")},
				Configs: configs,
			})
			Expect(s.Edges).To(Equal([]v12.Edge{{
				Key:    "e1",
				Source: v11.Handle{Node: "n1", Param: "out"},
				Target: v11.Handle{Node: "n2", Param: "in"},
				Config: v12.EdgeConfig{Variant: v12.PipeEdgeConfig(pipe)},
			}}))
		},
		Entry("with no config", map[string]v11.ElementConfig{}),
		Entry("whose config names no variant", map[string]v11.ElementConfig{"e1": {}}),
		Entry(
			"whose config is a node config",
			map[string]v11.ElementConfig{"e1": {Variant: valve}},
		),
	)

	DescribeTable("Should drop an edge",
		func(ctx SpecContext, e v11.Edge, configs map[string]v11.ElementConfig) {
			s := migrate(ctx, v11.Schematic{
				Nodes:   []v11.Node{node("n1"), node("n2"), node("bare")},
				Edges:   []v11.Edge{e, edge("kept", "n1", "n2")},
				Configs: configs,
			})
			Expect(s.Edges).To(HaveLen(1))
			Expect(s.Edges[0].Key).To(Equal("kept"))
		},
		Entry(
			"whose source node is dropped",
			edge("e1", "bare", "n2"),
			map[string]v11.ElementConfig{
				"n1":   {Variant: valve},
				"n2":   {Variant: valve},
				"e1":   {Variant: pipe},
				"kept": {Variant: pipe},
			},
		),
		Entry(
			"whose target node is dropped",
			edge("e1", "n1", "bare"),
			map[string]v11.ElementConfig{
				"n1":   {Variant: valve},
				"n2":   {Variant: valve},
				"e1":   {Variant: pipe},
				"kept": {Variant: pipe},
			},
		),
		Entry(
			"whose target node does not exist",
			edge("e1", "n1", "ghost"),
			map[string]v11.ElementConfig{
				"n1":   {Variant: valve},
				"n2":   {Variant: valve},
				"e1":   {Variant: pipe},
				"kept": {Variant: pipe},
			},
		),
	)

	It("Should not change the schematic it was given", func(ctx SpecContext) {
		old := v11.Schematic{
			Nodes:   []v11.Node{node("n1"), node("n2")},
			Configs: map[string]v11.ElementConfig{"n2": {Variant: valve}},
		}
		migrate(ctx, old)
		Expect(old.Nodes).To(Equal([]v11.Node{node("n1"), node("n2")}))
	})
})

var _ = Describe("MigrateNode", func() {
	It("Should copy the node with no config", func(ctx SpecContext) {
		Expect(MustSucceed(v12.MigrateNode(ctx, v11.Node{
			Key:      "n1",
			Position: spatial.XY{X: 1, Y: 2},
			ZIndex:   3,
		}))).To(Equal(v12.Node{Key: "n1", Position: spatial.XY{X: 1, Y: 2}, ZIndex: 3}))
	})
})

var _ = Describe("MigrateEdge", func() {
	It("Should copy the edge with no config", func(ctx SpecContext) {
		source := v11.Handle{Node: "n1", Param: "out"}
		target := v11.Handle{Node: "n2", Param: "in"}
		Expect(MustSucceed(v12.MigrateEdge(ctx, v11.Edge{
			Key:    "e1",
			Source: source,
			Target: target,
		}))).To(Equal(v12.Edge{Key: "e1", Source: source, Target: target}))
	})
})
