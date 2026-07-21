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
	label "github.com/synnaxlabs/synnax/pkg/service/label/types/v0"
	status "github.com/synnaxlabs/synnax/pkg/service/status/types/v1"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	v1 "github.com/synnaxlabs/synnax/pkg/service/task/types/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/types/v2"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("v1 -> current Task migration", func() {
	It("rewrites v1-encoded entries through the new codec", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[v1.Key, v1.Task]{DB: db},
		))
		seed := v1.Task{
			Key:      v1.Key(0x0000_0001_0000_0042),
			Name:     "Seed Task",
			Type:     "modbus_read",
			Config:   msgpack.EncodedJSON{"poll_rate": float64(100)},
			Internal: true,
			Snapshot: false,
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[task.Key, task.Task]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.NewEntryMigration("v54_drop_status", v2.MigrateTask),
				},
			},
		))

		var got task.Task
		Expect(currentTable.NewRetrieve().
			Where(gorp.MatchKeys[task.Key, task.Task](task.Key(seed.Key))).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(task.Key(seed.Key)))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Type).To(Equal(seed.Type))
		Expect(got.Config).To(Equal(msgpack.EncodedJSON(seed.Config)))
		Expect(got.Internal).To(Equal(seed.Internal))
		Expect(got.Snapshot).To(Equal(seed.Snapshot))
		Expect(got.Status).To(BeNil())
	})

	It("drops Status and preserves core wire fields when v1 entries carry a populated Status", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[v1.Key, v1.Task]{DB: db},
		))
		key := v1.Key(0x0000_0001_0000_00ab)
		seed := v1.Task{
			Key:    key,
			Name:   "Loaded Task",
			Type:   "opc_read",
			Config: msgpack.EncodedJSON{"endpoint": "opc.tcp://localhost:4840"},
			Status: &v1.Status{
				Key:         "task:" + uuid.NewString(),
				Name:        "running",
				Variant:     status.VariantSuccess,
				Message:     "task acquiring",
				Description: "5 channels",
				Time:        telem.TimeStamp(telem.Now()),
				Details: v1.StatusDetails{
					Task:    key,
					Running: true,
					Cmd:     "start",
				},
				Labels: []label.Label{
					{Key: uuid.New(), Name: "active", Color: color.Color{G: 200, A: 1}},
				},
			},
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[task.Key, task.Task]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.NewEntryMigration("v54_drop_status", v2.MigrateTask),
				},
			},
		))

		var got task.Task
		Expect(currentTable.NewRetrieve().
			Where(gorp.MatchKeys[task.Key, task.Task](task.Key(seed.Key))).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(task.Key(seed.Key)))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Type).To(Equal(seed.Type))
		Expect(got.Status).To(BeNil())
	})
})
