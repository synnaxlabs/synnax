// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrations_test

import (
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v5"
	v55 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v55"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

//go:embed testdata/*.json
var fixtures embed.FS

func loadFixture(name string) (msgpack.EncodedJSON, map[string]any) {
	raw := MustSucceed(fixtures.ReadFile("testdata/" + name))
	var m map[string]any
	Expect(json.Unmarshal(raw, &m)).To(Succeed())
	return msgpack.EncodedJSON(m), m
}

func jsonMap(raw string) msgpack.EncodedJSON {
	var m map[string]any
	Expect(json.Unmarshal([]byte(raw), &m)).To(Succeed())
	return m
}

func rawJSON(s string) json.RawMessage { return json.RawMessage(s) }

func stringOr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// assertMigrated compares got against the canonical .migrated.json file for
// fixture, or rewrites it if UPDATE_MIGRATED=1 is set. Outputs are
// canonicalized via json.MarshalIndent (which sorts map keys) so diffs are
// deterministic.
func assertMigrated(fixture string, got schematic.Schematic) {
	pretty := MustSucceed(json.MarshalIndent(got, "", "  "))
	pretty = append(pretty, '\n')
	stem := strings.TrimSuffix(fixture, ".json")
	p := filepath.Join("testdata", stem+".migrated.json")
	if os.Getenv("UPDATE_MIGRATED") == "1" {
		Expect(os.WriteFile(p, pretty, 0o644)).To(Succeed())
		return
	}
	expected := MustSucceed(os.ReadFile(p))
	Expect(pretty).To(MatchJSON(expected),
		"%s drifted from its canonical migrated form — review the diff and rerun with UPDATE_MIGRATED=1 if intentional", fixture)
}

// nonZeroV0 builds a v0.Data with every field populated to a non-zero value
// so step-migrate passthrough regressions surface.
func nonZeroV0() v0.Data {
	z := -2
	w, h := 100.0, 50.0
	srcH, tgtH := "out", "in"
	return v0.Data{
		Version:         v0.Version,
		Editable:        true,
		FitViewOnResize: true,
		Snapshot:        true,
		RemoteCreated:   true,
		Viewport:        v0.Viewport{Position: v0.XY{X: 12, Y: 34}, Zoom: 1.5},
		Nodes: []v0.Node{
			{
				Key: "n1", Position: v0.XY{X: 1, Y: 2},
				ZIndex: &z, Type: "default",
				Measured: &v0.Measured{Width: &w, Height: &h},
			},
			{Key: "n2", Position: v0.XY{X: 3, Y: 4}},
		},
		Edges: []v0.Edge{{
			Key:          "e1",
			Source:       "n1",
			Target:       "n2",
			SourceHandle: &srcH,
			TargetHandle: &tgtH,
			Data:         rawJSON(`{"segments":[{"direction":"x","length":10}],"color":"#ff0000"}`),
		}},
		Props: map[string]json.RawMessage{
			"n1": rawJSON(`{"key":"valve","color":"#00ff00"}`),
		},
		Control: "released",
	}
}

var _ = Describe("MigrateSchematic", func() {
	// Snapshot tests against the canonical .migrated.json output for every
	// captured production fixture. Run with UPDATE_MIGRATED=1 to regenerate
	// the .migrated.json files after intentional migration changes.
	Describe("canonical migrated output", func() {
		fixedKey := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		DescribeTable("Should produce the canonical typed Schematic",
			func(ctx SpecContext, fixture string) {
				blob, _ := loadFixture(fixture)
				snap := v55.Schematic{Key: fixedKey, Name: fixture, Data: blob}
				out := MustSucceed(schematic.MigrateSchematic(ctx, snap))
				assertMigrated(fixture, out)
			},
			Entry("v2 condensed", "v2_gse_condensed.json"),
			Entry("v2 value-only", "v2_value_test.json"),
			Entry("v3 shop flow", "v3_shop_flow.json"),
			Entry("v4 empty", "v4_empty.json"),
			Entry("v5 hardware workspace", "v5_hardware_workspace.json"),
		)
	})

	// Drives MigrateSchematic through the real gorp migration pipeline so the
	// on-disk v55 -> typed Schematic path is exercised end-to-end.
	Describe("storage integration", func() {
		openMigratedTable := func(ctx SpecContext, db *gorp.DB) *gorp.Table[uuid.UUID, schematic.Schematic] {
			return MustOpen(gorp.OpenTable[uuid.UUID, schematic.Schematic](
				ctx, gorp.TableConfig[schematic.Schematic]{
					DB: db,
					Migrations: []migrate.Migration{
						gorp.NewEntryMigration[uuid.UUID, uuid.UUID, v55.Schematic, schematic.Schematic](
							"v55_lift_typed_schematic",
							schematic.MigrateSchematic,
						),
					},
				},
			))
		}

		seedV55 := func(ctx SpecContext, db *gorp.DB, name, body string) v55.Schematic {
			t := MustOpen(gorp.OpenTable[uuid.UUID, v55.Schematic](
				ctx, gorp.TableConfig[v55.Schematic]{DB: db},
			))
			seed := v55.Schematic{Key: uuid.New(), Name: name, Data: jsonMap(body)}
			Expect(t.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())
			return seed
		}

		retrieve := func(ctx SpecContext, db *gorp.DB, t *gorp.Table[uuid.UUID, schematic.Schematic], key uuid.UUID) schematic.Schematic {
			var got schematic.Schematic
			Expect(t.NewRetrieve().
				Where(gorp.MatchKeys[schematic.Key, schematic.Schematic](key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			return got
		}

		It("Should lift a v5 wire-format blob into the typed Schematic on retrieve", func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			seed := seedV55(ctx, db, "Tank Farm", `{
				"version": "5.0.0",
				"authority": 7,
				"nodes": [{"key": "n1", "position": {"x": 100, "y": 200}}],
				"edges": [{"key": "e1", "source": "n1", "target": "n2", "sourceHandle": "outlet", "targetHandle": "inlet"}],
				"props": {"n1": {"key": "tank", "color": "#0080ff"}},
				"legend": {"visible": true, "position": {"x": 50, "y": 50, "units": {"x": "px", "y": "px"}}, "colors": {}}
			}`)
			got := retrieve(ctx, db, openMigratedTable(ctx, db), seed.Key)
			Expect(got.Authority).To(BeEquivalentTo(7))
			Expect(got.Edges[0].Source).To(Equal(schematic.Handle{Node: "n1", Param: "outlet"}))
			Expect(got.Props["n1"]["variant"]).To(Equal("tank"))
		})

		It("Should chain a legacy v0 blob through every migration step on retrieve", func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			seed := seedV55(ctx, db, "Legacy", `{
				"version": "0.0.0",
				"nodes": [{"key": "n1", "position": {"x": 0, "y": 0}}],
				"edges": [{"key": "e1", "source": "n1", "target": "n2", "sourceHandle": "out", "targetHandle": "in"}],
				"props": {"n1": {"key": "valve"}}
			}`)
			got := retrieve(ctx, db, openMigratedTable(ctx, db), seed.Key)
			Expect(got.Edges[0].Source).To(Equal(schematic.Handle{Node: "n1", Param: "out"}))
			Expect(got.Authority).To(BeEquivalentTo(1))
			Expect(got.Props["n1"]["variant"]).To(Equal("valve"))
		})
	})

	// Each spec uses a v5-shaped blob and asserts a single reshape rule from
	// the v6 console contract. Keep one concern per spec so failures localize.
	Describe("v5 reshape semantics", func() {
		migrateV5 := func(ctx SpecContext, body string) schematic.Schematic {
			return MustSucceed(schematic.MigrateSchematic(ctx, v55.Schematic{
				Key:  uuid.New(),
				Data: jsonMap(`{"version": "5.0.0", "nodes": [], "edges": [], "props": {}, ` + body + `}`),
			}))
		}

		It("Should reshape edge endpoints into nested Handle{Node, Param}", func(ctx SpecContext) {
			out := MustSucceed(schematic.MigrateSchematic(ctx, v55.Schematic{
				Key: uuid.New(),
				Data: jsonMap(`{
					"version": "5.0.0",
					"nodes": [], "props": {},
					"edges": [{"key": "e1", "source": "n1", "target": "n2", "sourceHandle": "outlet", "targetHandle": "inlet"}]
				}`),
			}))
			Expect(out.Edges[0].Source).To(Equal(schematic.Handle{Node: "n1", Param: "outlet"}))
			Expect(out.Edges[0].Target).To(Equal(schematic.Handle{Node: "n2", Param: "inlet"}))
		})

		It("Should lift edge.data segments, color, and variant into props keyed by edge id", func(ctx SpecContext) {
			out := MustSucceed(schematic.MigrateSchematic(ctx, v55.Schematic{
				Key: uuid.New(),
				Data: jsonMap(`{
					"version": "5.0.0",
					"nodes": [], "props": {},
					"edges": [{
						"key": "e1", "source": "n1", "target": "n2",
						"data": {"segments": [{"direction": "x", "length": 30}], "color": "#0000ff", "variant": "pipe"}
					}]
				}`),
			}))
			Expect(out.Props["e1"]).To(SatisfyAll(
				HaveKeyWithValue("variant", "pipe"),
				HaveKeyWithValue("color", "#0000ff"),
				HaveKey("segments"),
			))
		})

		It("Should default edge-prop variant to pipe when edge.data is non-null but empty", func(ctx SpecContext) {
			out := MustSucceed(schematic.MigrateSchematic(ctx, v55.Schematic{
				Key: uuid.New(),
				Data: jsonMap(`{
					"version": "5.0.0",
					"nodes": [], "props": {},
					"edges": [{"key": "e1", "source": "n1", "target": "n2", "data": {}}]
				}`),
			}))
			Expect(out.Props["e1"]["variant"]).To(Equal("pipe"))
		})

		It("Should produce no edge-prop entry when edge.data is missing or null", func(ctx SpecContext) {
			out := MustSucceed(schematic.MigrateSchematic(ctx, v55.Schematic{
				Key: uuid.New(),
				Data: jsonMap(`{
					"version": "5.0.0",
					"nodes": [], "props": {},
					"edges": [
						{"key": "missing", "source": "n1", "target": "n2"},
						{"key": "null", "source": "n1", "target": "n2", "data": null}
					]
				}`),
			}))
			Expect(out.Props).NotTo(HaveKey("missing"))
			Expect(out.Props).NotTo(HaveKey("null"))
		})

		It("Should rename node-prop key to variant", func(ctx SpecContext) {
			out := migrateV5(ctx, `"props": {"n1": {"key": "valve", "color": "#ff0000"}}`)
			Expect(out.Props["n1"]).To(SatisfyAll(
				HaveKeyWithValue("variant", "valve"),
				HaveKeyWithValue("color", "#ff0000"),
				Not(HaveKey("key")),
			))
		})

		It("Should overwrite an existing variant with the v0 key field per console v6 contract", func(ctx SpecContext) {
			out := migrateV5(ctx, `"props": {"n1": {"key": "tank", "variant": "stale"}}`)
			Expect(out.Props["n1"]["variant"]).To(Equal("tank"))
		})

		It("Should default authority to 1 when the blob carries zero", func(ctx SpecContext) {
			out := migrateV5(ctx, `"authority": 0`)
			Expect(out.Authority).To(BeEquivalentTo(1))
		})

		It("Should preserve user-set zIndex on nodes", func(ctx SpecContext) {
			out := MustSucceed(schematic.MigrateSchematic(ctx, v55.Schematic{
				Key: uuid.New(),
				Data: jsonMap(`{
					"version": "5.0.0", "edges": [], "props": {},
					"nodes": [
						{"key": "back", "position": {"x": 0, "y": 0}, "zIndex": -1},
						{"key": "front", "position": {"x": 0, "y": 0}, "zIndex": 7}
					]
				}`),
			}))
			Expect(out.Nodes[0].ZIndex).To(BeEquivalentTo(-1))
			Expect(out.Nodes[1].ZIndex).To(BeEquivalentTo(7))
		})

		It("Should default zIndex to 0 when the wire form omits it", func(ctx SpecContext) {
			out := MustSucceed(schematic.MigrateSchematic(ctx, v55.Schematic{
				Key: uuid.New(),
				Data: jsonMap(`{
					"version": "5.0.0", "edges": [], "props": {},
					"nodes": [{"key": "n1", "position": {"x": 0, "y": 0}}]
				}`),
			}))
			Expect(out.Nodes[0].ZIndex).To(BeEquivalentTo(0))
		})

		It("Should pass legend visible and position through unchanged", func(ctx SpecContext) {
			out := migrateV5(ctx, `"legend": {"visible": true, "position": {"x": 75, "y": 25, "units": {"x": "px", "y": "px"}}, "colors": {}}`)
			Expect(out.Legend.Visible).To(BeTrue())
			Expect(out.Legend.Position.X).To(Equal(75.0))
			Expect(out.Legend.Position.Y).To(Equal(25.0))
		})

		It("Should pass through the gorp-entry fields (key, name, snapshot)", func(ctx SpecContext) {
			key := uuid.New()
			out := MustSucceed(schematic.MigrateSchematic(ctx, v55.Schematic{
				Key: key, Name: "tank-1", Snapshot: true,
				Data: jsonMap(`{"version": "5.0.0"}`),
			}))
			Expect(out.Key).To(Equal(key))
			Expect(out.Name).To(Equal("tank-1"))
			Expect(out.Snapshot).To(BeTrue())
		})

		It("Should handle a nil data blob without erroring", func(ctx SpecContext) {
			out := MustSucceed(schematic.MigrateSchematic(ctx, v55.Schematic{
				Key: uuid.New(), Name: "empty", Data: nil,
			}))
			Expect(out.Nodes).To(BeEmpty())
			Expect(out.Edges).To(BeEmpty())
			Expect(out.Authority).To(BeEquivalentTo(1))
		})
	})
})

var _ = Describe("legacy.MigrateData", func() {
	// Walk each captured production fixture through the chain and assert
	// invariants: counts, edge.data preservation, orphan filter, dispatch.
	Describe("real-world fixtures", func() {
		DescribeTable("Should walk the chain to v5.Data, preserving edge.data and dropping orphans",
			func(fixture string, expectNodes, expectEdges, expectInputOrphans int) {
				blob, raw := loadFixture(fixture)
				rawNodes, _ := raw["nodes"].([]any)
				rawEdges, _ := raw["edges"].([]any)
				Expect(rawNodes).To(HaveLen(expectNodes))
				Expect(rawEdges).To(HaveLen(expectEdges))

				validRawEdges := make([]map[string]any, 0, len(rawEdges))
				for _, re := range rawEdges {
					em, _ := re.(map[string]any)
					if stringOr(em["source"]) == "" || stringOr(em["target"]) == "" {
						continue
					}
					validRawEdges = append(validRawEdges, em)
				}
				Expect(len(rawEdges) - len(validRawEdges)).To(Equal(expectInputOrphans))

				out := MustSucceed(legacy.MigrateData(blob))
				Expect(out.Version).To(Equal(v5.Version))
				Expect(out.Nodes).To(HaveLen(len(rawNodes)))
				Expect(out.Edges).To(HaveLen(len(validRawEdges)))

				for _, e := range out.Edges {
					Expect(e.Source).NotTo(BeEmpty(), "orphan edge survived")
					Expect(e.Target).NotTo(BeEmpty(), "orphan edge survived")
				}
				for i, e := range out.Edges {
					rawEdge := validRawEdges[i]
					Expect(e.Key).To(Equal(rawEdge["key"]))
					Expect(e.Source).To(Equal(stringOr(rawEdge["source"])))
					Expect(e.Target).To(Equal(stringOr(rawEdge["target"])))
					if data, ok := rawEdge["data"].(map[string]any); ok && len(data) > 0 {
						Expect(e.Data).NotTo(BeEmpty(),
							"edge %v lost ReactFlow data bag through chain", e.Key)
					}
				}
			},
			Entry("v2 condensed (52 nodes, 39 edges, edge.data preserved)",
				"v2_gse_condensed.json", 52, 39, 0),
			Entry("v2 value-only (3 nodes, 7 props, no edges)",
				"v2_value_test.json", 3, 0, 0),
			Entry("v3 shop flow (42 nodes, 29 edges incl. 1 orphan dropped)",
				"v3_shop_flow.json", 42, 29, 1),
			Entry("v4 empty (version dispatch only)",
				"v4_empty.json", 0, 0, 0),
			Entry("v5 hardware workspace (real mode/toolbar/authority)",
				"v5_hardware_workspace.json", 2, 0, 0),
		)
	})

	// Synthesized inputs cover the chain semantics that real fixtures don't
	// exercise: bottom of the chain (v0), version dispatch edge cases,
	// orphan filtering, and error paths.
	Describe("synthesized inputs", func() {
		It("Should chain a v0 blob through every step migration", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{
				"version": "0.0.0",
				"nodes": [{"key": "n1", "position": {"x": 1, "y": 2}}],
				"edges": [{"key": "e1", "source": "n1", "target": "n2", "sourceHandle": "out", "targetHandle": "in"}],
				"props": {"n1": {"key": "valve"}}
			}`)))
			Expect(out.Version).To(Equal(v5.Version))
			Expect(out.Authority).To(BeEquivalentTo(1))
			Expect(out.Mode).To(Equal("select"))
			Expect(out.Legend.Visible).To(BeTrue())
		})

		It("Should fall back to v0 when the blob has no version field", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{"nodes": [], "edges": [], "props": {}}`)))
			Expect(out.Version).To(Equal(v5.Version))
		})

		It("Should preserve user-set zIndex on nodes through the chain", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{
				"version": "0.0.0",
				"nodes": [
					{"key": "back", "position": {"x": 0, "y": 0}, "zIndex": -1},
					{"key": "front", "position": {"x": 0, "y": 0}, "zIndex": 7}
				],
				"edges": [], "props": {}
			}`)))
			Expect(*out.Nodes[0].ZIndex).To(Equal(-1))
			Expect(*out.Nodes[1].ZIndex).To(Equal(7))
		})

		It("Should preserve edge.data through a v0 blob into v5.Edge.Data", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{
				"version": "0.0.0",
				"nodes": [], "props": {},
				"edges": [{
					"key": "e1", "source": "n1", "target": "n2",
					"data": {"segments": [{"direction": "x", "length": 10}], "color": "#ff0000"}
				}]
			}`)))
			Expect(out.Edges[0].Data).NotTo(BeEmpty())
		})

		It("Should drop edges with empty source", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{
				"version": "5.0.0",
				"nodes": [], "props": {},
				"edges": [
					{"key": "good", "source": "n1", "target": "n2"},
					{"key": "orphan", "source": "", "target": "n2"}
				]
			}`)))
			Expect(out.Edges).To(HaveLen(1))
			Expect(out.Edges[0].Key).To(Equal("good"))
		})

		It("Should drop edges with null source or target", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{
				"version": "5.0.0",
				"nodes": [], "props": {},
				"edges": [
					{"key": "orphan-a", "source": null, "target": "n2"},
					{"key": "orphan-b", "source": "n1", "target": null}
				]
			}`)))
			Expect(out.Edges).To(BeEmpty())
		})

		It("Should walk the chain on a nil blob and produce a zero v5.Data", func() {
			out := MustSucceed(legacy.MigrateData(nil))
			Expect(out.Version).To(Equal(v5.Version))
			Expect(out.Nodes).To(BeEmpty())
			Expect(out.Edges).To(BeEmpty())
		})

		It("Should error on an unknown declared version", func() {
			Expect(legacy.MigrateData(jsonMap(`{"version": "99.0.0"}`))).Error().
				To(MatchError(ContainSubstring("unknown schematic data version")))
		})
	})
})

