// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	graphv0 "github.com/synnaxlabs/arc/graph/types/v0"
	irv0 "github.com/synnaxlabs/arc/ir/types/v0"
	textv0 "github.com/synnaxlabs/arc/text/types/v0"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/types/v0"
	arcv1 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v1"
	arcv2 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v2"
	labelv0 "github.com/synnaxlabs/synnax/pkg/service/label/types/v0"
	statusv1 "github.com/synnaxlabs/synnax/pkg/service/status/types/v1"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	spatialv0 "github.com/synnaxlabs/x/spatial/types/v0"
	"github.com/synnaxlabs/x/telem"
	telemv0 "github.com/synnaxlabs/x/telem/types/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("v0 -> current Arc migration", func() {
	It("rewrites v0-encoded entries through the new codec", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[v0.Key, v0.Arc]{DB: db},
		))
		seed := v0.Arc{
			Key:  uuid.New(),
			Name: "Seed",
			Mode: v0.ModeText,
			Text: textv0.Text{Raw: "channel x; 1 -> x"},
			Graph: graphv0.Graph{
				Viewport: graphv0.Viewport{
					Position: spatialv0.XY{X: 12, Y: -34},
					Zoom:     1.5,
				},
				Functions: irv0.Functions{
					{Key: "scale", Body: irv0.Body{Raw: "x * 2"}},
				},
				Edges: irv0.Edges{
					{
						Source: irv0.Handle{Node: "n1", Param: "out"},
						Target: irv0.Handle{Node: "n2", Param: "in"},
						Kind:   irv0.EdgeKindContinuous,
					},
				},
				Nodes: graphv0.Nodes{
					{Key: "n1", Type: "scale", Position: spatialv0.XY{X: 0, Y: 0}},
					{Key: "n2", Type: "scale", Position: spatialv0.XY{X: 100, Y: 50}},
				},
			},
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[arc.Key, arc.Arc]{
				DB:         db,
				Migrations: arcMigrations(),
			},
		))

		var got arc.Arc
		Expect(currentTable.NewRetrieve().
			Where(gorp.MatchKeys[arc.Key, arc.Arc](seed.Key)).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Mode).To(Equal(arc.Mode(seed.Mode)))
		Expect(got.Text.Materialize().Raw).To(Equal(seed.Text.Raw))
		Expect(got.Graph.Functions).To(HaveLen(1))
		Expect(got.Graph.Functions[0].Key).To(Equal("scale"))
		Expect(got.Graph.Functions[0].Body.Raw).To(Equal("x * 2"))
		Expect(got.Graph.Edges).To(HaveLen(1))
		Expect(got.Graph.Edges[0].Source.Node).To(Equal("n1"))
		Expect(got.Graph.Edges[0].Target.Param).To(Equal("in"))
		Expect(got.Graph.Nodes).To(HaveLen(2))
		Expect(got.Graph.Nodes[1].Position.X).To(Equal(100.0))
		Expect(got.Program).To(BeNil())
		Expect(got.Status).To(BeNil())
	})

	It("rewrites deprecated set_status graph nodes to status.set", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[v0.Key, v0.Arc]{DB: db},
		))
		seed := v0.Arc{
			Key:  uuid.New(),
			Name: "Legacy Status Graph",
			Mode: v0.ModeGraph,
			Graph: graphv0.Graph{
				Nodes: graphv0.Nodes{
					{
						Key:  "alarm",
						Type: "set_status",
						Config: msgpack.EncodedJSON{
							"statusKey":   "ox_alarm",
							"variant":     "error",
							"message":     "Overpressure",
							"description": "dropped on migrate",
						},
						Position: spatialv0.XY{X: 0, Y: 0},
					},
					{
						Key:      "scale",
						Type:     "scale",
						Config:   msgpack.EncodedJSON{"factor": "2"},
						Position: spatialv0.XY{X: 100, Y: 0},
					},
				},
			},
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[arc.Key, arc.Arc]{
				DB:         db,
				Migrations: arcMigrations(),
			},
		))

		var got arc.Arc
		Expect(currentTable.NewRetrieve().
			Where(gorp.MatchKeys[arc.Key, arc.Arc](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Graph.Nodes).To(HaveLen(2))

		_ = MustBeOk(got.Graph.Nodes.Find("alarm"))
		alarmCfg := got.Graph.Inputs["alarm"]
		Expect(alarmCfg["type"]).To(Equal("status.set"))
		Expect(alarmCfg["key_or_name"]).To(Equal("ox_alarm"))
		Expect(alarmCfg["variant"]).To(Equal("error"))
		Expect(alarmCfg["message"]).To(Equal("Overpressure"))
		Expect(alarmCfg).ToNot(HaveKey("statusKey"))
		Expect(alarmCfg).ToNot(HaveKey("description"))

		_ = MustBeOk(got.Graph.Nodes.Find("scale"))
		scaleCfg := got.Graph.Inputs["scale"]
		Expect(scaleCfg["type"]).To(Equal("scale"))
		Expect(scaleCfg["factor"]).To(Equal("2"))
	})

	It("defaults missing set_status config parameters", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[v0.Key, v0.Arc]{DB: db},
		))
		seed := v0.Arc{
			Key:  uuid.New(),
			Name: "Bare Status Node",
			Mode: v0.ModeGraph,
			Graph: graphv0.Graph{
				Nodes: graphv0.Nodes{{Key: "alarm", Type: "set_status"}},
			},
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[arc.Key, arc.Arc]{
				DB:         db,
				Migrations: arcMigrations(),
			},
		))

		var got arc.Arc
		Expect(currentTable.NewRetrieve().
			Where(gorp.MatchKeys[arc.Key, arc.Arc](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		_ = MustBeOk(got.Graph.Nodes.Find("alarm"))
		alarmCfg := got.Graph.Inputs["alarm"]
		Expect(alarmCfg["type"]).To(Equal("status.set"))
		Expect(alarmCfg["key_or_name"]).To(Equal(""))
		Expect(alarmCfg["variant"]).To(Equal("success"))
		Expect(alarmCfg["message"]).To(Equal(""))
	})

	It("drops Status and Program and preserves core wire fields when v0 entries carry a populated Status", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[v0.Key, v0.Arc]{DB: db},
		))
		statusKey := uuid.New().String()
		labelKey := uuid.New()
		seed := v0.Arc{
			Key:  uuid.New(),
			Name: "Loaded Seed",
			Mode: v0.ModeGraph,
			Text: textv0.Text{Raw: ""},
			Status: &v0.Status{
				Key:         statusKey,
				Name:        "running",
				Variant:     statusv1.VariantSuccess,
				Message:     "task is running",
				Description: "started 5s ago",
				Time:        telemv0.TimeStamp(telem.Now()),
				Details:     v0.StatusDetails{Running: true},
				Labels: []labelv0.Label{
					{Key: labelKey, Name: "critical", Color: color.Color{R: 255, A: 1}},
				},
			},
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[arc.Key, arc.Arc]{
				DB:         db,
				Migrations: arcMigrations(),
			},
		))

		var got arc.Arc
		Expect(currentTable.NewRetrieve().
			Where(gorp.MatchKeys[arc.Key, arc.Arc](seed.Key)).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Mode).To(Equal(arc.Mode(seed.Mode)))
		Expect(got.Status).To(BeNil())
		Expect(got.Program).To(BeNil())
	})
})

func arcMigrations() []migrate.Migration {
	return []migrate.Migration{
		gorp.NewEntryMigration("v54_drop_program_status", arcv1.MigrateArc),
		migrate.WithAddedDeps(
			gorp.NewEntryMigration("v55_rename_set_status", arcv1.RenameSetStatus),
			"v54_drop_program_status",
		),
		migrate.WithAddedDeps(
			gorp.NewEntryMigration("v56_to_live", arcv2.MigrateArc),
			"v55_rename_set_status",
		),
	}
}
