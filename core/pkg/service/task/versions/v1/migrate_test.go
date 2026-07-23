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
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	label "github.com/synnaxlabs/synnax/pkg/service/label/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v1"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateTask", func() {
	migrateSeed := func(ctx SpecContext, seed v0.Task) v1.Task {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Task]{DB: db}))
		Expect(gorp.NewCreate[v0.Key, v0.Task]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())
		v0Applied := gorp.NewMigration(
			v0.NewMigration(v0.MigrationConfig{}).Key(),
			func(context.Context, gorp.Tx, alamos.Instrumentation) error { return nil },
		)
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Task",
			Migrations: append([]migrate.Migration{v0Applied}, v1.Migrations...),
		})).To(Succeed())
		var got v1.Task
		Expect(gorp.NewRetrieve[v1.Key, v1.Task]().
			Where(gorp.MatchKeys[v1.Key, v1.Task](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	It("Should lift a v0 task directly, dropping the status", func(ctx SpecContext) {
		migrated := migrateSeed(ctx, v0.Task{
			Key:      v0.Key(0x0000_0001_0000_0007),
			Name:     "Direct",
			Type:     "modbus_read",
			Config:   msgpack.EncodedJSON{"poll_rate": float64(50)},
			Internal: true,
			Snapshot: true,
			Status:   &v0.Status{Name: "running", Variant: "success"},
		})
		Expect(migrated.Key).To(Equal(v1.Key(0x0000_0001_0000_0007)))
		Expect(migrated.Name).To(Equal("Direct"))
		Expect(migrated.Type).To(Equal("modbus_read"))
		Expect(migrated.Config).To(Equal(msgpack.EncodedJSON{"poll_rate": float64(50)}))
		Expect(migrated.Internal).To(BeTrue())
		Expect(migrated.Snapshot).To(BeTrue())
		Expect(migrated.Status).To(BeNil())
	})

	It("rewrites v1-encoded entries through the new codec", func(ctx SpecContext) {
		seed := v0.Task{
			Key:      v0.Key(0x0000_0001_0000_0042),
			Name:     "Seed Task",
			Type:     "modbus_read",
			Config:   msgpack.EncodedJSON{"poll_rate": float64(100)},
			Internal: true,
			Snapshot: false,
		}
		got := migrateSeed(ctx, seed)
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Type).To(Equal(seed.Type))
		Expect(got.Config).To(Equal(msgpack.EncodedJSON(seed.Config)))
		Expect(got.Internal).To(Equal(seed.Internal))
		Expect(got.Snapshot).To(Equal(seed.Snapshot))
		Expect(got.Status).To(BeNil())
	})

	It("drops Status and preserves core wire fields when v1 entries carry a populated Status", func(ctx SpecContext) {
		key := v0.Key(0x0000_0001_0000_00ab)
		seed := v0.Task{
			Key:    key,
			Name:   "Loaded Task",
			Type:   "opc_read",
			Config: msgpack.EncodedJSON{"endpoint": "opc.tcp://localhost:4840"},
			Status: &v0.Status{
				Key:         "task:" + uuid.NewString(),
				Name:        "running",
				Variant:     "success",
				Message:     "task acquiring",
				Description: "5 channels",
				Time:        telem.TimeStamp(telem.Now()),
				Details: v0.StatusDetails{
					Task:    key,
					Running: true,
					Cmd:     "start",
				},
				Labels: []label.Label{
					{Key: uuid.New(), Name: "active", Color: color.Color{G: 200, A: 1}},
				},
			},
		}
		got := migrateSeed(ctx, seed)
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Type).To(Equal(seed.Type))
		Expect(got.Status).To(BeNil())
	})
})
