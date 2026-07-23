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
	v0 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v1"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateRack", func() {
	migrateSeed := func(ctx SpecContext, seed v0.Rack) v1.Rack {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Rack]{DB: db}))
		Expect(gorp.NewCreate[v0.Key, v0.Rack]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())
		v0Applied := gorp.NewMigration(
			v0.NewMigration(v0.MigrationConfig{}).Key(),
			func(context.Context, gorp.Tx, alamos.Instrumentation) error { return nil },
		)
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Rack",
			Migrations: append([]migrate.Migration{v0Applied}, v1.Migrations...),
		})).To(Succeed())
		var got v1.Rack
		Expect(gorp.NewRetrieve[v1.Key, v1.Rack]().
			Where(gorp.MatchKeys[v1.Key, v1.Rack](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	It("Should lift a v0 rack directly, dropping the status", func(ctx SpecContext) {
		migrated := migrateSeed(ctx, v0.Rack{
			Key:          v0.Key(0x0001_0009),
			Name:         "Direct",
			TaskCounter:  3,
			Embedded:     true,
			Integrations: []string{"ni"},
			Status:       &v0.Status{Name: "healthy", Variant: "success"},
		})
		Expect(migrated.Key).To(Equal(v1.Key(0x0001_0009)))
		Expect(migrated.Name).To(Equal("Direct"))
		Expect(migrated.TaskCounter).To(Equal(uint32(3)))
		Expect(migrated.Embedded).To(BeTrue())
		Expect(migrated.Integrations).To(Equal([]string{"ni"}))
		Expect(migrated.Status).To(BeNil())
	})

	It("rewrites v1-encoded entries through the new codec", func(ctx SpecContext) {
		seed := v0.Rack{
			Key:          v0.Key(0x0001_0001),
			Name:         "Seed Rack",
			TaskCounter:  7,
			Embedded:     true,
			Integrations: []string{"ni", "opc"},
		}
		got := migrateSeed(ctx, seed)
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.TaskCounter).To(Equal(seed.TaskCounter))
		Expect(got.Embedded).To(Equal(seed.Embedded))
		Expect(got.Integrations).To(Equal(seed.Integrations))
		Expect(got.Status).To(BeNil())
	})

	It("drops Status and preserves core wire fields when v1 entries carry a populated Status", func(ctx SpecContext) {
		key := v0.Key(0x0001_0002)
		seed := v0.Rack{
			Key:      key,
			Name:     "Loaded Rack",
			Embedded: false,
			Status: &v0.Status{
				Key:         "rack:" + uuid.NewString(),
				Name:        "healthy",
				Variant:     "success",
				Message:     "rack heartbeat received",
				Description: "all integrations responding",
				Time:        telem.Now(),
				Details:     v0.StatusDetails{Rack: key},
				Labels: []label.Label{
					{Key: uuid.New(), Name: "primary", Color: color.Color{R: 64, G: 128, B: 255, A: 1}},
				},
			},
		}
		got := migrateSeed(ctx, seed)
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Status).To(BeNil())
	})
})
