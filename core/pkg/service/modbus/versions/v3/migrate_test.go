// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v2 "github.com/synnaxlabs/synnax/pkg/service/modbus/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/modbus/versions/v3"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("MigrateReadChannel", func() {
	base := v2.BaseReadChannel{Key: "ch", Name: "ch", Channel: 7, Address: 12}
	DescribeTable(
		"Should lift a string register value to the string variant",
		func(ctx SpecContext, old v2.ReadChannel, expected v3.ReadChannel) {
			Expect(v3.MigrateReadChannel(ctx, old)).To(Equal(expected))
		},
		Entry(
			"holding register",
			v2.ReadChannel{Variant: v2.HoldingRegisterReadChannel{
				BaseReadChannel: base,
				RegisterValue: v2.RegisterValue{
					DataType:     "string",
					BytesSwapped: new(true),
					WordsSwapped: new(true),
				},
				StringLength: 6,
			}},
			v3.ReadChannel{Variant: v3.HoldingRegisterStringReadChannel{
				BaseReadChannel: base,
				RegisterString: v3.RegisterString{
					Length:       6,
					BytesSwapped: new(true),
				},
			}},
		),
		Entry(
			"input register",
			v2.ReadChannel{Variant: v2.InputRegisterReadChannel{
				BaseReadChannel: base,
				RegisterValue:   v2.RegisterValue{DataType: "string"},
				StringLength:    3,
			}},
			v3.ReadChannel{Variant: v3.InputRegisterStringReadChannel{
				BaseReadChannel: base,
				RegisterString:  v3.RegisterString{Length: 3},
			}},
		),
		Entry(
			"length below the minimum",
			v2.ReadChannel{Variant: v2.InputRegisterReadChannel{
				BaseReadChannel: base,
				RegisterValue:   v2.RegisterValue{DataType: "string"},
				StringLength:    0,
			}},
			v3.ReadChannel{Variant: v3.InputRegisterStringReadChannel{
				BaseReadChannel: base,
				RegisterString:  v3.RegisterString{Length: v3.MinStringLength},
			}},
		),
		Entry(
			"negative length",
			v2.ReadChannel{Variant: v2.HoldingRegisterReadChannel{
				BaseReadChannel: base,
				RegisterValue:   v2.RegisterValue{DataType: "string"},
				StringLength:    -4,
			}},
			v3.ReadChannel{Variant: v3.HoldingRegisterStringReadChannel{
				BaseReadChannel: base,
				RegisterString:  v3.RegisterString{Length: v3.MinStringLength},
			}},
		),
		Entry(
			"length above the maximum",
			v2.ReadChannel{Variant: v2.HoldingRegisterReadChannel{
				BaseReadChannel: base,
				RegisterValue:   v2.RegisterValue{DataType: "string"},
				StringLength:    251,
			}},
			v3.ReadChannel{Variant: v3.HoldingRegisterStringReadChannel{
				BaseReadChannel: base,
				RegisterString:  v3.RegisterString{Length: v3.MaxStringLength},
			}},
		),
	)

	DescribeTable(
		"Should keep the shape of every other variant",
		func(ctx SpecContext, old v2.ReadChannel, expected v3.ReadChannel) {
			Expect(v3.MigrateReadChannel(ctx, old)).To(Equal(expected))
		},
		Entry("no variant", v2.ReadChannel{}, v3.ReadChannel{}),
		Entry(
			"numeric register value",
			v2.ReadChannel{Variant: v2.HoldingRegisterReadChannel{
				BaseReadChannel: base,
				RegisterValue: v2.RegisterValue{
					DataType:     "float32",
					WordsSwapped: new(true),
				},
				StringLength: 9,
			}},
			v3.ReadChannel{Variant: v3.HoldingRegisterReadChannel{
				BaseReadChannel: base,
				RegisterValue: v3.RegisterValue{
					DataType:     "float32",
					WordsSwapped: new(true),
				},
			}},
		),
		Entry(
			"register value without a data type",
			v2.ReadChannel{Variant: v2.InputRegisterReadChannel{
				BaseReadChannel: base,
			}},
			v3.ReadChannel{Variant: v3.InputRegisterReadChannel{
				BaseReadChannel: base,
				RegisterValue:   v3.RegisterValue{DataType: "uint8"},
			}},
		),
		Entry(
			"coil",
			v2.ReadChannel{Variant: v2.CoilReadChannel{BaseReadChannel: base}},
			v3.ReadChannel{Variant: v3.CoilReadChannel{BaseReadChannel: base}},
		),
	)
})

