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
			"coil",
			v2.ReadChannel{Variant: v2.CoilReadChannel{BaseReadChannel: base}},
			v3.ReadChannel{Variant: v3.CoilReadChannel{BaseReadChannel: base}},
		),
		Entry(
			"register bit",
			v2.ReadChannel{Variant: v2.InputRegisterBitReadChannel{
				BaseReadChannel: base,
				RegisterBit:     v2.RegisterBit{Bit: 3},
			}},
			v3.ReadChannel{Variant: v3.InputRegisterBitReadChannel{
				BaseReadChannel: base,
				RegisterBit:     v3.RegisterBit{Bit: 3},
			}},
		),
	)
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
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v2.ReadConfig]{
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
