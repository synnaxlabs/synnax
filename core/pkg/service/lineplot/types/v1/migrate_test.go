// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/types/v0"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/types/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/text"
)

//go:embed testdata/*.json
var fixtures embed.FS

func loadFixture(name string) msgpack.EncodedJSON {
	raw := MustSucceed(fixtures.ReadFile("testdata/" + name))
	var m map[string]any
	Expect(json.Unmarshal(raw, &m)).To(Succeed())
	return msgpack.EncodedJSON(m)
}

func jsonMap(raw string) msgpack.EncodedJSON {
	var m map[string]any
	Expect(json.Unmarshal([]byte(raw), &m)).To(Succeed())
	return m
}

// assertMigrated compares got against the canonical .migrated.json file for
// fixture, or rewrites it if UPDATE_MIGRATED=1 is set. Outputs are
// canonicalized via json.MarshalIndent (which sorts map keys) so diffs are
// deterministic.
func assertMigrated(fixture string, got lineplot.LinePlot) {
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
		"%s drifted from its canonical migrated form, review the diff and rerun with UPDATE_MIGRATED=1 if intentional", fixture)
}

var _ = Describe("MigrateLinePlot", func() {
	// Snapshot tests against the canonical .migrated.json output for every
	// captured fixture. Run with UPDATE_MIGRATED=1 to regenerate the
	// .migrated.json files after intentional migration changes.
	Describe("canonical migrated output", func() {
		fixedKey := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		DescribeTable("Should produce the canonical typed LinePlot",
			func(ctx SpecContext, fixture string) {
				blob := loadFixture(fixture)
				snap := v0.LinePlot{Key: fixedKey, Name: fixture, Data: blob}
				out := MustSucceed(v1.MigrateLinePlot(ctx, snap))
				assertMigrated(fixture, out)
			},
			Entry("v0 minimal", "v0_minimal.json"),
			Entry("v2 typical", "v2_typical.json"),
			Entry("v4 full", "v4_full.json"),
		)
	})

	// Drives MigrateLinePlot through the real gorp migration pipeline so the
	// on-disk v0 -> typed LinePlot path is exercised end-to-end.
	Describe("storage integration", func() {
		openMigratedTable := func(ctx SpecContext, db *gorp.DB) *gorp.Table[uuid.UUID, lineplot.LinePlot] {
			return MustOpen(gorp.OpenTable[uuid.UUID, lineplot.LinePlot](
				ctx, gorp.TableConfig[uuid.UUID, lineplot.LinePlot]{
					DB: db,
					Migrations: []migrate.Migration{
						gorp.NewEntryMigration[uuid.UUID, uuid.UUID, v0.LinePlot, lineplot.LinePlot](
							"v55_lift_typed_lineplot",
							v1.MigrateLinePlot,
						),
					},
				},
			))
		}

		seedV55 := func(ctx SpecContext, db *gorp.DB, name, body string) v0.LinePlot {
			t := MustOpen(gorp.OpenTable[uuid.UUID, v0.LinePlot](
				ctx, gorp.TableConfig[uuid.UUID, v0.LinePlot]{DB: db},
			))
			seed := v0.LinePlot{Key: uuid.New(), Name: name, Data: jsonMap(body)}
			Expect(t.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())
			return seed
		}

		retrieve := func(ctx SpecContext, db *gorp.DB, t *gorp.Table[uuid.UUID, lineplot.LinePlot], key uuid.UUID) lineplot.LinePlot {
			var got lineplot.LinePlot
			Expect(t.NewRetrieve().
				Where(gorp.MatchKeys[lineplot.Key, lineplot.LinePlot](key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			return got
		}

		It("Should lift a v4 wire-format blob into the typed LinePlot on retrieve", func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			seed := seedV55(ctx, db, "Tank Pressure", `{
				"version": "4.0.0",
				"key": "p1",
				"remoteCreated": true,
				"title": {"level": "h4", "visible": true},
				"legend": {"visible": true, "position": {"x": 50, "y": 50, "units": {"x": "px", "y": "px"}, "root": {"x": "left", "y": "top"}}},
				"channels": {"x1": 1, "x2": 0, "y1": [10], "y2": [], "y3": [], "y4": []},
				"ranges": {"x1": ["00000000-0000-0000-0000-00000000aaaa"], "x2": []},
				"axes": {"renderTrigger": 0, "hasHadChannelSet": true, "axes": {
					"x1": {"key": "x1", "label": "t", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75, "type": "time"},
					"x2": {"key": "x2", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75, "type": "time"},
					"y1": {"key": "y1", "label": "p", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 10}, "autoBounds": {"lower": false, "upper": false}, "tickSpacing": 75},
					"y2": {"key": "y2", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y3": {"key": "y3", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y4": {"key": "y4", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75}
				}},
				"lines": [{"key": "y1-r-10", "label": "P1", "color": "#ff0000", "strokeWidth": 2, "downsample": 1, "downsampleMode": "decimate"}],
				"rules": [],
				"viewport": {"renderTrigger": 0, "zoom": {"width": 1, "height": 1}, "pan": {"x": 0, "y": 0}},
				"selection": {"box": {"one": {"x": 0, "y": 0}, "two": {"x": 0, "y": 0}, "root": {"x": "left", "y": "top"}}},
				"mode": "zoom",
				"control": {"hold": false, "clickMode": null, "enableTooltip": true},
				"toolbar": {"activeTab": "data"},
				"measure": {"mode": "one"},
				"annotations": {"visible": true}
			}`)
			got := retrieve(ctx, db, openMigratedTable(ctx, db), seed.Key)
			Expect(got.Key).To(Equal(seed.Key))
			Expect(got.Name).To(Equal("Tank Pressure"))
			Expect(got.Title.Level).To(Equal(text.LevelH4))
			Expect(got.Channels.X1).To(BeEquivalentTo(1))
			Expect(got.Channels.Y1).To(ConsistOf(channel.Key(10)))
			Expect(got.Axes.X1.Type).NotTo(BeNil())
			Expect(*got.Axes.X1.Type).To(Equal(lineplot.TickTypeTime))
			Expect(got.Axes.Y1.Type).To(BeNil())
			Expect(got.Lines).To(HaveLen(1))
			Expect(got.Lines[0].Label).NotTo(BeNil())
			Expect(*got.Lines[0].Label).To(Equal("P1"))
		})

		It("Should chain a legacy v0 blob through every migration step on retrieve", func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			seed := seedV55(ctx, db, "Legacy", `{
				"version": "0.0.0",
				"key": "p0",
				"remoteCreated": false,
				"title": {"level": "h4", "visible": false},
				"legend": {"visible": false},
				"channels": {"x1": 0, "x2": 0, "y1": [], "y2": [], "y3": [], "y4": []},
				"ranges": {"x1": [], "x2": []},
				"axes": {"renderTrigger": 0, "hasHadChannelSet": false, "axes": {
					"x1": {"key": "x1", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"x2": {"key": "x2", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y1": {"key": "y1", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y2": {"key": "y2", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y3": {"key": "y3", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y4": {"key": "y4", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75}
				}},
				"lines": [],
				"rules": []
			}`)
			got := retrieve(ctx, db, openMigratedTable(ctx, db), seed.Key)
			Expect(got.Key).To(Equal(seed.Key))
			// v1 lift forces the legend shown, so the typed Hidden is false, and sets
			// default position.
			Expect(got.Legend.Hidden).To(BeFalse())
			Expect(got.Legend.Position.X).To(Equal(50.0))
			Expect(got.Legend.Position.Units).NotTo(BeNil())
			Expect(got.Legend.Position.Units.X).To(BeEquivalentTo("px"))
			// v2 lift sets x-axes to "time" and flips y-axes labelDirection to "y".
			Expect(got.Axes.X1.Type).NotTo(BeNil())
			Expect(*got.Axes.X1.Type).To(Equal(lineplot.TickTypeTime))
			Expect(got.Axes.X2.Type).NotTo(BeNil())
			Expect(*got.Axes.X2.Type).To(Equal(lineplot.TickTypeTime))
			Expect(got.Axes.Y1.LabelDirection).To(BeEquivalentTo("y"))
			Expect(got.Axes.Y4.LabelDirection).To(BeEquivalentTo("y"))
			Expect(got.Axes.Y1.Type).To(BeNil())
		})
	})

	// Each spec asserts a single reshape rule. Keep one concern per spec so
	// failures localize.
	Describe("lift semantics", func() {
		migrateV4 := func(ctx SpecContext, body string) lineplot.LinePlot {
			return MustSucceed(v1.MigrateLinePlot(ctx, v0.LinePlot{
				Key:  uuid.New(),
				Data: jsonMap(`{"version": "4.0.0", ` + body + `}`),
			}))
		}

		It("Should cast Title.Level into the typed text.Level enum", func(ctx SpecContext) {
			out := migrateV4(ctx, `"title": {"level": "h2", "visible": true}`)
			Expect(out.Title.Level).To(Equal(text.LevelH2))
			Expect(out.Title.Visible).To(BeTrue())
		})

		It("Should pass Legend.Position root and units through as typed values", func(ctx SpecContext) {
			out := migrateV4(ctx, `"legend": {"visible": true, "position": {"x": 12, "y": 34, "root": {"x": "right", "y": "bottom"}, "units": {"x": "decimal", "y": "decimal"}}}`)
			Expect(out.Legend.Position.X).To(Equal(12.0))
			Expect(out.Legend.Position.Root.X).To(BeEquivalentTo("right"))
			Expect(out.Legend.Position.Root.Y).To(BeEquivalentTo("bottom"))
			Expect(out.Legend.Position.Units.X).To(BeEquivalentTo("decimal"))
		})

		It("Should default Legend.Position root and units when the wire omits them", func(ctx SpecContext) {
			out := migrateV4(ctx, `"legend": {"visible": true, "position": {"x": 0, "y": 0}}`)
			Expect(out.Legend.Position.Root.X).To(BeEquivalentTo("left"))
			Expect(out.Legend.Position.Root.Y).To(BeEquivalentTo("top"))
			Expect(out.Legend.Position.Units.X).To(BeEquivalentTo("px"))
			Expect(out.Legend.Position.Units.Y).To(BeEquivalentTo("px"))
		})

		It("Should preserve Channels arrays per axis", func(ctx SpecContext) {
			out := migrateV4(ctx, `"channels": {"x1": 99, "x2": 0, "y1": [10, 11], "y2": [12], "y3": [], "y4": []}`)
			Expect(out.Channels.X1).To(BeEquivalentTo(99))
			Expect(out.Channels.Y1).To(Equal([]channel.Key{10, 11}))
			Expect(out.Channels.Y2).To(Equal([]channel.Key{12}))
		})

		It("Should preserve Ranges arrays per x-axis", func(ctx SpecContext) {
			out := migrateV4(ctx, `"ranges": {"x1": ["00000000-0000-0000-0000-00000000000a"], "x2": []}`)
			Expect(out.Ranges.X1).To(HaveLen(1))
			Expect(out.Ranges.X1[0]).To(Equal("00000000-0000-0000-0000-00000000000a"))
		})

		It("Should drop Axes wrapper bookkeeping but preserve per-axis config", func(ctx SpecContext) {
			out := migrateV4(ctx, `"axes": {"renderTrigger": 99, "hasHadChannelSet": true, "axes": {
				"x1": {"key": "x1", "label": "t", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 1, "upper": 2}, "autoBounds": {"lower": false, "upper": true}, "tickSpacing": 60, "type": "time"},
				"x2": {"key": "x2", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
				"y1": {"key": "y1", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
				"y2": {"key": "y2", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
				"y3": {"key": "y3", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
				"y4": {"key": "y4", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75}
			}}`)
			Expect(out.Axes.X1.Label).To(Equal("t"))
			Expect(out.Axes.X1.Bounds.Lower).To(Equal(1.0))
			Expect(out.Axes.X1.ManualBounds.Upper).To(BeFalse())
			Expect(out.Axes.X1.ManualBounds.Lower).To(BeTrue())
			Expect(out.Axes.X1.TickSpacing).To(Equal(60.0))
		})

		It("Should leave Axis.Type nil when the wire omits it", func(ctx SpecContext) {
			out := migrateV4(ctx, `"axes": {"renderTrigger": 0, "hasHadChannelSet": false, "axes": {
				"x1": {"key": "x1", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
				"x2": {"key": "x2", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
				"y1": {"key": "y1", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
				"y2": {"key": "y2", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
				"y3": {"key": "y3", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
				"y4": {"key": "y4", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75}
			}}`)
			Expect(out.Axes.X1.Type).To(BeNil())
			Expect(out.Axes.Y1.Type).To(BeNil())
		})

		It("Should preserve Line.Label as nil when the wire omits it", func(ctx SpecContext) {
			out := migrateV4(ctx, `"lines": [
				{"key": "l1", "color": "#ff0000", "strokeWidth": 2, "downsample": 1, "downsampleMode": "decimate"}
			]`)
			Expect(out.Lines).To(HaveLen(1))
			Expect(out.Lines[0].Label).To(BeNil())
		})

		It("Should preserve Line.Label as a pointer to the override when present", func(ctx SpecContext) {
			out := migrateV4(ctx, `"lines": [
				{"key": "l1", "label": "custom", "color": "#ff0000", "strokeWidth": 2, "downsample": 1, "downsampleMode": "decimate"}
			]`)
			Expect(out.Lines[0].Label).NotTo(BeNil())
			Expect(*out.Lines[0].Label).To(Equal("custom"))
		})

		It("Should cast Line.DownsampleMode into the typed enum", func(ctx SpecContext) {
			out := migrateV4(ctx, `"lines": [
				{"key": "l1", "color": "#ff0000", "strokeWidth": 1, "downsample": 4, "downsampleMode": "average"}
			]`)
			Expect(out.Lines[0].DownsampleMode).To(Equal(lineplot.DownsampleModeAverage))
		})

		It("Should drop the wire-only Rule.Selected field when projecting to the typed Rule", func(ctx SpecContext) {
			out := migrateV4(ctx, `"rules": [
				{"selected": true, "key": "r1", "label": "max", "color": "#0000ff", "axis": "y1", "lineWidth": 1, "lineDash": 2, "units": "psi", "position": 4.5}
			]`)
			Expect(out.Rules).To(HaveLen(1))
			Expect(out.Rules[0].Key).To(Equal("r1"))
			Expect(out.Rules[0].Axis).To(Equal(lineplot.AxisKeyY1))
			Expect(out.Rules[0].Position).To(Equal(4.5))
		})

		It("Should pass through the gorp-entry fields (key, name)", func(ctx SpecContext) {
			key := uuid.New()
			out := MustSucceed(v1.MigrateLinePlot(ctx, v0.LinePlot{
				Key: key, Name: "tank-1",
				Data: jsonMap(`{"version": "4.0.0"}`),
			}))
			Expect(out.Key).To(Equal(key))
			Expect(out.Name).To(Equal("tank-1"))
		})

		It("Should handle a nil data blob without erroring", func(ctx SpecContext) {
			out := MustSucceed(v1.MigrateLinePlot(ctx, v0.LinePlot{
				Key: uuid.New(), Name: "empty", Data: nil,
			}))
			Expect(out.Lines).To(BeEmpty())
			Expect(out.Rules).To(BeEmpty())
		})
	})
})
