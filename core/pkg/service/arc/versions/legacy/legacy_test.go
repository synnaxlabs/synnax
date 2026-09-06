// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	graphv0 "github.com/synnaxlabs/arc/graph/versions/v0"
	graph "github.com/synnaxlabs/arc/graph/versions/v1"
	ir "github.com/synnaxlabs/arc/ir/versions/v0"
	text "github.com/synnaxlabs/arc/text/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/console"
	"github.com/synnaxlabs/x/encoding/msgpack"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

// v0State builds a v0 Console Arc state with every model field populated so
// passthrough regressions in the chain surface.
func v0State() msgpack.EncodedJSON {
	return msgpack.EncodedJSON{
		"version": "0.0.0",
		"graph": map[string]any{
			"nodes": []any{map[string]any{
				"key":      "n1",
				"position": map[string]any{"x": 3, "y": 4},
			}},
			"edges": []any{map[string]any{
				"key":          "e1",
				"source":       "n1",
				"target":       "n2",
				"sourceHandle": "out",
				"targetHandle": "in",
			}},
			"props": map[string]any{
				"n1": map[string]any{"key": "stl.on", "channel": "valve_cmd"},
			},
		},
		"text": map[string]any{"raw": "chan a = 1"},
		"mode": "graph",
	}
}

// stateWithProps builds a state at the given version carrying props for one node.
// version takes both wire forms: the "N.0.0" semver string and the bare number.
func stateWithProps(version any, props map[string]any) msgpack.EncodedJSON {
	return msgpack.EncodedJSON{
		"version": version,
		"graph":   map[string]any{"props": map[string]any{"n1": props}},
		"mode":    "graph",
	}
}

var _ = Describe("MigrateData", func() {
	It("Should error on an unknown declared version", func() {
		Expect(legacy.MigrateData(msgpack.EncodedJSON{"version": "99.0.0"})).
			Error().To(MatchError(ContainSubstring("unknown arc state version 99")))
	})

	It("Should error on a malformed version", func() {
		Expect(legacy.MigrateData(msgpack.EncodedJSON{"version": ""})).
			Error().To(MatchError(ContainSubstring("invalid version")))
	})

	It("Should walk a v0 state forward into the typed document", func() {
		doc := MustSucceed(legacy.MigrateData(v0State()))

		Expect(doc.Graph.Nodes).To(Equal(graph.Nodes{
			{Key: "n1", Position: spatial.XY{X: 3, Y: 4}},
		}))
		Expect(doc.Graph.Edges).To(Equal(graph.Edges{{
			Key:    "e1",
			Source: ir.Handle{Node: "n1", Param: "out"},
			Target: ir.Handle{Node: "n2", Param: "in"},
			Kind:   ir.EdgeKindContinuous,
		}}))
		Expect(doc.Text).To(Equal(text.Text{Raw: "chan a = 1"}))
		Expect(doc.Mode).To(Equal("graph"))
	})

	It("Should rename the props function key onto the typed input type", func() {
		doc := MustSucceed(legacy.MigrateData(v0State()))

		Expect(doc.Graph.Inputs).To(Equal(map[string]msgpack.EncodedJSON{
			"n1": {"type": "stl.on", "channel": "valve_cmd"},
		}))
	})

	It("Should read an absent v0 handle as an empty param", func() {
		state := v0State()
		edge := state["graph"].(map[string]any)["edges"].([]any)[0].(map[string]any)
		delete(edge, "sourceHandle")
		delete(edge, "targetHandle")

		doc := MustSucceed(legacy.MigrateData(state))

		Expect(doc.Graph.Edges[0].Source).To(Equal(ir.Handle{Node: "n1"}))
		Expect(doc.Graph.Edges[0].Target).To(Equal(ir.Handle{Node: "n2"}))
	})

	It("Should apply the v2 set_status remap to a v0 state", func() {
		doc := MustSucceed(legacy.MigrateData(stateWithProps("0.0.0", map[string]any{
			"key":       "set_status",
			"statusKey": "pump_state",
			"variant":   "error",
			"message":   "pump down",
		})))

		Expect(doc.Graph.Inputs["n1"]).To(Equal(msgpack.EncodedJSON{
			"type":        "status.set",
			"key_or_name": "pump_state",
			"variant":     "error",
			"message":     "pump down",
		}))
	})

	It("Should take a v1 state's edges already nested", func() {
		doc := MustSucceed(legacy.MigrateData(msgpack.EncodedJSON{
			"version": "1.0.0",
			"graph": map[string]any{"edges": []any{map[string]any{
				"key":    "e1",
				"source": map[string]any{"node": "n1", "param": "out"},
				"target": map[string]any{"node": "n2", "param": "in"},
			}}},
		}))

		Expect(doc.Graph.Edges).To(Equal(graph.Edges{{
			Key:    "e1",
			Source: ir.Handle{Node: "n1", Param: "out"},
			Target: ir.Handle{Node: "n2", Param: "in"},
			Kind:   ir.EdgeKindContinuous,
		}}))
	})

	It("Should leave set_status props alone in a state already at v2", func() {
		doc := MustSucceed(legacy.MigrateData(stateWithProps("2.0.0", map[string]any{
			"key":       "set_status",
			"statusKey": "pump_state",
		})))

		Expect(doc.Graph.Inputs["n1"]).To(Equal(msgpack.EncodedJSON{
			"type":      "set_status",
			"statusKey": "pump_state",
		}))
	})

	It("Should accept the numeric version form", func() {
		doc := MustSucceed(legacy.MigrateData(stateWithProps(1, map[string]any{
			"key": "stl.on",
		})))

		Expect(doc.Graph.Inputs["n1"]).To(Equal(msgpack.EncodedJSON{"type": "stl.on"}))
	})

	It("Should fall back to v0 when the blob carries no version field", func() {
		state := v0State()
		delete(state, "version")

		doc := MustSucceed(legacy.MigrateData(state))

		Expect(doc.Graph.Edges[0].Source).To(Equal(ir.Handle{Node: "n1", Param: "out"}))
	})

	It("Should leave the inputs nil when the state carries no props", func() {
		state := v0State()
		delete(state["graph"].(map[string]any), "props")

		Expect(MustSucceed(legacy.MigrateData(state)).Graph.Inputs).To(BeNil())
	})

	It("Should produce a zero document for a nil blob", func() {
		doc := MustSucceed(legacy.MigrateData(nil))

		Expect(doc.Graph.Nodes).To(BeEmpty())
		Expect(doc.Graph.Edges).To(BeEmpty())
		Expect(doc.Graph.Inputs).To(BeNil())
		Expect(doc.Text).To(Equal(text.Text{}))
		Expect(doc.Mode).To(BeEmpty())
	})
})

