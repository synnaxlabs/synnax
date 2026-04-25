// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v54_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	graphv54 "github.com/synnaxlabs/arc/graph/migrations/v54"
	irv54 "github.com/synnaxlabs/arc/ir/migrations/v54"
	textv54 "github.com/synnaxlabs/arc/text/migrations/v54"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	v54 "github.com/synnaxlabs/synnax/pkg/service/arc/migrations/v54"
	colorv54 "github.com/synnaxlabs/x/color/migrations/v54"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	labelv54 "github.com/synnaxlabs/x/label/migrations/v54"
	"github.com/synnaxlabs/x/migrate"
	spatialv54 "github.com/synnaxlabs/x/spatial/migrations/v54"
	statusv54 "github.com/synnaxlabs/x/status/migrations/v54"
	"github.com/synnaxlabs/x/telem"
	telemv54 "github.com/synnaxlabs/x/telem/migrations/v54"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("v54 -> current Arc migration", func() {
	It("rewrites v54-encoded entries through the new codec", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable[uuid.UUID, v54.Arc](
			ctx, gorp.TableConfig[v54.Arc]{DB: db},
		))
		seed := v54.Arc{
			Key:  uuid.New(),
			Name: "Seed",
			Mode: v54.ModeText,
			Text: textv54.Text{Raw: "channel x; 1 -> x"},
			Graph: graphv54.Graph{
				Viewport: graphv54.Viewport{
					Position: spatialv54.XY{X: 12, Y: -34},
					Zoom:     1.5,
				},
				Functions: irv54.Functions{
					{Key: "scale", Body: irv54.Body{Raw: "x * 2"}},
				},
				Edges: irv54.Edges{
					{
						Source: irv54.Handle{Node: "n1", Param: "out"},
						Target: irv54.Handle{Node: "n2", Param: "in"},
						Kind:   irv54.EdgeKindContinuous,
					},
				},
				Nodes: graphv54.Nodes{
					{Key: "n1", Type: "scale", Position: spatialv54.XY{X: 0, Y: 0}},
					{Key: "n2", Type: "scale", Position: spatialv54.XY{X: 100, Y: 50}},
				},
			},
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable[uuid.UUID, arc.Arc](
			ctx, gorp.TableConfig[arc.Arc]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.NewEntryMigration[uuid.UUID, uuid.UUID, v54.Arc, arc.Arc](
						"v54_drop_program_status",
						arc.MigrateArc,
					),
				},
			},
		))

		var got arc.Arc
		Expect(currentTable.NewRetrieve().
			WhereKeys(seed.Key).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Mode).To(Equal(arc.Mode(seed.Mode)))
		Expect(got.Text.Raw).To(Equal(seed.Text.Raw))
		Expect(got.Graph.Viewport.Zoom).To(Equal(seed.Graph.Viewport.Zoom))
		Expect(got.Graph.Viewport.Position.X).To(Equal(seed.Graph.Viewport.Position.X))
		Expect(got.Graph.Viewport.Position.Y).To(Equal(seed.Graph.Viewport.Position.Y))
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

	It("drops Status and Program and preserves core wire fields when v54 entries carry a populated Status", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable[uuid.UUID, v54.Arc](
			ctx, gorp.TableConfig[v54.Arc]{DB: db},
		))
		statusKey := uuid.New().String()
		labelKey := uuid.New()
		seed := v54.Arc{
			Key:  uuid.New(),
			Name: "Loaded Seed",
			Mode: v54.ModeGraph,
			Text: textv54.Text{Raw: ""},
			Status: &v54.Status{
				Key:         statusKey,
				Name:        "running",
				Variant:     statusv54.VariantSuccess,
				Message:     "task is running",
				Description: "started 5s ago",
				Time:        telemv54.TimeStamp(telem.Now()),
				Details:     v54.StatusDetails{Running: true},
				Labels: []labelv54.Label{
					{Key: labelKey, Name: "critical", Color: colorv54.Color{R: 255, A: 1}},
				},
			},
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable[uuid.UUID, arc.Arc](
			ctx, gorp.TableConfig[arc.Arc]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.NewEntryMigration[uuid.UUID, uuid.UUID, v54.Arc, arc.Arc](
						"v54_drop_program_status",
						arc.MigrateArc,
					),
				},
			},
		))

		var got arc.Arc
		Expect(currentTable.NewRetrieve().
			WhereKeys(seed.Key).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Mode).To(Equal(arc.Mode(seed.Mode)))
		Expect(got.Status).To(BeNil())
		Expect(got.Program).To(BeNil())
	})
})
