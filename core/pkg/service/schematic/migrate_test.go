// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	v55 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v55"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

// jsonMap re-marshals a JSON literal into the msgpack.EncodedJSON shape that
// the v55 snapshot stores. Tests use it to build schematic data blobs that
// match the wire format consoles actually persist.
func jsonMap(raw string) msgpack.EncodedJSON {
	var m map[string]any
	Expect(json.Unmarshal([]byte(raw), &m)).To(Succeed())
	return m
}

var _ = Describe("MigrateSchematic", func() {
	Describe("v5 wire format", func() {
		It("Should reshape edges, lift edge.data, rename node-prop key, and pass legend / authority through", func(ctx SpecContext) {
			key := uuid.New()
			snap := v55.Schematic{
				Key:  key,
				Name: "tank-1",
				Data: jsonMap(`{
					"version": "5.0.0",
					"authority": 5,
					"nodes": [{"key": "n1", "position": {"x": 10, "y": 20}}],
					"edges": [{"key": "e1", "source": "n1", "target": "n2", "sourceHandle": "outlet", "targetHandle": "inlet"}],
					"props": {"n1": {"key": "valve", "color": "#ff0000"}},
					"legend": {"visible": true, "position": {"x": 50, "y": 50, "units": {"x": "px", "y": "px"}}, "colors": {}}
				}`),
			}
			out := MustSucceed(schematic.MigrateSchematic(ctx, snap))
			Expect(out.Key).To(Equal(key))
			Expect(out.Name).To(Equal("tank-1"))
			Expect(out.Authority).To(BeEquivalentTo(5))
			Expect(out.Nodes).To(HaveLen(1))
			Expect(out.Nodes[0].Key).To(Equal("n1"))
			Expect(out.Nodes[0].Position.X).To(Equal(10.0))
			Expect(out.Edges).To(HaveLen(1))
			Expect(out.Edges[0].Source).To(Equal(schematic.Handle{Node: "n1", Param: "outlet"}))
			Expect(out.Edges[0].Target).To(Equal(schematic.Handle{Node: "n2", Param: "inlet"}))
			Expect(out.Props["n1"]).To(HaveKey("variant"))
			Expect(out.Props["n1"]).NotTo(HaveKey("key"))
			Expect(out.Legend.Visible).To(BeTrue())
		})

		It("Should default authority to 1 when the blob carries zero", func(ctx SpecContext) {
			snap := v55.Schematic{
				Key:  uuid.New(),
				Data: jsonMap(`{"version": "5.0.0", "authority": 0, "nodes": [], "edges": [], "props": {}}`),
			}
			out := MustSucceed(schematic.MigrateSchematic(ctx, snap))
			Expect(out.Authority).To(BeEquivalentTo(1))
		})
	})

	Describe("older wire formats", func() {
		It("Should chain a v0 blob through every step migration", func(ctx SpecContext) {
			snap := v55.Schematic{
				Key:  uuid.New(),
				Data: jsonMap(`{
					"version": "0.0.0",
					"nodes": [{"key": "n1", "position": {"x": 0, "y": 0}}],
					"edges": [{"key": "e1", "source": "n1", "target": "n2", "sourceHandle": "out", "targetHandle": "in"}],
					"props": {"n1": {"key": "valve"}}
				}`),
			}
			out := MustSucceed(schematic.MigrateSchematic(ctx, snap))
			Expect(out.Edges[0].Source).To(Equal(schematic.Handle{Node: "n1", Param: "out"}))
			Expect(out.Edges[0].Target).To(Equal(schematic.Handle{Node: "n2", Param: "in"}))
			Expect(out.Authority).To(BeEquivalentTo(1))
		})

		It("Should rename node-prop key to variant when lifting from v5", func(ctx SpecContext) {
			snap := v55.Schematic{
				Key: uuid.New(),
				Data: jsonMap(`{
					"version": "5.0.0",
					"nodes": [{"key": "n1", "position": {"x": 0, "y": 0}}],
					"edges": [],
					"props": {"n1": {"key": "tank", "color": "#00ff00"}}
				}`),
			}
			out := MustSucceed(schematic.MigrateSchematic(ctx, snap))
			Expect(out.Props).To(HaveKey("n1"))
			Expect(out.Props["n1"]).To(HaveKey("variant"))
			Expect(out.Props["n1"]).NotTo(HaveKey("key"))
			Expect(out.Props["n1"]["variant"]).To(Equal("tank"))
		})

		It("Should lift edge.data segments, color, and variant into props keyed by edge id", func(ctx SpecContext) {
			snap := v55.Schematic{
				Key: uuid.New(),
				Data: jsonMap(`{
					"version": "5.0.0",
					"nodes": [],
					"edges": [{
						"key": "e1",
						"source": "n1",
						"target": "n2",
						"sourceHandle": "out",
						"targetHandle": "in",
						"data": {"segments": [{"direction": "x", "length": 30}], "color": "#0000ff", "variant": "pipe"}
					}],
					"props": {}
				}`),
			}
			out := MustSucceed(schematic.MigrateSchematic(ctx, snap))
			Expect(out.Props).To(HaveKey("e1"))
			Expect(out.Props["e1"]).To(HaveKey("segments"))
			Expect(out.Props["e1"]).To(HaveKey("color"))
			Expect(out.Props["e1"]).To(HaveKey("variant"))
		})

		It("Should preserve user-set zIndex on nodes through the chain", func(ctx SpecContext) {
			snap := v55.Schematic{
				Key: uuid.New(),
				Data: jsonMap(`{
					"version": "0.0.0",
					"nodes": [
						{"key": "back", "position": {"x": 0, "y": 0}, "zIndex": -1},
						{"key": "front", "position": {"x": 10, "y": 10}, "zIndex": 7}
					],
					"edges": [],
					"props": {}
				}`),
			}
			out := MustSucceed(schematic.MigrateSchematic(ctx, snap))
			Expect(out.Nodes).To(HaveLen(2))
			Expect(out.Nodes[0].ZIndex).To(BeEquivalentTo(-1))
			Expect(out.Nodes[1].ZIndex).To(BeEquivalentTo(7))
		})

		It("Should fall back to v0 when the blob has no version field", func(ctx SpecContext) {
			snap := v55.Schematic{
				Key:  uuid.New(),
				Data: jsonMap(`{"nodes": [], "edges": [], "props": {}}`),
			}
			out := MustSucceed(schematic.MigrateSchematic(ctx, snap))
			Expect(out.Authority).To(BeEquivalentTo(1))
		})
	})

	Describe("nil and empty blobs", func() {
		It("Should handle a nil data blob without erroring", func(ctx SpecContext) {
			snap := v55.Schematic{Key: uuid.New(), Name: "empty", Data: nil}
			out := MustSucceed(schematic.MigrateSchematic(ctx, snap))
			Expect(out.Nodes).To(BeEmpty())
			Expect(out.Edges).To(BeEmpty())
			Expect(out.Authority).To(BeEquivalentTo(1))
		})
	})

})
