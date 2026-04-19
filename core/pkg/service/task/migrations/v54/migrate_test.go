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
	"github.com/synnaxlabs/synnax/pkg/service/task"
	v54 "github.com/synnaxlabs/synnax/pkg/service/task/migrations/v54"
	colorv54 "github.com/synnaxlabs/x/color/migrations/v54"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	labelv54 "github.com/synnaxlabs/x/label/migrations/v54"
	"github.com/synnaxlabs/x/migrate"
	statusv54 "github.com/synnaxlabs/x/status/migrations/v54"
	"github.com/synnaxlabs/x/telem"
	telemv54 "github.com/synnaxlabs/x/telem/migrations/v54"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("v54 -> current Task migration", func() {
	It("rewrites v54-encoded entries through the new codec", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		seed := v54.Task{
			Key:      v54.Key(0x0000_0001_0000_0042),
			Name:     "Seed Task",
			Type:     "modbus_read",
			Config:   msgpack.EncodedJSON{"poll_rate": float64(100)},
			Internal: true,
			Snapshot: false,
		}
		MustSucceed(gorp.OpenTable[v54.Key, v54.Task](
			ctx, gorp.TableConfig[v54.Task]{DB: db},
		))
		Expect(gorp.NewCreate[v54.Key, v54.Task]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())

		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Task",
			Migrations: []migrate.Migration{
				gorp.NewEntryMigration[v54.Key, task.Key, v54.Task, task.Task](
					"v54_drop_status",
					task.MigrateTask,
				),
			},
		})).To(Succeed())

		var got task.Task
		Expect(gorp.NewRetrieve[task.Key, task.Task]().
			WhereKeys(task.Key(seed.Key)).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(task.Key(seed.Key)))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Type).To(Equal(seed.Type))
		Expect(got.Config).To(Equal(msgpack.EncodedJSON(seed.Config)))
		Expect(got.Internal).To(Equal(seed.Internal))
		Expect(got.Snapshot).To(Equal(seed.Snapshot))
		Expect(got.Status).To(BeNil())
	})

	It("preserves wire fields when v54 entries carry a populated Status with Labels", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

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
		MustSucceed(gorp.OpenTable[v54.Key, v54.Task](
			ctx, gorp.TableConfig[v54.Task]{DB: db},
		))
		Expect(gorp.NewCreate[v54.Key, v54.Task]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())

		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Task",
			Migrations: []migrate.Migration{
				gorp.NewEntryMigration[v54.Key, task.Key, v54.Task, task.Task](
					"v54_drop_status",
					task.MigrateTask,
				),
			},
		})).To(Succeed())

		var got task.Task
		Expect(gorp.NewRetrieve[task.Key, task.Task]().
			WhereKeys(task.Key(seed.Key)).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(task.Key(seed.Key)))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Type).To(Equal(seed.Type))
		Expect(got.Status).To(BeNil())
	})
})
