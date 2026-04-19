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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	v54 "github.com/synnaxlabs/synnax/pkg/service/task/migrations/v54"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
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
})
