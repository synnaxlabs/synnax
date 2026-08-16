// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v7_test

import (
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy"
	legacyv6 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v6"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v0"
	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

//go:embed testdata/*.json
var fixtures embed.FS

func loadFixture(name string) (msgpack.EncodedJSON, map[string]any) {
	GinkgoHelper()
	raw := MustSucceed(fixtures.ReadFile("testdata/" + name))
	var m map[string]any
	Expect(json.Unmarshal(raw, &m)).To(Succeed())
	return msgpack.EncodedJSON(m), m
}

func jsonMap(raw string) msgpack.EncodedJSON {
	GinkgoHelper()
	var m map[string]any
	Expect(json.Unmarshal([]byte(raw), &m)).To(Succeed())
	return m
}

// migrateSeed runs the v7 migration chain over a gorp-seeded v6 schematic and returns
// the migrated typed Schematic.
func migrateSeed(ctx SpecContext, seed v0.Schematic) v7.Schematic {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	MustSucceed(gorp.OpenTable(
		ctx, gorp.TableConfig[v0.Key, v0.Schematic]{DB: db},
	))
	Expect(gorp.NewCreate[v0.Key, v0.Schematic]().
		Entry(&seed).Exec(ctx, db)).To(Succeed())
	Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:         db,
		Namespace:  "Schematic",
		Migrations: []migrate.Migration{v0.Migration, v7.Migration},
	})).To(Succeed())
	var got v7.Schematic
	Expect(gorp.NewRetrieve[v7.Key, v7.Schematic]().
		Where(gorp.MatchKeys[v7.Key, v7.Schematic](seed.Key)).
		Entry(&got).Exec(ctx, db)).To(Succeed())
	return got
}

// variantOf returns the decoded union variant stored under key, failing the spec when
// the key is absent.
func variantOf(s v7.Schematic, key string) v7.ElementConfigVariant {
	GinkgoHelper()
	cfg, ok := s.Configs[key]
	Expect(ok).To(BeTrue(), "no config stored under %q", key)
	return cfg.Variant
}

// segmentsOf returns the routed segments of the segmented edge config stored under key.
func segmentsOf(s v7.Schematic, key string) []v7.Segment {
	GinkgoHelper()
	switch cfg := variantOf(s, key).(type) {
	case v7.PipeElementConfig:
		return cfg.Segments
	case v7.JacketedElementConfig:
		return cfg.Segments
	default:
		Fail("config is not a segmented edge")
		return nil
	}
}