var _ = Describe("MigrateWriteChannel", func() {
	base := v2.BaseWriteChannel{Key: "ch", Name: "ch", Channel: 7, Address: 12}
	DescribeTable(
		"Should lift each v2 variant to the same v3 variant",
		func(ctx SpecContext, old v2.WriteChannel, expected v3.WriteChannel) {
			Expect(v3.MigrateWriteChannel(ctx, old)).To(Equal(expected))
		},
		Entry("no variant", v2.WriteChannel{}, v3.WriteChannel{}),
		Entry(
			"coil",
			v2.WriteChannel{Variant: v2.CoilWriteChannel{BaseWriteChannel: base}},
			v3.WriteChannel{Variant: v3.CoilWriteChannel{BaseWriteChannel: base}},
		),
		Entry(
			"holding register",
			v2.WriteChannel{Variant: v2.HoldingRegisterWriteChannel{
				BaseWriteChannel: base,
				RegisterValue: v2.RegisterValue{
					DataType:     "uint8",
					BytesSwapped: new(true),
				},
			}},
			v3.WriteChannel{Variant: v3.HoldingRegisterWriteChannel{
				BaseWriteChannel: base,
				RegisterValue: v3.RegisterValue{
					DataType:     "uint8",
					BytesSwapped: new(true),
				},
			}},
		),
		Entry(
			"holding register without a data type",
			v2.WriteChannel{Variant: v2.HoldingRegisterWriteChannel{
				BaseWriteChannel: base,
			}},
			v3.WriteChannel{Variant: v3.HoldingRegisterWriteChannel{
				BaseWriteChannel: base,
				RegisterValue:    v3.RegisterValue{DataType: "uint8"},
			}},
		),
	)
})

var _ = Describe("RegisterValue", func() {
	It("Should read a v2 register value without a data type as uint8",
		func(ctx SpecContext) {
			var old v2.ReadConfig
			Expect(json.Unmarshal([]byte(`{"channels": [
				{"type": "holding_register", "key": "a"},
				{"type": "input_register", "key": "b"}
			]}`), &old)).To(Succeed())
			cfg := MustSucceed(v3.MigrateReadConfig(ctx, old))
			Expect(cfg.Channels).To(HaveLen(2))
			holding := cfg.Channels[0].Variant.(v3.HoldingRegisterReadChannel)
			Expect(holding.DataType).To(BeEquivalentTo("uint8"))
			input := cfg.Channels[1].Variant.(v3.InputRegisterReadChannel)
			Expect(input.DataType).To(BeEquivalentTo("uint8"))
		},
	)

	It("Should read an empty v2 data type as uint8", func(ctx SpecContext) {
		var read v2.ReadConfig
		Expect(json.Unmarshal([]byte(`{"channels": [
			{"type": "holding_register", "key": "a", "data_type": ""}
		]}`), &read)).To(Succeed())
		readCfg := MustSucceed(v3.MigrateReadConfig(ctx, read))
		holding := readCfg.Channels[0].Variant.(v3.HoldingRegisterReadChannel)
		Expect(holding.DataType).To(BeEquivalentTo("uint8"))
		var write v2.WriteConfig
		Expect(json.Unmarshal([]byte(`{"channels": [
			{"type": "holding_register", "key": "b", "data_type": ""}
		]}`), &write)).To(Succeed())
		writeCfg := MustSucceed(v3.MigrateWriteConfig(ctx, write))
		written := writeCfg.Channels[0].Variant.(v3.HoldingRegisterWriteChannel)
		Expect(written.DataType).To(BeEquivalentTo("uint8"))
	})

	It("Should read a v3 register value without a data type as uint16", func() {
		var cfg v3.WriteConfig
		Expect(json.Unmarshal(
			[]byte(`{"channels": [{"type": "holding_register", "key": "a"}]}`),
			&cfg,
		)).To(Succeed())
		holding := cfg.Channels[0].Variant.(v3.HoldingRegisterWriteChannel)
		Expect(holding.DataType).To(BeEquivalentTo("uint16"))
	})
})

var _ = Describe("MigrateReadConfig", func() {
	It("Should lift every channel of the config", func(ctx SpecContext) {
		cfg := MustSucceed(v3.MigrateReadConfig(ctx, v2.ReadConfig{
			Device: "dev",
			Channels: []v2.ReadChannel{
				{Variant: v2.HoldingRegisterReadChannel{
					BaseReadChannel: v2.BaseReadChannel{Key: "str"},
					RegisterValue:   v2.RegisterValue{DataType: "string"},
					StringLength:    300,
				}},
				{Variant: v2.CoilReadChannel{
					BaseReadChannel: v2.BaseReadChannel{Key: "coil"},
				}},
			},
		}))
		Expect(cfg.Device).To(Equal("dev"))
		Expect(cfg.Channels).To(HaveLen(2))
		str, ok := cfg.Channels[0].Variant.(v3.HoldingRegisterStringReadChannel)
		Expect(ok).To(BeTrue())
		Expect(str.Key).To(Equal("str"))
		Expect(str.Length).To(BeEquivalentTo(v3.MaxStringLength))
		Expect(cfg.Channels[1].Variant).To(BeAssignableToTypeOf(v3.CoilReadChannel{}))
	})
})

