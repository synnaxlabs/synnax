// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v14_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	v14 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v14"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateSchematic", func() {
	var (
		valve = v13.ValveElementConfig{Orientation: "left", Scale: 2}
		pipe  = v13.PipeElementConfig{
			SegmentedEdgeConfig: v13.SegmentedEdgeConfig{Segments: []v13.Segment{}},
		}
	)
	node := func(key string) v13.Node {
		return v13.Node{Key: key, Position: spatial.XY{X: 1, Y: 2}, ZIndex: 3}
	}
	edge := func(key, source, target string) v13.Edge {
		return v13.Edge{
			Key:    key,
			Source: v13.Handle{Node: source, Param: "out"},
			Target: v13.Handle{Node: target, Param: "in"},
		}
	}
	migrate := func(ctx SpecContext, old v13.Schematic) v14.Schematic {
		GinkgoHelper()
		return MustSucceed(v14.MigrateSchematic(ctx, old))
	}

	It("Should move each config onto its node or edge", func(ctx SpecContext) {
		key := uuid.New()
		s := migrate(ctx, v13.Schematic{
			Key:   key,
			Name:  "Engine",
			Nodes: []v13.Node{node("n1"), node("n2")},
			Edges: []v13.Edge{edge("e1", "n1", "n2")},
			Configs: map[string]v13.ElementConfig{
				"n1": {Variant: valve},
				"n2": {Variant: valve},
				"e1": {Variant: pipe},
			},
		})
		Expect(s.Key).To(Equal(key))
		Expect(s.Name).To(Equal("Engine"))
		Expect(s.Nodes).To(Equal([]v14.Node{
			{
				Key:      "n1",
				Position: spatial.XY{X: 1, Y: 2},
				ZIndex:   3,
				Config:   v14.NodeConfig{Variant: v14.ValveNodeConfig(valve)},
			},
			{
				Key:      "n2",
				Position: spatial.XY{X: 1, Y: 2},
				ZIndex:   3,
				Config:   v14.NodeConfig{Variant: v14.ValveNodeConfig(valve)},
			},
		}))
		Expect(s.Edges).To(Equal([]v14.Edge{{
			Key:    "e1",
			Source: v13.Handle{Node: "n1", Param: "out"},
			Target: v13.Handle{Node: "n2", Param: "in"},
			Config: v14.EdgeConfig{Variant: v14.PipeEdgeConfig(pipe)},
		}}))
	})

	It("Should drop a config that no node or edge has", func(ctx SpecContext) {
		s := migrate(ctx, v13.Schematic{
			Nodes: []v13.Node{node("n1")},
			Configs: map[string]v13.ElementConfig{
				"n1":     {Variant: valve},
				"orphan": {Variant: valve},
			},
		})
		Expect(s.Nodes).To(HaveLen(1))
		Expect(s.Nodes[0].Key).To(Equal("n1"))
	})

	DescribeTable("Should drop a node",
		func(ctx SpecContext, configs map[string]v13.ElementConfig) {
			s := migrate(ctx, v13.Schematic{
				Nodes:   []v13.Node{node("n1"), node("n2")},
				Configs: configs,
			})
			Expect(s.Nodes).To(HaveLen(1))
			Expect(s.Nodes[0].Key).To(Equal("n2"))
		},
		Entry("with no config", map[string]v13.ElementConfig{
			"n2": {Variant: valve},
		}),
		Entry("whose config is an edge config", map[string]v13.ElementConfig{
			"n1": {Variant: pipe},
			"n2": {Variant: valve},
		}),
	)

	DescribeTable("Should give the default pipe to an edge",
		func(ctx SpecContext, configs map[string]v13.ElementConfig) {
			configs["n1"] = v13.ElementConfig{Variant: valve}
			configs["n2"] = v13.ElementConfig{Variant: valve}
			s := migrate(ctx, v13.Schematic{
				Nodes:   []v13.Node{node("n1"), node("n2")},
				Edges:   []v13.Edge{edge("e1", "n1", "n2")},
				Configs: configs,
			})
			Expect(s.Edges).To(Equal([]v14.Edge{{
				Key:    "e1",
				Source: v13.Handle{Node: "n1", Param: "out"},
				Target: v13.Handle{Node: "n2", Param: "in"},
				Config: v14.EdgeConfig{Variant: v14.PipeEdgeConfig(pipe)},
			}}))
		},
		Entry("with no config", map[string]v13.ElementConfig{}),
		Entry(
			"whose config is a node config",
			map[string]v13.ElementConfig{"e1": {Variant: valve}},
		),
	)

	DescribeTable("Should drop an edge",
		func(ctx SpecContext, e v13.Edge, configs map[string]v13.ElementConfig) {
			s := migrate(ctx, v13.Schematic{
				Nodes:   []v13.Node{node("n1"), node("n2"), node("bare")},
				Edges:   []v13.Edge{e, edge("kept", "n1", "n2")},
				Configs: configs,
			})
			Expect(s.Edges).To(HaveLen(1))
			Expect(s.Edges[0].Key).To(Equal("kept"))
		},
		Entry(
			"whose source node is dropped",
			edge("e1", "bare", "n2"),
			map[string]v13.ElementConfig{
				"n1":   {Variant: valve},
				"n2":   {Variant: valve},
				"e1":   {Variant: pipe},
				"kept": {Variant: pipe},
			},
		),
		Entry(
			"whose target node is dropped",
			edge("e1", "n1", "bare"),
			map[string]v13.ElementConfig{
				"n1":   {Variant: valve},
				"n2":   {Variant: valve},
				"e1":   {Variant: pipe},
				"kept": {Variant: pipe},
			},
		),
		Entry(
			"whose target node does not exist",
			edge("e1", "n1", "ghost"),
			map[string]v13.ElementConfig{
				"n1":   {Variant: valve},
				"n2":   {Variant: valve},
				"e1":   {Variant: pipe},
				"kept": {Variant: pipe},
			},
		),
	)

	It("Should not change the schematic it was given", func(ctx SpecContext) {
		old := v13.Schematic{
			Nodes:   []v13.Node{node("n1"), node("n2")},
			Configs: map[string]v13.ElementConfig{"n2": {Variant: valve}},
		}
		migrate(ctx, old)
		Expect(old.Nodes).To(Equal([]v13.Node{node("n1"), node("n2")}))
	})
})

var _ = Describe("MigrateNode", func() {
	It("Should copy the node with no config", func(ctx SpecContext) {
		Expect(MustSucceed(v14.MigrateNode(ctx, v13.Node{
			Key:      "n1",
			Position: spatial.XY{X: 1, Y: 2},
			ZIndex:   3,
		}))).To(Equal(v14.Node{Key: "n1", Position: spatial.XY{X: 1, Y: 2}, ZIndex: 3}))
	})
})

var _ = Describe("MigrateEdge", func() {
	It("Should copy the edge with no config", func(ctx SpecContext) {
		source := v13.Handle{Node: "n1", Param: "out"}
		target := v13.Handle{Node: "n2", Param: "in"}
		Expect(MustSucceed(v14.MigrateEdge(ctx, v13.Edge{
			Key:    "e1",
			Source: source,
			Target: target,
		}))).To(Equal(v14.Edge{Key: "e1", Source: source, Target: target}))
	})
})
