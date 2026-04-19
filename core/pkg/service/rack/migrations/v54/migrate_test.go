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
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	v54 "github.com/synnaxlabs/synnax/pkg/service/rack/migrations/v54"
	colorv54 "github.com/synnaxlabs/x/color/migrations/v54"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	labelv54 "github.com/synnaxlabs/x/label/migrations/v54"
	"github.com/synnaxlabs/x/migrate"
	statusv54 "github.com/synnaxlabs/x/status/migrations/v54"
	"github.com/synnaxlabs/x/telem"
	telemv54 "github.com/synnaxlabs/x/telem/migrations/v54"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("v54 -> current Rack migration", func() {
	It("rewrites v54-encoded entries through the new codec", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		seed := v54.Rack{
			Key:          v54.Key(0x0001_0001),
			Name:         "Seed Rack",
			TaskCounter:  7,
			Embedded:     true,
			Integrations: []string{"ni", "opc"},
		}
		MustSucceed(gorp.OpenTable[v54.Key, v54.Rack](
			ctx, gorp.TableConfig[v54.Rack]{DB: db},
		))
		Expect(gorp.NewCreate[v54.Key, v54.Rack]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())

		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Rack",
			Migrations: []migrate.Migration{
				gorp.NewEntryMigration[v54.Key, rack.Key, v54.Rack, rack.Rack](
					"v54_drop_status",
					rack.MigrateRack,
				),
			},
		})).To(Succeed())

		var got rack.Rack
		Expect(gorp.NewRetrieve[rack.Key, rack.Rack]().
			WhereKeys(rack.Key(seed.Key)).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(rack.Key(seed.Key)))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.TaskCounter).To(Equal(seed.TaskCounter))
		Expect(got.Embedded).To(Equal(seed.Embedded))
		Expect(got.Integrations).To(Equal(seed.Integrations))
		Expect(got.Status).To(BeNil())
	})

	It("drops Status and preserves core wire fields when v54 entries carry a populated Status", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		key := v54.Key(0x0001_0002)
		seed := v54.Rack{
			Key:      key,
			Name:     "Loaded Rack",
			Embedded: false,
			Status: &v54.Status{
				Key:         "rack:" + uuid.NewString(),
				Name:        "healthy",
				Variant:     statusv54.VariantSuccess,
				Message:     "rack heartbeat received",
				Description: "all integrations responding",
				Time:        telemv54.TimeStamp(telem.Now()),
				Details:     v54.StatusDetails{Rack: key},
				Labels: []labelv54.Label{
					{Key: uuid.New(), Name: "primary", Color: colorv54.Color{R: 64, G: 128, B: 255, A: 1}},
				},
			},
		}
		MustSucceed(gorp.OpenTable[v54.Key, v54.Rack](
			ctx, gorp.TableConfig[v54.Rack]{DB: db},
		))
		Expect(gorp.NewCreate[v54.Key, v54.Rack]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())

		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Rack",
			Migrations: []migrate.Migration{
				gorp.NewEntryMigration[v54.Key, rack.Key, v54.Rack, rack.Rack](
					"v54_drop_status",
					rack.MigrateRack,
				),
			},
		})).To(Succeed())

		var got rack.Rack
		Expect(gorp.NewRetrieve[rack.Key, rack.Rack]().
			WhereKeys(rack.Key(seed.Key)).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(rack.Key(seed.Key)))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Status).To(BeNil())
	})
})
