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
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/legacy/device/versions/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("EtherCATMakeMigration", func() {
	DescribeTable("Should migrate the make",
		func(ctx SpecContext, stored, want string) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			d := v1.Device{Key: "slave", Make: stored}
			Expect(gorp.NewCreate[v1.Key, v1.Device]().Entry(&d).Exec(ctx, db)).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "Device",
				Migrations: []gorp.Migration{v1.EtherCATMakeMigration},
			})).To(Succeed())
			var got v1.Device
			Expect(gorp.NewRetrieve[v1.Key, v1.Device]().
				Where(gorp.MatchKeys[v1.Key, v1.Device](d.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Make).To(Equal(want))
		},
		Entry("of a slave a released Python client made", "EtherCAT", "ethercat"),
		Entry("of another device unchanged", "NI", "NI"),
	)
})

var _ = Describe("EtherCATChannelMapMigration", func() {
	migrateDevice := func(ctx SpecContext, d v1.Device) v1.Device {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		Expect(gorp.NewCreate[v1.Key, v1.Device]().Entry(&d).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Device",
			Migrations: []gorp.Migration{v1.EtherCATChannelMapMigration},
		})).To(Succeed())
		var got v1.Device
		Expect(gorp.NewRetrieve[v1.Key, v1.Device]().
			Where(gorp.MatchKeys[v1.Key, v1.Device](d.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}
	// The properties a released Console stored for a slave with two "Input" entries
	// at 0x6000:01 and 0x6010:01, a unique "Status Word" input at 0x6020:02, and an
	// "Output" output at 0x7000:01.
	properties := func(read, write string) msgpack.EncodedJSON {
		GinkgoHelper()
		var p msgpack.EncodedJSON
		Expect(json.Unmarshal([]byte(`{
			"pdos": {
				"inputs": [
					{"name": "Input", "index": 24576, "sub_index": 1},
					{"name": "Input", "index": 24592, "sub_index": 1},
					{"name": "Status Word", "index": 24608, "sub_index": 2}
				],
				"outputs": [{"name": "Output", "index": 28672, "sub_index": 1}]
			},
			"read": {"channels": `+read+`},
			"write": {"channels": `+write+`}
		}`), &p)).To(Succeed())
		return p
	}
	channels := func(d v1.Device, dir string) map[string]any {
		GinkgoHelper()
		return d.Properties[dir].(map[string]any)["channels"].(map[string]any)
	}

	It("Should key automatic and manual read channels by address", func(
		ctx SpecContext,
	) {
		got := migrateDevice(ctx, v1.Device{
			Key:  "slave",
			Make: "ethercat",
			Properties: properties(
				`{"auto_Status Word": 1, "manual_24576_1": 2}`,
				`{}`,
			),
		})
		Expect(channels(got, "read")).To(Equal(map[string]any{
			"24608_2": float64(1),
			"24576_1": float64(2),
		}))
	})

	It("Should key a repeated name by the first entry the Driver read", func(
		ctx SpecContext,
	) {
		got := migrateDevice(ctx, v1.Device{
			Key:        "slave",
			Make:       "ethercat",
			Properties: properties(`{"auto_Input": 7}`, `{}`),
		})
		Expect(channels(got, "read")).To(Equal(map[string]any{"24576_1": float64(7)}))
	})

	It("Should match every case form the Console stored a key in", func(
		ctx SpecContext,
	) {
		got := migrateDevice(ctx, v1.Device{
			Key:        "slave",
			Make:       "ethercat",
			Properties: properties(`{}`, `{"auto_Output": 3, "auto_OutputState": 4}`),
		})
		Expect(channels(got, "write")).To(Equal(map[string]any{
			"28672_1":       float64(3),
			"state_28672_1": float64(4),
		}))
	})

	It("Should key manual write state channels by address", func(ctx SpecContext) {
		got := migrateDevice(ctx, v1.Device{
			Key:  "slave",
			Make: "ethercat",
			Properties: properties(
				`{}`,
				`{"manual_28672_1": 5, "manual_28672_1_state": 6}`,
			),
		})
		Expect(channels(got, "write")).To(Equal(map[string]any{
			"28672_1":       float64(5),
			"state_28672_1": float64(6),
		}))
	})

	It("Should keep a key that names no entry on the slave", func(ctx SpecContext) {
		got := migrateDevice(ctx, v1.Device{
			Key:        "slave",
			Make:       "ethercat",
			Properties: properties(`{"auto_Gone": 9}`, `{}`),
		})
		Expect(channels(got, "read")).To(Equal(map[string]any{"auto_Gone": float64(9)}))
	})

	It("Should key the camel case form the Console stored", func(ctx SpecContext) {
		got := migrateDevice(ctx, v1.Device{
			Key:        "slave",
			Make:       "ethercat",
			Properties: properties(`{"autoInput": 1}`, `{}`),
		})
		Expect(channels(got, "read")).To(Equal(map[string]any{"24576_1": float64(1)}))
	})

	DescribeTable("Should give an address to the first key in sorted order",
		func(ctx SpecContext, read, write, dir string, expected map[string]any) {
			got := migrateDevice(ctx, v1.Device{
				Key:        "slave",
				Make:       "ethercat",
				Properties: properties(read, write),
			})
			Expect(channels(got, dir)).To(Equal(expected))
		},
		Entry(
			"automatic and manual keys for one entry",
			`{"manual_24608_2": 2, "auto_Status Word": 1}`,
			`{}`,
			"read",
			map[string]any{"24608_2": float64(1), "manual_24608_2": float64(2)},
		),
		Entry(
			"two case forms of one state key",
			`{}`,
			`{"auto_Output_state": 2, "auto_OutputState": 1}`,
			"write",
			map[string]any{
				"state_28672_1":     float64(1),
				"auto_Output_state": float64(2),
			},
		),
	)

	It("Should leave channels unchanged on a slave without PDOs", func(
		ctx SpecContext,
	) {
		got := migrateDevice(ctx, v1.Device{
			Key:  "slave",
			Make: "ethercat",
			Properties: msgpack.EncodedJSON{
				"read": map[string]any{"channels": map[string]any{"auto_Input": 7}},
			},
		})
		Expect(
			channels(got, "read"),
		).To(Equal(map[string]any{"auto_Input": float64(7)}))
	})

	It("Should reject a slave with malformed PDOs", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		d := v1.Device{
			Key:        "slave",
			Make:       "ethercat",
			Properties: msgpack.EncodedJSON{"pdos": "garbage"},
		}
		Expect(gorp.NewCreate[v1.Key, v1.Device]().Entry(&d).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Device",
			Migrations: []gorp.Migration{v1.EtherCATChannelMapMigration},
		})).To(SatisfyAll(
			MatchError(ContainSubstring(
				"migration v60_ethercat_channel_map_addresses failed: entry slave "+
					"(transform): device slave properties: json: ",
			)),
			MatchError(ContainSubstring(`within "/pdos"`)),
		))
	})

	It("Should leave other makes unchanged", func(ctx SpecContext) {
		p := properties(`{"auto_Input": 7}`, `{}`)
		got := migrateDevice(ctx, v1.Device{Key: "ni", Make: "NI", Properties: p})
		Expect(
			channels(got, "read"),
		).To(Equal(map[string]any{"auto_Input": float64(7)}))
	})
})
