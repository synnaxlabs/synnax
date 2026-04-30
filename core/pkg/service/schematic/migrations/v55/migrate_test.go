// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v55_test

import (
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	v55 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v55"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

func encodeBlob(raw string) msgpack.EncodedJSON {
	var m map[string]any
	Expect(json.Unmarshal([]byte(raw), &m)).To(Succeed())
	return m
}

var _ = Describe("v55 -> current Schematic migration", func() {
	It("Should lift a v5 wire-format blob into the typed Schematic on retrieve", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v55Table := MustOpen(gorp.OpenTable[uuid.UUID, v55.Schematic](
			ctx, gorp.TableConfig[v55.Schematic]{DB: db},
		))
		seed := v55.Schematic{
			Key:  uuid.New(),
			Name: "Tank Farm",
			Data: encodeBlob(`{
				"version": "5.0.0",
				"authority": 7,
				"nodes": [{"key": "n1", "position": {"x": 100, "y": 200}}],
				"edges": [{"key": "e1", "source": "n1", "target": "n2", "sourceHandle": "outlet", "targetHandle": "inlet"}],
				"props": {"n1": {"key": "tank", "color": "#0080ff"}},
				"legend": {"visible": true, "position": {"x": 50, "y": 50, "units": {"x": "px", "y": "px"}}, "colors": {}}
			}`),
		}
		Expect(v55Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable[uuid.UUID, schematic.Schematic](
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

		var got schematic.Schematic
		Expect(currentTable.NewRetrieve().
			WhereKeys(seed.Key).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal("Tank Farm"))
		Expect(got.Authority).To(BeEquivalentTo(7))
		Expect(got.Nodes).To(HaveLen(1))
		Expect(got.Nodes[0].Key).To(Equal("n1"))
		Expect(got.Nodes[0].Position.X).To(Equal(100.0))
		Expect(got.Edges).To(HaveLen(1))
		Expect(got.Edges[0].Source).To(Equal(schematic.Handle{Node: "n1", Param: "outlet"}))
		Expect(got.Edges[0].Target).To(Equal(schematic.Handle{Node: "n2", Param: "inlet"}))
		Expect(got.Props["n1"]["variant"]).To(Equal("tank"))
		Expect(got.Legend.Visible).To(BeTrue())
	})

	It("Should chain a legacy v0 blob through every migration step on retrieve", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v55Table := MustOpen(gorp.OpenTable[uuid.UUID, v55.Schematic](
			ctx, gorp.TableConfig[v55.Schematic]{DB: db},
		))
		seed := v55.Schematic{
			Key:  uuid.New(),
			Name: "Legacy",
			Data: encodeBlob(`{
				"version": "0.0.0",
				"nodes": [{"key": "n1", "position": {"x": 0, "y": 0}}],
				"edges": [{"key": "e1", "source": "n1", "target": "n2", "sourceHandle": "out", "targetHandle": "in"}],
				"props": {"n1": {"key": "valve"}}
			}`),
		}
		Expect(v55Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable[uuid.UUID, schematic.Schematic](
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

		var got schematic.Schematic
		Expect(currentTable.NewRetrieve().
			WhereKeys(seed.Key).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Edges[0].Source).To(Equal(schematic.Handle{Node: "n1", Param: "out"}))
		Expect(got.Edges[0].Target).To(Equal(schematic.Handle{Node: "n2", Param: "in"}))
		Expect(got.Authority).To(BeEquivalentTo(1))
		Expect(got.Props["n1"]["variant"]).To(Equal("valve"))
	})
})
