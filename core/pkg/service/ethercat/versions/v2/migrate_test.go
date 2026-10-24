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
	"context"
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

type pdoEntry struct {
	name     string
	index    uint16
	subIndex uint8
	dataType string
}

func slaveProperties(inputs, outputs []pdoEntry) msgpack.EncodedJSON {
	list := func(entries []pdoEntry) []any {
		return lo.Map(entries, func(e pdoEntry, _ int) any {
			return map[string]any{
				"name":       e.name,
				"index":      e.index,
				"sub_index":  e.subIndex,
				"bit_length": 8,
				"data_type":  e.dataType,
			}
		})
	}
	return msgpack.EncodedJSON{
		"pdos": map[string]any{"inputs": list(inputs), "outputs": list(outputs)},
	}
}

// el1008 has eight entries named "Input", one per PDO.
func el1008() msgpack.EncodedJSON {
	inputs := make([]pdoEntry, 8)
	for i := range inputs {
		inputs[i] = pdoEntry{
			name:     "Input",
			index:    0x6000 + uint16(i)*0x10,
			subIndex: 1,
			dataType: "uint8",
		}
	}
	return slaveProperties(inputs, nil)
}

// mixed lists "Sync error" twice at one address and "Repeated" at two addresses.
func mixed() msgpack.EncodedJSON {
	return slaveProperties(
		[]pdoEntry{
			{name: "Value", index: 0x6000, subIndex: 0x11, dataType: "int16"},
			{name: "Sync error", index: 0x1C32, subIndex: 0x20, dataType: "uint8"},
			{name: "Sync error", index: 0x1C32, subIndex: 0x20, dataType: "uint8"},
		},
		[]pdoEntry{
			{name: "Output", index: 0x7000, subIndex: 1, dataType: "uint8"},
			{name: "Repeated", index: 0x7010, subIndex: 1, dataType: "uint8"},
			{name: "Repeated", index: 0x7020, subIndex: 1, dataType: "uint8"},
		},
	)
}

func unresolvedRead(device, pdo string) v2.ReadChannel {
	return v2.ReadChannel{Variant: v2.UnresolvedReadChannel{
		BaseReadChannel: v1.BaseReadChannel{Key: pdo, Device: device, Channel: 12},
		Pdo:             pdo,
	}}
}

func unresolvedWrite(device, pdo string) v2.WriteChannel {
	return v2.WriteChannel{Variant: v2.UnresolvedWriteChannel{
		Key:          pdo,
		Device:       device,
		CmdChannel:   3,
		StateChannel: 4,
		Pdo:          pdo,
	}}
}

func automaticRead(device, pdo string) v1.ReadChannel {
	return v1.ReadChannel{Variant: v1.AutomaticReadChannel{
		BaseReadChannel: v1.BaseReadChannel{Key: pdo, Device: device, Channel: 12},
		Pdo:             pdo,
	}}
}

func automaticWrite(device, pdo string) v1.WriteChannel {
	return v1.WriteChannel{Variant: v1.AutomaticWriteChannel{
		Key:          pdo,
		Device:       device,
		CmdChannel:   3,
		StateChannel: 4,
		Pdo:          pdo,
	}}
}

