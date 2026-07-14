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
	labelv54 "github.com/synnaxlabs/synnax/pkg/service/label/migrations/v54"
	statusv54 "github.com/synnaxlabs/synnax/pkg/service/status/migrations/v54"
	v54 "github.com/synnaxlabs/synnax/pkg/service/task/migrations/v54"
	v56 "github.com/synnaxlabs/synnax/pkg/service/task/migrations/v56"
	colorv54 "github.com/synnaxlabs/x/color/migrations/v54"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/telem"
	telemv54 "github.com/synnaxlabs/x/telem/migrations/v54"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("v54 -> v56 Task migration", func() {
	It("rewrites v54-encoded entries through the v56 codec", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable[v54.Key, v54.Task](
			ctx, gorp.TableConfig[v54.Key, v54.Task]{DB: db},
		))
		seed := v54.Task{
			Key:      v54.Key(0x0000_0001_0000_0042),
			Name:     "Seed Task",
			Type:     "modbus_read",
			Config:   msgpack.EncodedJSON{"poll_rate": float64(100)},
			Internal: true,
			Snapshot: false,
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable[v56.Key, v56.Task](
			ctx, gorp.TableConfig[v56.Key, v56.Task]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.NewEntryMigration[v54.Key, v56.Key, v54.Task, v56.Task](
						"v54_drop_status",
						v56.MigrateTask,
					),
				},
			},
		))

		var got v56.Task
		Expect(currentTable.NewRetrieve().
			Where(gorp.MatchKeys[v56.Key, v56.Task](v56.Key(seed.Key))).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(v56.Key(seed.Key)))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Type).To(Equal(seed.Type))
		Expect(got.Config).To(Equal(msgpack.EncodedJSON(seed.Config)))
		Expect(got.Internal).To(Equal(seed.Internal))
		Expect(got.Snapshot).To(Equal(seed.Snapshot))
	})

	It("drops Status and preserves core wire fields when v54 entries carry a populated Status", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable[v54.Key, v54.Task](
			ctx, gorp.TableConfig[v54.Key, v54.Task]{DB: db},
		))
		key := v54.Key(0x0000_0001_0000_00ab)
		seed := v54.Task{
			Key:    key,
			Name:   "Loaded Task",
			Type:   "opc_read",
			Config: msgpack.EncodedJSON{"endpoint": "opc.tcp://localhost:4840"},
			Status: &v54.Status{
				Key:         "task:" + uuid.NewString(),
				Name:        "running",
				Variant:     statusv54.VariantSuccess,
				Message:     "task acquiring",
				Description: "5 channels",
				Time:        telemv54.TimeStamp(telem.Now()),
				Details: v54.StatusDetails{
					Task:    key,
					Running: true,
					Cmd:     "start",
				},
				Labels: []labelv54.Label{
					{Key: uuid.New(), Name: "active", Color: colorv54.Color{G: 200, A: 1}},
				},
			},
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable[v56.Key, v56.Task](
			ctx, gorp.TableConfig[v56.Key, v56.Task]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.NewEntryMigration[v54.Key, v56.Key, v54.Task, v56.Task](
						"v54_drop_status",
						v56.MigrateTask,
					),
				},
			},
		))

		var got v56.Task
		Expect(currentTable.NewRetrieve().
			Where(gorp.MatchKeys[v56.Key, v56.Task](v56.Key(seed.Key))).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(v56.Key(seed.Key)))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Type).To(Equal(seed.Type))
	})
})