var _ = Describe("MigrateExport", func() {
	It("Should key the export's edges and fold node config into the inputs", func(
		ctx SpecContext,
	) {
		doc := MustSucceed(legacy.MigrateExport(ctx, legacy.Export{
			Mode: "graph",
			Text: text.Text{Raw: "a = 1"},
			Graph: console.Graph{
				Edges: ir.Edges{{
					Source: ir.Handle{Node: "n1", Param: "out"},
					Target: ir.Handle{Node: "n2", Param: "in"},
					Kind:   ir.EdgeKindContinuous,
				}},
				Nodes: graphv0.Nodes{{
					Key:      "n1",
					Type:     "constant",
					Config:   msgpack.EncodedJSON{"keyOrName": "kept"},
					Position: spatial.XY{X: 9, Y: 9},
				}},
			},
		}))

		Expect(doc.Mode).To(Equal("graph"))
		Expect(doc.Text).To(Equal(text.Text{Raw: "a = 1"}))
		Expect(doc.Graph.Nodes).To(Equal(graph.Nodes{
			{Key: "n1", Position: spatial.XY{X: 9, Y: 9}},
		}))
		Expect(doc.Graph.Edges[0].Edge).To(Equal(ir.Edge{
			Source: ir.Handle{Node: "n1", Param: "out"},
			Target: ir.Handle{Node: "n2", Param: "in"},
			Kind:   ir.EdgeKindContinuous,
		}))
		// The Console wrote keyless edges; the graph migration mints one.
		Expect(doc.Graph.Edges[0].Key).ToNot(BeEmpty())
		Expect(doc.Graph.Inputs["n1"]).To(Equal(msgpack.EncodedJSON{
			"type": "constant", "keyOrName": "kept",
		}))
	})

	It("Should produce a zero document from a zero export", func(ctx SpecContext) {
		doc := MustSucceed(legacy.MigrateExport(ctx, legacy.Export{}))

		Expect(doc.Graph.Nodes).To(BeEmpty())
		Expect(doc.Graph.Edges).To(BeEmpty())
		Expect(doc.Mode).To(BeEmpty())
	})
})