var _ = Describe("Migrate", func() {
	var (
		requested [][]string
		props     map[string]msgpack.EncodedJSON
	)
	devices := func(
		_ context.Context,
		_ gorp.Tx,
		keys []string,
	) (map[string]msgpack.EncodedJSON, error) {
		requested = append(requested, keys)
		return lo.PickByKeys(props, keys), nil
	}
	BeforeEach(func() {
		requested = nil
		props = map[string]msgpack.EncodedJSON{
			"el1008":    el1008(),
			"mixed":     mixed(),
			"malformed": {"pdos": "garbage"},
		}
	})

	Describe("MigrateReadChannel", func() {
		It("Should keep a manual channel's address", func(ctx SpecContext) {
			old := v1.ReadChannel{Variant: v1.ManualReadChannel{
				BaseReadChannel: v1.BaseReadChannel{Key: "k", Name: "n", Device: "d"},
				PDOAddress: v1.PDOAddress{
					Index: 0x6010, SubIndex: 2, BitLength: 32, DataType: "float32",
				},
			}}
			Expect(v2.MigrateReadChannel(ctx, old)).To(Equal(v2.ReadChannel{
				Variant: v2.EntryReadChannel{
					BaseReadChannel: v1.BaseReadChannel{
						Key:    "k",
						Name:   "n",
						Device: "d",
					},
					Index:    0x6010,
					SubIndex: 2,
					DataType: "float32",
					Scale:    v2.Scale{Variant: v2.NoneScale{}},
				},
			}))
		})

		It("Should mark an automatic channel unresolved", func(ctx SpecContext) {
			Expect(v2.MigrateReadChannel(ctx, automaticRead("d", "Input"))).
				To(Equal(unresolvedRead("d", "Input")))
		})

		It("Should keep a channel with no variant empty", func(ctx SpecContext) {
			Expect(v2.MigrateReadChannel(ctx, v1.ReadChannel{})).
				To(Equal(v2.ReadChannel{}))
		})
	})

	Describe("MigrateWriteChannel", func() {
		It("Should keep a manual channel's address", func(ctx SpecContext) {
			old := v1.WriteChannel{Variant: v1.ManualWriteChannel{
				Key:          "k",
				Device:       "d",
				CmdChannel:   3,
				StateChannel: 4,
				PDOAddress: v1.PDOAddress{
					Index: 0x7010, SubIndex: 2, BitLength: 16, DataType: "uint16",
				},
			}}
			Expect(v2.MigrateWriteChannel(ctx, old)).To(Equal(v2.WriteChannel{
				Variant: v2.EntryWriteChannel{
					Key:          "k",
					Device:       "d",
					CmdChannel:   3,
					StateChannel: 4,
					Index:        0x7010,
					SubIndex:     2,
					DataType:     "uint16",
				},
			}))
		})

		It("Should mark an automatic channel unresolved", func(ctx SpecContext) {
			Expect(v2.MigrateWriteChannel(ctx, automaticWrite("d", "Output"))).
				To(Equal(unresolvedWrite("d", "Output")))
		})

		It("Should keep a channel with no variant empty", func(ctx SpecContext) {
			Expect(v2.MigrateWriteChannel(ctx, v1.WriteChannel{})).
				To(Equal(v2.WriteChannel{}))
		})
	})

	Describe("MigrateReadConfig", func() {
		It("Should keep the task fields and lift every channel", func(
			ctx SpecContext,
		) {
			var old v1.ReadConfig
			old.SampleRate = 10
			old.Channels = []v1.ReadChannel{automaticRead("d", "Input")}
			cfg := MustSucceed(v2.MigrateReadConfig(ctx, old))
			Expect(cfg.SampleRate).To(BeEquivalentTo(10))
			Expect(
				cfg.Channels,
			).To(Equal([]v2.ReadChannel{unresolvedRead("d", "Input")}))
		})
	})

	Describe("MigrateWriteConfig", func() {
		It("Should keep the task fields and lift every channel", func(
			ctx SpecContext,
		) {
			var old v1.WriteConfig
			old.StateRate = 20
			old.Channels = []v1.WriteChannel{automaticWrite("d", "Output")}
			cfg := MustSucceed(v2.MigrateWriteConfig(ctx, old))
			Expect(cfg.StateRate).To(BeEquivalentTo(20))
			Expect(cfg.Channels).
				To(Equal([]v2.WriteChannel{unresolvedWrite("d", "Output")}))
		})
	})

	Describe("ResolveReadConfig", func() {
		resolveRead := func(ctx SpecContext, channels ...v2.ReadChannel) v2.ReadConfig {
			GinkgoHelper()
			cfg := v2.ReadConfig{Channels: channels}
			Expect(v2.ResolveReadConfig(ctx, nil, devices, &cfg)).To(Succeed())
			return cfg
		}

		It("Should address a unique entry name with its data type", func(
			ctx SpecContext,
		) {
			cfg := resolveRead(ctx, unresolvedRead("mixed", "Value"))
			Expect(cfg.Channels[0].Variant).To(Equal(v2.EntryReadChannel{
				Key:      "Value",
				Device:   "mixed",
				Channel:  12,
				Index:    0x6000,
				SubIndex: 0x11,
				DataType: "int16",
				Scale:    v2.Scale{Variant: v2.NoneScale{}},
			}))
		})

		It("Should address a name repeated at one address", func(ctx SpecContext) {
			cfg := resolveRead(ctx, unresolvedRead("mixed", "Sync error"))
			Expect(cfg.Channels[0].Variant).To(Equal(v2.EntryReadChannel{
				Key:      "Sync error",
				Device:   "mixed",
				Channel:  12,
				Index:    0x1C32,
				SubIndex: 0x20,
				DataType: "uint8",
				Scale:    v2.Scale{Variant: v2.NoneScale{}},
			}))
		})

		DescribeTable("Should leave a channel unresolved",
			func(ctx SpecContext, device, pdo string) {
				cfg := resolveRead(ctx, unresolvedRead(device, pdo))
				Expect(cfg.Channels[0]).To(Equal(unresolvedRead(device, pdo)))
			},
			Entry("on a name at several addresses", "el1008", "Input"),
			Entry("on a name the slave does not list", "mixed", "Missing"),
			Entry("on an output entry name", "mixed", "Output"),
			Entry("on a missing device", "gone", "Value"),
		)

		It("Should read no devices when every channel has an address", func(
			ctx SpecContext,
		) {
			entry := v2.ReadChannel{Variant: v2.EntryReadChannel{
				BaseReadChannel: v1.BaseReadChannel{Key: "k", Device: "malformed"},
				Index:           0x6010,
				SubIndex:        2,
				DataType:        "float32",
			}}
			Expect(resolveRead(ctx, entry).Channels).To(Equal([]v2.ReadChannel{entry}))
			Expect(requested).To(BeEmpty())
		})

		It("Should read each referenced device once", func(ctx SpecContext) {
			resolveRead(
				ctx,
				unresolvedRead("mixed", "Value"),
				unresolvedRead("mixed", "Sync error"),
				unresolvedRead("el1008", "Input"),
			)
			Expect(requested).To(Equal([][]string{{"mixed", "el1008"}}))
		})

		It("Should reject a referenced device with malformed properties", func(
			ctx SpecContext,
		) {
			cfg := v2.ReadConfig{
				Channels: []v2.ReadChannel{unresolvedRead("malformed", "Value")},
			}
			Expect(v2.ResolveReadConfig(ctx, nil, devices, &cfg)).To(SatisfyAll(
				MatchError(ContainSubstring("device malformed properties: json: ")),
				MatchError(ContainSubstring(`within "/pdos"`)),
			))
		})

		It("Should return the error of the device lookup", func(ctx SpecContext) {
			cfg := v2.ReadConfig{
				Channels: []v2.ReadChannel{unresolvedRead("mixed", "Value")},
			}
			Expect(v2.ResolveReadConfig(
				ctx,
				nil,
				func(context.Context, gorp.Tx, []string) (map[string]msgpack.EncodedJSON, error) {
					return nil, errors.New("lookup failed")
				},
				&cfg,
			)).To(MatchError("lookup failed"))
		})
	})

	Describe("ResolveWriteConfig", func() {
		resolveWrite := func(
			ctx SpecContext,
			channels ...v2.WriteChannel,
		) v2.WriteConfig {
			GinkgoHelper()
			cfg := v2.WriteConfig{Channels: channels}
			Expect(v2.ResolveWriteConfig(ctx, nil, devices, &cfg)).To(Succeed())
			return cfg
		}

		It("Should address a unique output entry name", func(ctx SpecContext) {
			cfg := resolveWrite(ctx, unresolvedWrite("mixed", "Output"))
			Expect(cfg.Channels[0].Variant).To(Equal(v2.EntryWriteChannel{
				Key:          "Output",
				Device:       "mixed",
				CmdChannel:   3,
				StateChannel: 4,
				Index:        0x7000,
				SubIndex:     1,
				DataType:     "uint8",
			}))
		})

		DescribeTable("Should leave a channel unresolved",
			func(ctx SpecContext, pdo string) {
				cfg := resolveWrite(ctx, unresolvedWrite("mixed", pdo))
				Expect(cfg.Channels[0]).To(Equal(unresolvedWrite("mixed", pdo)))
			},
			Entry("on a name at several addresses", "Repeated"),
			Entry("on an input entry name", "Value"),
		)

		It("Should reject a referenced device with malformed properties", func(
			ctx SpecContext,
		) {
			cfg := v2.WriteConfig{
				Channels: []v2.WriteChannel{unresolvedWrite("malformed", "Output")},
			}
			Expect(v2.ResolveWriteConfig(ctx, nil, devices, &cfg)).
				To(MatchError(ContainSubstring("device malformed properties: json: ")))
		})
	})

	Describe("Stored migrations", func() {
		var db *gorp.DB
		BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

		It("Should lift stored read configs and resolve their channels", func(
			ctx SpecContext,
		) {
			var old v1.ReadConfig
			old.Key = uuid.New()
			old.Channels = []v1.ReadChannel{
				automaticRead("mixed", "Value"),
				automaticRead("el1008", "Input"),
			}
			Expect(gorp.WrapWriter[uuid.UUID, v1.ReadConfig](db).Set(ctx, old)).
				To(Succeed())
			table := MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v2.ReadConfig]{
					DB:         db,
					Migrations: []migrate.Migration{v2.NewReadMigration(devices)},
				}),
			)
			var cfg v2.ReadConfig
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[uuid.UUID, v2.ReadConfig](old.Key)).
				Entry(&cfg).
				Exec(ctx, db)).To(Succeed())
			Expect(cfg.Channels[0].Variant).
				To(BeAssignableToTypeOf(v2.EntryReadChannel{}))
			Expect(cfg.Channels[1]).To(Equal(unresolvedRead("el1008", "Input")))
		})

		It("Should lift stored write configs and resolve their channels", func(
			ctx SpecContext,
		) {
			var old v1.WriteConfig
			old.Key = uuid.New()
			old.Channels = []v1.WriteChannel{
				automaticWrite("mixed", "Output"),
				automaticWrite("mixed", "Repeated"),
			}
			Expect(gorp.WrapWriter[uuid.UUID, v1.WriteConfig](db).Set(ctx, old)).
				To(Succeed())
			table := MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v2.WriteConfig]{
					DB:         db,
					Migrations: []migrate.Migration{v2.NewWriteMigration(devices)},
				}),
			)
			var cfg v2.WriteConfig
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[uuid.UUID, v2.WriteConfig](old.Key)).
				Entry(&cfg).
				Exec(ctx, db)).To(Succeed())
			Expect(cfg.Channels[0].Variant).
				To(BeAssignableToTypeOf(v2.EntryWriteChannel{}))
			Expect(cfg.Channels[1]).To(Equal(unresolvedWrite("mixed", "Repeated")))
		})

		It("Should fail when a stored config references a malformed device", func(
			ctx SpecContext,
		) {
			var old v1.ReadConfig
			old.Key = uuid.New()
			old.Channels = []v1.ReadChannel{automaticRead("malformed", "Value")}
			Expect(gorp.WrapWriter[uuid.UUID, v1.ReadConfig](db).Set(ctx, old)).
				To(Succeed())
			Expect(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v2.ReadConfig]{
				DB:         db,
				Migrations: []migrate.Migration{v2.NewReadMigration(devices)},
			})).Error().To(MatchError(ContainSubstring(
				fmt.Sprintf("config %s: device malformed properties", old.Key),
			)))
		})
	})
})
