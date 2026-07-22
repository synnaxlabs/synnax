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
	graph "github.com/synnaxlabs/arc/graph/types/v0"
	ir "github.com/synnaxlabs/arc/ir/types/v0"
	text "github.com/synnaxlabs/arc/text/types/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v2"
	label "github.com/synnaxlabs/synnax/pkg/service/label/types/v0"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	spatial "github.com/synnaxlabs/x/spatial/types/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateArc", func() {
	Describe("v1 -> current", func() {
		It("Should seed the document from the previously persisted raw text", func(ctx SpecContext) {
			got := migrateFromV0(ctx, v0.Arc{
				Key:  uuid.New(),
				Name: "legacy",
				Mode: v0.ModeText,
				Text: text.Text{Raw: "x := 1"},
			})
			Expect(got.Text.Materialize().Raw).To(Equal("x := 1"))
		})

		It("Should produce an empty document when there is no prior text", func(ctx SpecContext) {
			got := migrateFromV0(ctx, v0.Arc{
				Key:  uuid.New(),
				Name: "empty",
				Mode: v0.ModeGraph,
			})
			Expect(got.Text.Materialize().Raw).To(Equal(""))
		})
	})

	Describe("v0 -> current", func() {
		It("rewrites v0-encoded entries through the new codec", func(ctx SpecContext) {
			seed := v0.Arc{
				Key:  uuid.New(),
				Name: "Seed",
				Mode: v0.ModeText,
				Text: text.Text{Raw: "channel x; 1 -> x"},
				Graph: graph.Graph{
					Viewport: graph.Viewport{
						Position: spatial.XY{X: 12, Y: -34},
						Zoom:     1.5,
					},
					Functions: ir.Functions{
						{Key: "scale", Body: ir.Body{Raw: "x * 2"}},
					},
					Edges: ir.Edges{
						{
							Source: ir.Handle{Node: "n1", Param: "out"},
							Target: ir.Handle{Node: "n2", Param: "in"},
							Kind:   ir.EdgeKindContinuous,
						},
					},
					Nodes: graph.Nodes{
						{Key: "n1", Type: "scale", Position: spatial.XY{X: 0, Y: 0}},
						{Key: "n2", Type: "scale", Position: spatial.XY{X: 100, Y: 50}},
					},
				},
			}
			got := migrateFromV0(ctx, seed)
			Expect(got.Key).To(Equal(seed.Key))
			Expect(got.Name).To(Equal(seed.Name))
			Expect(got.Mode).To(Equal(v2.Mode(seed.Mode)))
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
			seed := v0.Arc{
				Key:  uuid.New(),
				Name: "Legacy Status Graph",
				Mode: v0.ModeGraph,
				Graph: graph.Graph{
					Nodes: graph.Nodes{
						{
							Key:  "alarm",
							Type: "set_status",
							Config: msgpack.EncodedJSON{
								"statusKey":   "ox_alarm",
								"variant":     "error",
								"message":     "Overpressure",
								"description": "dropped on migrate",
							},
							Position: spatial.XY{X: 0, Y: 0},
						},
						{
							Key:      "scale",
							Type:     "scale",
							Config:   msgpack.EncodedJSON{"factor": "2"},
							Position: spatial.XY{X: 100, Y: 0},
						},
					},
				},
			}
			got := migrateFromV0(ctx, seed)
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
			seed := v0.Arc{
				Key:  uuid.New(),
				Name: "Bare Status Node",
				Mode: v0.ModeGraph,
				Graph: graph.Graph{
					Nodes: graph.Nodes{{Key: "alarm", Type: "set_status"}},
				},
			}
			got := migrateFromV0(ctx, seed)
			_ = MustBeOk(got.Graph.Nodes.Find("alarm"))
			alarmCfg := got.Graph.Inputs["alarm"]
			Expect(alarmCfg["type"]).To(Equal("status.set"))
			Expect(alarmCfg["key_or_name"]).To(Equal(""))
			Expect(alarmCfg["variant"]).To(Equal("success"))
			Expect(alarmCfg["message"]).To(Equal(""))
		})

		It("drops Status and Program and preserves core wire fields when v0 entries carry a populated Status", func(ctx SpecContext) {
			statusKey := uuid.New().String()
			labelKey := uuid.New()
			seed := v0.Arc{
				Key:  uuid.New(),
				Name: "Loaded Seed",
				Mode: v0.ModeGraph,
				Text: text.Text{Raw: ""},
				Status: &v0.Status{
					Key:         statusKey,
					Name:        "running",
					Variant:     "success",
					Message:     "task is running",
					Description: "started 5s ago",
					Time:        telem.Now(),
					Details:     v0.StatusDetails{Running: true},
					Labels: []label.Label{
						{Key: labelKey, Name: "critical", Color: color.Color{R: 255, A: 1}},
					},
				},
			}
			got := migrateFromV0(ctx, seed)
			Expect(got.Key).To(Equal(seed.Key))
			Expect(got.Name).To(Equal(seed.Name))
			Expect(got.Mode).To(Equal(v2.Mode(seed.Mode)))
			Expect(got.Status).To(BeNil())
			Expect(got.Program).To(BeNil())
		})
	})
})

// migrateFromV0 runs the full arc migration chain over a gorp-seeded v0 arc and
// returns the migrated current Arc.
func migrateFromV0(ctx SpecContext, seed v0.Arc) v2.Arc {
	db := DeferClose(gorp.Wrap(memkv.New()))
	MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Arc]{DB: db}))
	Expect(gorp.NewCreate[v0.Key, v0.Arc]().Entry(&seed).Exec(ctx, db)).To(Succeed())
	Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:        db,
		Namespace: "Arc",
		Migrations: append(
			append([]migrate.Migration{}, v1.Migrations...), v2.Migration,
		),
	})).To(Succeed())
	var got v2.Arc
	Expect(gorp.NewRetrieve[v2.Key, v2.Arc]().
		Where(gorp.MatchKeys[v2.Key, v2.Arc](seed.Key)).
		Entry(&got).Exec(ctx, db)).To(Succeed())
	return got
}
