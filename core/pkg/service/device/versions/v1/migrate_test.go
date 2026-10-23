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
	"encoding/json"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	v0 "github.com/synnaxlabs/synnax/pkg/service/device/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/device/versions/v1"
	label "github.com/synnaxlabs/synnax/pkg/service/label/versions/v0"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateDevice", func() {
	migrateSeed := func(ctx SpecContext, seed v0.Device) v1.Device {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Device]{DB: db}))
		Expect(gorp.NewCreate[v0.Key, v0.Device]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())
		v0Chain := v0.NewMigrations(v0.MigrationConfig{})
		v0Applied := gorp.NewMigration(
			v0Chain[0].Key(),
			func(context.Context, gorp.Tx, alamos.Instrumentation) error { return nil },
		)
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Device",
			Migrations: append(
				append([]migrate.Migration{v0Applied}, v0Chain[1:]...), v1.Migration,
			),
		})).To(Succeed())
		var got v1.Device
		Expect(gorp.NewRetrieve[v1.Key, v1.Device]().
			Where(gorp.MatchKeys[v1.Key, v1.Device](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	It(
		"Should lift a v0 device directly, dropping the status and parent",
		func(ctx SpecContext) {
			migrated := migrateSeed(ctx, v0.Device{
				Key:        "DEV-DIRECT-001",
				Rack:       42,
				Location:   "Lab Bench 3",
				Make:       "LabJack",
				Model:      "T7",
				Name:       "Direct",
				Configured: true,
				Properties: msgpack.EncodedJSON{"serial": "T7-001"},
				Status:     &v0.Status{Name: "connected", Variant: "success"},
				Parent:     &ontology.ID{Type: "device", Key: "DEV-PARENT"},
			})
			Expect(migrated.Key).To(Equal(v1.Key("DEV-DIRECT-001")))
			Expect(migrated.Rack).To(BeEquivalentTo(42))
			Expect(migrated.Location).To(Equal("Lab Bench 3"))
			Expect(migrated.Make).To(Equal("LabJack"))
			Expect(migrated.Model).To(Equal("T7"))
			Expect(migrated.Name).To(Equal("Direct"))
			Expect(migrated.Configured).To(BeTrue())
			Expect(
				migrated.Properties,
			).To(Equal(msgpack.EncodedJSON{"serial": "T7-001"}))
			Expect(migrated.Status).To(BeNil())
			Expect(migrated.Parent).To(BeNil())
		},
	)

	It("rewrites v1-encoded entries through the new codec", func(ctx SpecContext) {
		seed := v0.Device{
			Key:        "DEV-SERIAL-001",
			Rack:       42,
			Location:   "Lab Bench 3",
			Make:       "LabJack",
			Model:      "T7",
			Name:       "Seed Device",
			Configured: true,
			Properties: msgpack.EncodedJSON{"sample_rate": float64(1000)},
		}
		got := migrateSeed(ctx, seed)
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

	It(
		"drops Status and Parent and preserves core wire fields when v1 entries carry populated Status and Parent",
		func(ctx SpecContext) {
			key := "DEV-SERIAL-002"
			seed := v0.Device{
				Key:        key,
				Rack:       7,
				Location:   "Lab Bench 4",
				Make:       "NI",
				Model:      "cDAQ-9189",
				Name:       "Loaded Device",
				Configured: true,
				Properties: msgpack.EncodedJSON{"slot": float64(3)},
				Status: &v0.Status{
					Key:         "device:" + key,
					Name:        "configured",
					Variant:     "success",
					Message:     "device ready",
					Description: "all modules detected",
					Time:        telem.Now(),
					Details:     v0.StatusDetails{Rack: 7, Device: key},
					Labels: []label.Label{
						{
							Key:   uuid.New(),
							Name:  "ni",
							Color: color.Color{R: 0, G: 173, B: 239, A: 1},
						},
					},
				},
				Parent: &ontology.ID{Type: "device", Key: "DEV-SERIAL-PARENT"},
			}
			got := migrateSeed(ctx, seed)
			Expect(got.Key).To(Equal(seed.Key))
			Expect(got.Name).To(Equal(seed.Name))
			Expect(got.Status).To(BeNil())
			Expect(got.Parent).To(BeNil())
		},
	)
})

var _ = Describe("HTTPHealthCheckMigration", func() {
	migrateDevice := func(ctx SpecContext, d v1.Device) v1.Device {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		Expect(gorp.NewCreate[v1.Key, v1.Device]().Entry(&d).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Device",
			Migrations: []migrate.Migration{v1.HTTPHealthCheckMigration},
		})).To(Succeed())
		var got v1.Device
		Expect(gorp.NewRetrieve[v1.Key, v1.Device]().
			Where(gorp.MatchKeys[v1.Key, v1.Device](d.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}
	// The properties a v0.53.0 Console stored for an HTTP device.
	v053Properties := func() msgpack.EncodedJSON {
		GinkgoHelper()
		var p msgpack.EncodedJSON
		Expect(json.Unmarshal([]byte(`{
			"max_concurrent_requests": 6, "write": {}, "timeout_ms": 100,
			"secure": false, "read": {}, "version": 1, "verify_ssl": true,
			"health_check": {
				"headers": {"X-Token": "abc", "Accept": "application/json"},
				"query_params": {"probe": "1"},
				"method": "GET", "path": "/health", "validate_response": false
			},
			"auth": {"type": "none"}
		}`), &p)).To(Succeed())
		return p
	}

	It("Should list the health check headers and parameters of a v0.53.0 device",
		func(ctx SpecContext) {
			got := migrateDevice(ctx, v1.Device{
				Key: "k53-http", Make: "http", Properties: v053Properties(),
			})
			hc := got.Properties["health_check"].(map[string]any)
			Expect(hc["headers"]).To(Equal([]any{
				map[string]any{"name": "Accept", "value": "application/json"},
				map[string]any{"name": "X-Token", "value": "abc"},
			}))
			Expect(hc["query_params"]).To(Equal([]any{
				map[string]any{"parameter": "probe", "value": "1"},
			}))
			Expect(hc["path"]).To(Equal("/health"))
		})

	It("Should leave a health check that already lists its headers untouched",
		func(ctx SpecContext) {
			props := v053Properties()
			hc := props["health_check"].(map[string]any)
			hc["headers"] = []any{map[string]any{"name": "X-Token", "value": "abc"}}
			delete(hc, "query_params")
			got := migrateDevice(
				ctx,
				v1.Device{Key: "http", Make: "http", Properties: props},
			)
			Expect(got.Properties).To(Equal(props))
		})

	It("Should leave a device of another make untouched", func(ctx SpecContext) {
		props := v053Properties()
		got := migrateDevice(
			ctx,
			v1.Device{Key: "lj", Make: "LabJack", Properties: props},
		)
		Expect(got.Properties).To(Equal(v053Properties()))
	})
})
