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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	labelv0 "github.com/synnaxlabs/synnax/pkg/service/label/types/v0"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	v0 "github.com/synnaxlabs/synnax/pkg/service/rack/types/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/rack/types/v1"
	statusv0 "github.com/synnaxlabs/synnax/pkg/service/status/types/v0"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/telem"
	telemv0 "github.com/synnaxlabs/x/telem/types/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("v1 -> current Rack migration", func() {
	It("rewrites v1-encoded entries through the new codec", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[v0.Key, v0.Rack]{DB: db},
		))
		seed := v0.Rack{
			Key:          v0.Key(0x0001_0001),
			Name:         "Seed Rack",
			TaskCounter:  7,
			Embedded:     true,
			Integrations: []string{"ni", "opc"},
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[rack.Key, rack.Rack]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.NewEntryMigration("v54_drop_status", v2.MigrateRack),
				},
			},
		))

		var got rack.Rack
		Expect(currentTable.NewRetrieve().
			Where(gorp.MatchKeys[rack.Key, rack.Rack](rack.Key(seed.Key))).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(rack.Key(seed.Key)))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.TaskCounter).To(Equal(seed.TaskCounter))
		Expect(got.Embedded).To(Equal(seed.Embedded))
		Expect(got.Integrations).To(Equal(seed.Integrations))
		Expect(got.Status).To(BeNil())
	})

	It("drops Status and preserves core wire fields when v1 entries carry a populated Status", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[v0.Key, v0.Rack]{DB: db},
		))
		key := v0.Key(0x0001_0002)
		seed := v0.Rack{
			Key:      key,
			Name:     "Loaded Rack",
			Embedded: false,
			Status: &v0.Status{
				Key:         "rack:" + uuid.NewString(),
				Name:        "healthy",
				Variant:     statusv0.VariantSuccess,
				Message:     "rack heartbeat received",
				Description: "all integrations responding",
				Time:        telemv0.TimeStamp(telem.Now()),
				Details:     v0.StatusDetails{Rack: key},
				Labels: []labelv0.Label{
					{Key: uuid.New(), Name: "primary", Color: color.Color{R: 64, G: 128, B: 255, A: 1}},
				},
			},
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[rack.Key, rack.Rack]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.NewEntryMigration("v54_drop_status", v2.MigrateRack),
				},
			},
		))

		var got rack.Rack
		Expect(currentTable.NewRetrieve().
			Where(gorp.MatchKeys[rack.Key, rack.Rack](rack.Key(seed.Key))).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(rack.Key(seed.Key)))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Status).To(BeNil())
	})
})