var _ = Describe("ReadMigration", func() {
	It("Should clamp stored string lengths and log each clamp", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		str := func(key string, length int32) v2.ReadChannel {
			return v2.ReadChannel{Variant: v2.InputRegisterReadChannel{
				BaseReadChannel: v2.BaseReadChannel{Key: key},
				RegisterValue:   v2.RegisterValue{DataType: "string"},
				StringLength:    length,
			}}
		}
		stored := v2.ReadConfig{
			Key:      uuid.New(),
			Channels: []v2.ReadChannel{str("short", 0), str("ok", 8), str("long", 400)},
		}
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v2.ReadConfig]{
			DB: db,
		}))
		Expect(gorp.NewCreate[uuid.UUID, v2.ReadConfig]().
			Entry(&stored).Exec(ctx, db)).To(Succeed())
		core, logs := observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			DB:              db,
			Namespace:       "modbus_read",
			Migrations:      []gorp.Migration{v3.ReadMigration},
		})).To(Succeed())
		var got v3.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v3.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v3.ReadConfig](stored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		lengths := make([]uint16, len(got.Channels))
		for i, ch := range got.Channels {
			lengths[i] = ch.Variant.(v3.InputRegisterStringReadChannel).Length
		}
		Expect(lengths).To(Equal([]uint16{1, 8, 250}))
		clamped := logs.FilterMessage(
			"clamped the length of a Modbus string channel",
		).All()
		Expect(clamped).To(HaveLen(2))
		Expect(clamped[0].ContextMap()).To(And(
			HaveKeyWithValue("config", stored.Key.String()),
			HaveKeyWithValue("channel", "short"),
			HaveKeyWithValue("field", "length"),
			HaveKeyWithValue("old", BeEquivalentTo(0)),
			HaveKeyWithValue("new", BeEquivalentTo(1)),
		))
		Expect(clamped[1].ContextMap()).To(And(
			HaveKeyWithValue("channel", "long"),
			HaveKeyWithValue("old", BeEquivalentTo(400)),
			HaveKeyWithValue("new", BeEquivalentTo(250)),
		))
	})
})

var _ = Describe("MigrateWriteConfig", func() {
	It("Should lift every channel of the config", func(ctx SpecContext) {
		cfg := MustSucceed(v3.MigrateWriteConfig(ctx, v2.WriteConfig{
			Device: "dev",
			Channels: []v2.WriteChannel{
				{Variant: v2.HoldingRegisterWriteChannel{
					BaseWriteChannel: v2.BaseWriteChannel{Key: "reg"},
					RegisterValue:    v2.RegisterValue{DataType: "float32"},
				}},
				{Variant: v2.CoilWriteChannel{
					BaseWriteChannel: v2.BaseWriteChannel{Key: "coil"},
				}},
			},
		}))
		Expect(cfg.Device).To(Equal("dev"))
		Expect(cfg.Channels).To(Equal([]v3.WriteChannel{
			{Variant: v3.HoldingRegisterWriteChannel{
				BaseWriteChannel: v3.BaseWriteChannel{Key: "reg"},
				RegisterValue:    v3.RegisterValue{DataType: "float32"},
			}},
			{Variant: v3.CoilWriteChannel{
				BaseWriteChannel: v3.BaseWriteChannel{Key: "coil"},
			}},
		}))
	})
})

var _ = Describe("WriteMigration", func() {
	It("Should lift stored write configs in place", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		stored := v2.WriteConfig{
			Key: uuid.New(),
			Channels: []v2.WriteChannel{{Variant: v2.HoldingRegisterWriteChannel{
				BaseWriteChannel: v2.BaseWriteChannel{Key: "reg"},
			}}},
		}
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v2.WriteConfig]{
			DB: db,
		}))
		Expect(gorp.NewCreate[uuid.UUID, v2.WriteConfig]().
			Entry(&stored).Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "modbus_write",
			Migrations: []gorp.Migration{v3.WriteMigration},
		})).To(Succeed())
		var got v3.WriteConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v3.WriteConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v3.WriteConfig](stored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Channels).To(Equal([]v3.WriteChannel{
			{Variant: v3.HoldingRegisterWriteChannel{
				BaseWriteChannel: v3.BaseWriteChannel{Key: "reg"},
				RegisterValue:    v3.RegisterValue{DataType: "uint8"},
			}},
		}))
	})
})