// Each step is fed nonZeroV0() chained up to its input version. Tests assert
// the step's *new* fields and that every prior field passes through unchanged.

var _ = Describe("Step migrations", func() {
	Describe("v1.Migrate (v0 -> v1)", func() {
		It("Should attach the default legend", func() {
			out := v1.Migrate(nonZeroV0())
			Expect(out.Version).To(Equal(v1.Version))
			Expect(out.Legend).To(Equal(v1.ZeroLegend))
		})

		It("Should pass every v0 field through unchanged", func() {
			in := nonZeroV0()
			out := v1.Migrate(in)
			Expect(out.Editable).To(Equal(in.Editable))
			Expect(out.FitViewOnResize).To(Equal(in.FitViewOnResize))
			Expect(out.Snapshot).To(Equal(in.Snapshot))
			Expect(out.RemoteCreated).To(Equal(in.RemoteCreated))
			Expect(out.Viewport).To(Equal(in.Viewport))
			Expect(out.Nodes).To(Equal(in.Nodes))
			Expect(out.Edges).To(Equal(in.Edges))
			Expect(out.Props).To(Equal(in.Props))
			Expect(out.Control).To(Equal(in.Control))
		})

		It("Should produce a legend whose units default to px when no legend exists upstream", func() {
			out := v1.Migrate(v0.Data{})
			Expect(out.Legend.Visible).To(BeTrue())
			Expect(out.Legend.Position).To(Equal(v1.LegendPosition{
				X: 50, Y: 50,
				Units: &v1.LegendUnits{X: "px", Y: "px"},
			}))
			Expect(out.Legend.Colors).To(BeEmpty())
		})
	})

	Describe("v2.Migrate (v1 -> v2)", func() {
		It("Should add the schematic type literal and the default viewport mode", func() {
			out := v2.Migrate(v1.Migrate(nonZeroV0()))
			Expect(out.Version).To(Equal(v2.Version))
			Expect(out.Type).To(Equal("schematic"))
			Expect(out.ViewportMode).To(Equal("select"))
		})

		It("Should generate a fresh uuid key on every call", func() {
			in := v1.Migrate(v0.Data{})
			a := v2.Migrate(in)
			b := v2.Migrate(in)
			Expect(a.Key).NotTo(BeEmpty())
			Expect(b.Key).NotTo(BeEmpty())
			Expect(a.Key).NotTo(Equal(b.Key))
		})

		It("Should pass every v1 field through unchanged", func() {
			in := v1.Migrate(nonZeroV0())
			out := v2.Migrate(in)
			Expect(out.Editable).To(Equal(in.Editable))
			Expect(out.FitViewOnResize).To(Equal(in.FitViewOnResize))
			Expect(out.Snapshot).To(Equal(in.Snapshot))
			Expect(out.RemoteCreated).To(Equal(in.RemoteCreated))
			Expect(out.Viewport).To(Equal(in.Viewport))
			Expect(out.Nodes).To(Equal(in.Nodes))
			Expect(out.Edges).To(Equal(in.Edges))
			Expect(out.Props).To(Equal(in.Props))
			Expect(out.Control).To(Equal(in.Control))
			Expect(out.Legend).To(Equal(in.Legend))
		})
	})

	Describe("v3.Migrate (v2 -> v3)", func() {
		It("Should attach an empty segments slice to every edge", func() {
			out := v3.Migrate(v2.Migrate(v1.Migrate(nonZeroV0())))
			Expect(out.Version).To(Equal(v3.Version))
			for _, e := range out.Edges {
				Expect(e.Segments).NotTo(BeNil())
				Expect(e.Segments).To(BeEmpty())
			}
		})

		It("Should preserve edge.Data so v6 can lift segments/color/variant", func() {
			in := v2.Migrate(v1.Migrate(nonZeroV0()))
			out := v3.Migrate(in)
			Expect(out.Edges).To(HaveLen(len(in.Edges)))
			for i, e := range out.Edges {
				Expect(e.Data).To(Equal(in.Edges[i].Data))
			}
		})

		It("Should pass non-edge fields through unchanged", func() {
			in := v2.Migrate(v1.Migrate(nonZeroV0()))
			out := v3.Migrate(in)
			Expect(out.Nodes).To(Equal(in.Nodes))
			Expect(out.Props).To(Equal(in.Props))
			Expect(out.Key).To(Equal(in.Key))
			Expect(out.Type).To(Equal(in.Type))
			Expect(out.ViewportMode).To(Equal(in.ViewportMode))
			Expect(out.Legend).To(Equal(in.Legend))
		})
	})

	Describe("v4.Migrate (v3 -> v4)", func() {
		It("Should set authority to 1", func() {
			out := v4.Migrate(v3.Migrate(v2.Migrate(v1.Migrate(nonZeroV0()))))
			Expect(out.Version).To(Equal(v4.Version))
			Expect(out.Authority).To(Equal(1.0))
		})

		It("Should pass every v3 field through unchanged", func() {
			in := v3.Migrate(v2.Migrate(v1.Migrate(nonZeroV0())))
			out := v4.Migrate(in)
			Expect(out.Edges).To(Equal(in.Edges))
			Expect(out.Nodes).To(Equal(in.Nodes))
			Expect(out.Props).To(Equal(in.Props))
			Expect(out.Legend).To(Equal(in.Legend))
			Expect(out.Key).To(Equal(in.Key))
			Expect(out.Type).To(Equal(in.Type))
			Expect(out.ViewportMode).To(Equal(in.ViewportMode))
		})
	})

	Describe("v5.Migrate (v4 -> v5)", func() {
		It("Should drop the type literal and seed default mode and toolbar", func() {
			out := v5.Migrate(v4.Migrate(v3.Migrate(v2.Migrate(v1.Migrate(nonZeroV0())))))
			Expect(out.Version).To(Equal(v5.Version))
			Expect(out.Mode).To(Equal("select"))
			Expect(out.Toolbar).To(Equal(v0.ToolbarState{
				ActiveTab:           "symbols",
				SelectedSymbolGroup: "general",
			}))
		})

		It("Should pass every v4 field through unchanged", func() {
			in := v4.Migrate(v3.Migrate(v2.Migrate(v1.Migrate(nonZeroV0()))))
			out := v5.Migrate(in)
			Expect(out.Authority).To(Equal(in.Authority))
			Expect(out.Edges).To(Equal(in.Edges))
			Expect(out.Nodes).To(Equal(in.Nodes))
			Expect(out.Props).To(Equal(in.Props))
			Expect(out.Legend).To(Equal(in.Legend))
			Expect(out.Key).To(Equal(in.Key))
			Expect(out.ViewportMode).To(Equal(in.ViewportMode))
		})
	})
})