func stringOr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// assertMigrated compares got against the canonical .migrated.json file for fixture, or
// rewrites it if UPDATE_MIGRATED=1 is set. Outputs are canonicalized via
// json.MarshalIndent (which sorts map keys) so diffs are deterministic.
func assertMigrated(fixture string, got v7.Schematic) {
	GinkgoHelper()
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

var _ = Describe("MigrateSchematic", func() {
	// Snapshot tests against the canonical .migrated.json output for every captured
	// production fixture. Run with UPDATE_MIGRATED=1 to regenerate the .migrated.json
	// files after intentional migration changes.
	Describe("canonical migrated output", func() {
		fixedKey := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		DescribeTable("Should produce the canonical typed Schematic",
			func(ctx SpecContext, fixture string) {
				blob, _ := loadFixture(fixture)
				out := migrateSeed(ctx, v0.Schematic{
					Key: fixedKey, Name: fixture, Data: blob,
				})
				assertMigrated(fixture, out)
			},
			Entry("v2 condensed", "v2_gse_condensed.json"),
			Entry("v2 value-only", "v2_value_test.json"),
			Entry("v4 empty", "v4_empty.json"),
			Entry("v5 hardware workspace", "v5_hardware_workspace.json"),
			Entry("v5 operator console", "v5_operator.json"),
		)

		It(
			"Should produce the canonical output when called directly",
			func(ctx SpecContext) {
				blob, _ := loadFixture("v5_operator.json")
				out := migrateSeed(ctx, v0.Schematic{
					Key: fixedKey, Name: "v5_operator.json", Data: blob,
				})
				assertMigrated("v5_operator.json", out)
			},
		)
	})

	Describe("storage integration", func() {
		It(
			"Should lift a v5 wire-format blob into the typed Schematic on retrieve",
			func(ctx SpecContext) {
				got := migrateSeed(ctx, v0.Schematic{
					Key: uuid.New(), Name: "Tank Farm", Data: jsonMap(`{
					"version": "5.0.0",
					"authority": 7,
					"nodes": [{"key": "n1", "position": {"x": 100, "y": 200}}],
					"edges": [{"key": "e1", "source": "n1", "target": "n2", "sourceHandle": "outlet", "targetHandle": "inlet"}],
					"props": {"n1": {"key": "tank", "color": "#0080ff"}},
					"legend": {"visible": true, "position": {"x": 50, "y": 50, "units": {"x": "px", "y": "px"}}, "colors": {}}
				}`),
				})
				Expect(
					got.Edges[0].Source,
				).To(Equal(v7.Handle{Node: "n1", Param: "outlet"}))
				Expect(variantOf(got, "n1")).To(BeAssignableToTypeOf(
					v7.TankElementConfig{},
				))
			},
		)

		It(
			"Should chain a legacy v0 blob through every migration step on retrieve",
			func(ctx SpecContext) {
				got := migrateSeed(ctx, v0.Schematic{
					Key: uuid.New(), Name: "Legacy", Data: jsonMap(`{
					"version": "0.0.0",
					"nodes": [{"key": "n1", "position": {"x": 0, "y": 0}}],
					"edges": [{"key": "e1", "source": "n1", "target": "n2", "sourceHandle": "out", "targetHandle": "in"}],
					"props": {"n1": {"key": "valve"}}
				}`),
				})
				Expect(
					got.Edges[0].Source,
				).To(Equal(v7.Handle{Node: "n1", Param: "out"}))
				Expect(variantOf(got, "n1")).To(BeAssignableToTypeOf(
					v7.ValveElementConfig{},
				))
			},
		)
	})

	// Each spec uses a v5-shaped blob and asserts a single reshape rule from the v6
	// Console contract. Keep one concern per spec so failures localize.
	Describe("v5 reshape semantics", func() {
		migrateV5 := func(ctx SpecContext, body string) v7.Schematic {
			return migrateSeed(ctx, v0.Schematic{
				Key: uuid.New(),
				Data: jsonMap(
					`{"version": "5.0.0", "nodes": [], "edges": [], "props": {}, ` + body + `}`,
				),
			})
		}

		It(
			"Should reshape edge endpoints into nested Handle{Node, Param}",
			func(ctx SpecContext) {
				out := migrateSeed(ctx, v0.Schematic{
					Key: uuid.New(),
					Data: jsonMap(`{
					"version": "5.0.0",
					"nodes": [], "props": {},
					"edges": [{"key": "e1", "source": "n1", "target": "n2", "sourceHandle": "outlet", "targetHandle": "inlet"}]
				}`),
				})
				Expect(
					out.Edges[0].Source,
				).To(Equal(v7.Handle{Node: "n1", Param: "outlet"}))
				Expect(
					out.Edges[0].Target,
				).To(Equal(v7.Handle{Node: "n2", Param: "inlet"}))
			},
		)

		It(
			"Should lift edge.data segments, color, and variant into props keyed by edge id",
			func(ctx SpecContext) {
				out := migrateSeed(ctx, v0.Schematic{
					Key: uuid.New(),
					Data: jsonMap(`{
					"version": "5.0.0",
					"nodes": [], "props": {},
					"edges": [{
						"key": "e1", "source": "n1", "target": "n2",
						"data": {"segments": [{"direction": "x", "length": 30}], "color": "#0000ff", "variant": "pipe"}
					}]
				}`),
				})
				Expect(variantOf(out, "e1")).To(Equal(v7.PipeElementConfig{
					SegmentedEdgeConfig: v7.SegmentedEdgeConfig{
						Color:    new(MustSucceed(color.FromHex("#0000ff"))),
						Segments: []v7.Segment{{Direction: "x", Length: 10}},
					},
				}))
			},
		)

		It("Should strip legacy stumps from a full-path edge", func(ctx SpecContext) {
			// Real OX Pre-Valve -> OX MPV edge from a 0.55 schematic: the stored full
			// path includes both stumps, which would double on render and fold a
			// pigtail.
			out := migrateSeed(ctx, v0.Schematic{
				Key: uuid.New(),
				Data: jsonMap(`{
					"version": "5.0.0", "nodes": [], "props": {},
					"edges": [{
						"key": "e1", "source": "n1", "target": "n2",
						"sourceHandle": "2", "targetHandle": "2",
						"data": {"segments": [
							{"direction": "x", "length": 10},
							{"direction": "y", "length": -281.7166395035551},
							{"direction": "x", "length": 140.06190790464655},
							{"direction": "y", "length": 10}
						], "variant": "jacketed"}
					}]
				}`),
			})
			Expect(segmentsOf(out, "e1")).To(Equal([]v7.Segment{
				{Direction: "y", Length: -281.7166395035551},
				{Direction: "x", Length: 140.06190790464655},
			}))
		})

		It(
			"Should clear degenerate short edges so they auto-route",
			func(ctx SpecContext) {
				// A single segment shorter than two stumps (real 0.55 edge, 11.88px)
				// has no strippable middle; subtracting a full stump from each end
				// would flip it into a self-crossing spur, so it is cleared to an empty
				// (auto-routed) edge.
				out := migrateSeed(ctx, v0.Schematic{
					Key: uuid.New(),
					Data: jsonMap(`{
					"version": "5.0.0", "nodes": [], "props": {},
					"edges": [{"key": "e1", "source": "n1", "target": "n2",
						"data": {"segments": [{"direction": "y", "length": 11.88}], "variant": "pipe"}}]
				}`),
				})
				Expect(segmentsOf(out, "e1")).To(BeEmpty())
			},
		)

		It(
			"Should default edge-prop variant to pipe when edge.data is non-null but empty",
			func(ctx SpecContext) {
				out := migrateSeed(ctx, v0.Schematic{
					Key: uuid.New(),
					Data: jsonMap(`{
					"version": "5.0.0",
					"nodes": [], "props": {},
					"edges": [{"key": "e1", "source": "n1", "target": "n2", "data": {}}]
				}`),
				})
				Expect(variantOf(out, "e1")).To(BeAssignableToTypeOf(
					v7.PipeElementConfig{},
				))
			},
		)

		It(
			"Should produce no edge-prop entry when edge.data is missing or null",
			func(ctx SpecContext) {
				out := migrateSeed(ctx, v0.Schematic{
					Key: uuid.New(),
					Data: jsonMap(`{
					"version": "5.0.0",
					"nodes": [], "props": {},
					"edges": [
						{"key": "missing", "source": "n1", "target": "n2"},
						{"key": "null", "source": "n1", "target": "n2", "data": null}
					]
				}`),
				})
				Expect(out.Configs).NotTo(HaveKey("missing"))
				Expect(out.Configs).NotTo(HaveKey("null"))
			},
		)

		It("Should rename node-prop key to variant", func(ctx SpecContext) {
			out := migrateV5(
				ctx,
				`"props": {"n1": {"key": "valve", "color": "#ff0000"}}`,
			)
			Expect(variantOf(out, "n1")).To(Equal(v7.ValveElementConfig{
				ToggleSymbolConfig: v7.ToggleSymbolConfig{
					Color: new(MustSucceed(color.FromHex("#ff0000"))),
				},
			}))
		})

		It(
			"Should overwrite an existing variant with the v0 key field per console v6 contract",
			func(ctx SpecContext) {
				out := migrateV5(
					ctx,
					`"props": {"n1": {"key": "tank", "variant": "stale"}}`,
				)
				Expect(variantOf(out, "n1")).To(BeAssignableToTypeOf(
					v7.TankElementConfig{},
				))
			},
		)

		It("Should preserve user-set zIndex on nodes", func(ctx SpecContext) {
			out := migrateSeed(ctx, v0.Schematic{
				Key: uuid.New(),
				Data: jsonMap(`{
					"version": "5.0.0", "edges": [], "props": {},
					"nodes": [
						{"key": "back", "position": {"x": 0, "y": 0}, "zIndex": -1},
						{"key": "front", "position": {"x": 0, "y": 0}, "zIndex": 7}
					]
				}`),
			})
			Expect(out.Nodes[0].ZIndex).To(BeEquivalentTo(-1))
			Expect(out.Nodes[1].ZIndex).To(BeEquivalentTo(7))
		})

		It(
			"Should default zIndex to 0 when the wire form omits it",
			func(ctx SpecContext) {
				out := migrateSeed(ctx, v0.Schematic{
					Key: uuid.New(),
					Data: jsonMap(`{
					"version": "5.0.0", "edges": [], "props": {},
					"nodes": [{"key": "n1", "position": {"x": 0, "y": 0}}]
				}`),
				})
				Expect(out.Nodes[0].ZIndex).To(BeEquivalentTo(0))
			},
		)

		It(
			"Should pass through the gorp-entry fields (key, name, snapshot)",
			func(ctx SpecContext) {
				key := uuid.New()
				out := migrateSeed(ctx, v0.Schematic{
					Key: key, Name: "tank-1", Snapshot: true,
					Data: jsonMap(`{"version": "5.0.0"}`),
				})
				Expect(out.Key).To(Equal(key))
				Expect(out.Name).To(Equal("tank-1"))
				Expect(out.Snapshot).To(BeTrue())
			},
		)

		It("Should handle a nil data blob without erroring", func(ctx SpecContext) {
			out := migrateSeed(ctx, v0.Schematic{
				Key: uuid.New(), Name: "empty", Data: nil,
			})
			Expect(out.Nodes).To(BeEmpty())
			Expect(out.Edges).To(BeEmpty())
		})
	})
})

var _ = Describe("MigrateData", func() {
	// Walk each captured production fixture through the chain and assert invariants:
	// counts, edge.data preservation, orphan filter, dispatch.
	Describe("real-world fixtures", func() {
		DescribeTable(
			"Should walk the chain to legacy.Data, preserving edge.data and dropping orphans",
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
				Expect(out.Version).To(Equal(legacy.DataVersion))
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
					if data, ok := rawEdge["data"].(map[string]any); ok &&
						len(data) > 0 {
						Expect(e.Data).NotTo(BeEmpty(),
							"edge %v lost ReactFlow data bag through chain", e.Key)
					}
				}
			},
			Entry("v2 condensed (52 nodes, 39 edges, edge.data preserved)",
				"v2_gse_condensed.json", 52, 39, 0),
			Entry("v2 value-only (3 nodes, 7 props, no edges)",
				"v2_value_test.json", 3, 0, 0),
			Entry("v4 empty (version dispatch only)",
				"v4_empty.json", 0, 0, 0),
			Entry("v5 hardware workspace (real mode/toolbar/authority)",
				"v5_hardware_workspace.json", 2, 0, 0),
			Entry("v5 operator console (48 nodes, 40 edges, edge.data preserved)",
				"v5_operator.json", 48, 40, 0),
		)
	})

	// Synthesized inputs cover the chain semantics that real fixtures don't exercise:
	// bottom of the chain (v0), version dispatch edge cases, orphan filtering, and
	// error paths.
	Describe("synthesized inputs", func() {
		It("Should chain a v0 blob through every step migration", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{
				"version": "0.0.0",
				"nodes": [{"key": "n1", "position": {"x": 1, "y": 2}}],
				"edges": [{"key": "e1", "source": "n1", "target": "n2", "sourceHandle": "out", "targetHandle": "in"}],
				"props": {"n1": {"key": "valve"}}
			}`)))
			Expect(out.Version).To(Equal(legacy.DataVersion))
			Expect(out.Nodes).To(HaveLen(1))
			Expect(out.Edges).To(HaveLen(1))
			Expect(out.Props).To(HaveKey("n1"))
		})

		It("Should fall back to v0 when the blob has no version field", func() {
			out := MustSucceed(
				legacy.MigrateData(jsonMap(`{"nodes": [], "edges": [], "props": {}}`)),
			)
			Expect(out.Version).To(Equal(legacy.DataVersion))
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

		It(
			"Should preserve edge.data through a v0 blob into legacy.Edge.Data",
			func() {
				out := MustSucceed(legacy.MigrateData(jsonMap(`{
				"version": "0.0.0",
				"nodes": [], "props": {},
				"edges": [{
					"key": "e1", "source": "n1", "target": "n2",
					"data": {"segments": [{"direction": "x", "length": 10}], "color": "#ff0000"}
				}]
			}`)))
				Expect(out.Edges[0].Data).NotTo(BeEmpty())
			},
		)

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

		It(
			"Should walk the chain on a nil blob and produce a zero legacy.Data",
			func() {
				out := MustSucceed(legacy.MigrateData(nil))
				Expect(out.Version).To(Equal(legacy.DataVersion))
				Expect(out.Nodes).To(BeEmpty())
				Expect(out.Edges).To(BeEmpty())
			},
		)

		It("Should error on an unknown declared version", func() {
			Expect(legacy.MigrateData(jsonMap(`{"version": "99.0.0"}`))).Error().
				To(MatchError(ContainSubstring("unknown schematic data version")))
		})
	})
})

// Each step is fed nonZeroV0() chained up to its input version. Tests assert the step's
// *new* fields and that every prior field passes through unchanged.

var _ = Describe("SchematicFromConsole", func() {
	It("Should lift the Console export into the current Schematic", func() {
		out := v7.SchematicFromConsole(legacyv6.Data{
			Snapshot: true,
			Nodes: []legacyv6.Node{
				{Key: "n1", Position: spatial.XY{X: 1, Y: 2}, ZIndex: 4},
			},
			Edges: []legacyv6.Edge{{
				Key:    "e1",
				Source: legacyv6.Handle{Node: "n1", Param: "out"},
				Target: legacyv6.Handle{Node: "n2", Param: "in"},
			}},
			Configs: map[string]msgpack.EncodedJSON{"n1": {"variant": "valve"}},
		})

		Expect(out.Snapshot).To(BeTrue())
		Expect(out.Nodes).To(Equal([]v7.Node{
			{Key: "n1", Position: spatial.XY{X: 1, Y: 2}, ZIndex: 4},
		}))
		Expect(out.Edges).To(Equal([]v7.Edge{{
			Key:    "e1",
			Source: v7.Handle{Node: "n1", Param: "out"},
			Target: v7.Handle{Node: "n2", Param: "in"},
		}}))
		Expect(variantOf(out, "n1")).To(BeAssignableToTypeOf(
			v7.ValveElementConfig{},
		))
	})

	It("Should produce an empty Schematic from an empty export", func() {
		out := v7.SchematicFromConsole(legacyv6.Data{})

		Expect(out.Snapshot).To(BeFalse())
		Expect(out.Nodes).To(BeEmpty())
		Expect(out.Edges).To(BeEmpty())
		Expect(out.Configs).To(BeEmpty())
	})
})
