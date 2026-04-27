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
	ontologyv54 "github.com/synnaxlabs/synnax/pkg/distribution/ontology/migrations/v54"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	v54 "github.com/synnaxlabs/synnax/pkg/service/device/migrations/v54"
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

var _ = Describe("v54 -> current Device migration", func() {
	It("rewrites v54-encoded entries through the new codec", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable[v54.Key, v54.Device](
			ctx, gorp.TableConfig[v54.Key, v54.Device]{DB: db},
		))
		seed := v54.Device{
			Key:        "DEV-SERIAL-001",
			Rack:       42,
			Location:   "Lab Bench 3",
			Make:       "LabJack",
			Model:      "T7",
			Name:       "Seed Device",
			Configured: true,
			Properties: msgpack.EncodedJSON{"sample_rate": float64(1000)},
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable[device.Key, device.Device](
			ctx, gorp.TableConfig[device.Key, device.Device]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.NewEntryMigration[device.Key, device.Key, v54.Device, device.Device](
						"v54_drop_status_parent",
						device.MigrateDevice,
					),
				},
			},
		))

		var got device.Device
		Expect(currentTable.NewRetrieve().
			WhereKeys(seed.Key).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Rack).To(BeEquivalentTo(seed.Rack))
		Expect(got.Location).To(Equal(seed.Location))
		Expect(got.Make).To(Equal(seed.Make))
		Expect(got.Model).To(Equal(seed.Model))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Configured).To(Equal(seed.Configured))
		Expect(got.Properties).To(Equal(msgpack.EncodedJSON(seed.Properties)))
		Expect(got.Status).To(BeNil())
		Expect(got.Parent).To(BeNil())
	})

	It("drops Status and Parent and preserves core wire fields when v54 entries carry populated Status and Parent", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		v54Table := MustOpen(gorp.OpenTable[v54.Key, v54.Device](
			ctx, gorp.TableConfig[v54.Key, v54.Device]{DB: db},
		))
		key := "DEV-SERIAL-002"
		seed := v54.Device{
			Key:        key,
			Rack:       7,
			Location:   "Lab Bench 4",
			Make:       "NI",
			Model:      "cDAQ-9189",
			Name:       "Loaded Device",
			Configured: true,
			Properties: msgpack.EncodedJSON{"slot": float64(3)},
			Status: &v54.Status{
				Key:         "device:" + key,
				Name:        "configured",
				Variant:     statusv54.VariantSuccess,
				Message:     "device ready",
				Description: "all modules detected",
				Time:        telemv54.TimeStamp(telem.Now()),
				Details:     v54.StatusDetails{Rack: 7, Device: key},
				Labels: []labelv54.Label{
					{Key: uuid.New(), Name: "ni", Color: colorv54.Color{R: 0, G: 173, B: 239, A: 1}},
				},
			},
			Parent: &ontologyv54.ID{Type: "device", Key: "DEV-SERIAL-PARENT"},
		}
		Expect(v54Table.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		currentTable := MustOpen(gorp.OpenTable[device.Key, device.Device](
			ctx, gorp.TableConfig[device.Key, device.Device]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.NewEntryMigration[device.Key, device.Key, v54.Device, device.Device](
						"v54_drop_status_parent",
						device.MigrateDevice,
					),
				},
			},
		))

		var got device.Device
		Expect(currentTable.NewRetrieve().
			WhereKeys(seed.Key).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Status).To(BeNil())
		Expect(got.Parent).To(BeNil())
	})
})
