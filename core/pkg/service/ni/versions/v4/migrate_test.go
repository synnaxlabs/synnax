// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v4"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func funcGen(frequency float64) v3.AOChannel {
	return v3.AOChannel{Variant: v3.AOFuncGenChannel{
		BaseAOChannel: v3.BaseAOChannel{Key: "gen"},
		Frequency:     frequency,
	}}
}

var _ = Describe("MigrateAnalogWriteConfig", func() {
	DescribeTable("Should make a function generator frequency positive",
		func(ctx SpecContext, frequency, want float64) {
			got := MustSucceed(v4.MigrateAnalogWriteConfig(ctx, v3.AnalogWriteConfig{
				Channels: []v3.AOChannel{funcGen(frequency)},
			}))
			Expect(got.Channels[0].Variant.(v4.AOFuncGenChannel).Frequency).
				To(Equal(want))
		},
		Entry("missing", 0.0, 1.0),
		Entry("negative", -20.0, 1.0),
		Entry("NaN", math.NaN(), 1.0),
		Entry("infinite", math.Inf(1), 1.0),
		Entry("positive", 12.5, 12.5),
	)

	It("Should keep a commanded output", func(ctx SpecContext) {
		got := MustSucceed(v4.MigrateAnalogWriteConfig(ctx, v3.AnalogWriteConfig{
			Channels: []v3.AOChannel{{Variant: v3.AOVoltageChannel{
				BaseAOChannel: v3.BaseAOChannel{Key: "ao", Port: 2},
			}}},
		}))
		var want v4.AOVoltageChannel
		want.Key, want.Port = "ao", 2
		Expect(got.Channels[0].Variant).To(Equal(v4.AOChannelVariant(want)))
	})
})

var _ = Describe("AnalogWriteMigration", func() {
	It("Should make a stored frequency positive and log the change",
		func(ctx SpecContext) {
			stored := v3.AnalogWriteConfig{
				Key:      uuid.New(),
				Channels: []v3.AOChannel{funcGen(0), funcGen(5)},
			}
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[uuid.UUID, v3.AnalogWriteConfig]{DB: db},
			))
			Expect(gorp.NewCreate[uuid.UUID, v3.AnalogWriteConfig]().
				Entry(&stored).Exec(ctx, db)).To(Succeed())
			core, logs := observer.New(zapcore.WarnLevel)
			logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
				ZapLogger: zap.New(core),
			}))
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
				DB:              db,
				Namespace:       "ni_analog_write_config",
				Migrations:      []gorp.Migration{v4.AnalogWriteMigration},
			})).To(Succeed())
			var got v4.AnalogWriteConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v4.AnalogWriteConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v4.AnalogWriteConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Channels[0].Variant.(v4.AOFuncGenChannel).Frequency).
				To(Equal(1.0))
			Expect(got.Channels[1].Variant.(v4.AOFuncGenChannel).Frequency).
				To(Equal(5.0))
			entries := logs.FilterMessage("narrowed an NI channel field").All()
			Expect(entries).To(HaveLen(1))
			Expect(entries[0].ContextMap()).To(HaveKeyWithValue("field", "Frequency"))
			Expect(entries[0].ContextMap()).To(HaveKeyWithValue("new", 1.0))
		},
	)
})

var _ = Describe("MigrateCounterReadConfig", func() {
	It("Should keep every counter channel", func(ctx SpecContext) {
		got := MustSucceed(v4.MigrateCounterReadConfig(ctx, v3.CounterReadConfig{
			Channels: []v3.CIChannel{{Variant: v3.CIFrequencyChannel{
				BaseCIChannel: v3.BaseCIChannel{Key: "ctr", Port: 1},
				Divisor:       8,
			}}},
		}))
		Expect(got.Channels).To(HaveLen(1))
		ch := got.Channels[0].Variant.(v4.CIFrequencyChannel)
		Expect(ch.Key).To(Equal("ctr"))
		Expect(ch.Port).To(BeEquivalentTo(1))
		Expect(ch.Divisor).To(Equal(uint32(8)))
	})
})

var _ = Describe("MigrateDIChannel", func() {
	DescribeTable("Should clamp the port and disable the channel",
		func(ctx SpecContext, port uint32, want uint16, disabled bool) {
			got := MustSucceed(v4.MigrateDIChannel(
				ctx,
				v3.DIChannel{Key: "a", Port: port, Line: 7},
			))
			Expect(got.Port).To(Equal(want))
			Expect(got.Line).To(Equal(uint8(7)))
			Expect(got.Disabled).To(Equal(disabled))
		},
		Entry("in range", uint32(2), uint16(2), false),
		Entry("at the maximum", uint32(math.MaxUint16), uint16(math.MaxUint16), false),
		Entry("above uint16", uint32(70000), uint16(math.MaxUint16), true),
	)
})

var _ = Describe("MigrateDOChannel", func() {
	It("Should clamp a port above uint16 and disable the channel",
		func(ctx SpecContext) {
			got := MustSucceed(v4.MigrateDOChannel(
				ctx,
				v3.DOChannel{Key: "a", Port: 65536, Line: 2},
			))
			Expect(got.Port).To(Equal(uint16(math.MaxUint16)))
			Expect(got.Line).To(Equal(uint8(2)))
			Expect(got.Disabled).To(BeTrue())
		},
	)
})

var _ = Describe("MigrateBaseAIChannel", func() {
	DescribeTable("Should clamp the port and disable the channel",
		func(ctx SpecContext, port uint32, want uint16, disabled bool) {
			got := MustSucceed(
				v4.MigrateBaseAIChannel(ctx, v3.BaseAIChannel{Port: port}),
			)
			Expect(got.Port).To(Equal(want))
			Expect(got.Disabled).To(Equal(disabled))
		},
		Entry("in range", uint32(7), uint16(7), false),
		Entry("above uint16", uint32(math.MaxUint32), uint16(math.MaxUint16), true),
	)
})

var _ = Describe("MigrateBaseAOChannel", func() {
	It("Should clamp a port above uint16 and disable the channel",
		func(ctx SpecContext) {
			got := MustSucceed(
				v4.MigrateBaseAOChannel(ctx, v3.BaseAOChannel{Port: 70000}),
			)
			Expect(got.Port).To(Equal(uint16(math.MaxUint16)))
			Expect(got.Disabled).To(BeTrue())
		},
	)
})

var _ = Describe("MigrateBaseCIChannel", func() {
	It("Should clamp a port above uint16 and disable the channel",
		func(ctx SpecContext) {
			got := MustSucceed(
				v4.MigrateBaseCIChannel(ctx, v3.BaseCIChannel{Port: 70000}),
			)
			Expect(got.Port).To(Equal(uint16(math.MaxUint16)))
			Expect(got.Disabled).To(BeTrue())
		},
	)
})

var _ = Describe("MigrateCJC", func() {
	It("Should clamp a channel port above uint16", func(ctx SpecContext) {
		got := MustSucceed(
			v4.MigrateCJC(ctx, v3.CJC{Variant: v3.ChanCJC{Port: 65536}}),
		)
		Expect(got.Variant).To(Equal(v4.CJCVariant(v4.ChanCJC{Port: math.MaxUint16})))
	})

	It("Should keep a constant source", func(ctx SpecContext) {
		got := MustSucceed(
			v4.MigrateCJC(ctx, v3.CJC{Variant: v3.ConstValCJC{Val: 25}}),
		)
		Expect(got.Variant).To(Equal(v4.CJCVariant(v4.ConstValCJC{Val: 25})))
	})
})

var _ = Describe("MigrateTerminal", func() {
	DescribeTable("Should keep the terminal configuration",
		func(ctx SpecContext, config v3.TerminalConfig) {
			got := MustSucceed(
				v4.MigrateTerminal(ctx, v3.Terminal{TerminalConfig: config}),
			)
			Expect(string(got.TerminalConfig)).To(Equal(string(config)))
		},
		Entry("default", v3.TerminalConfigCfgDefault),
		Entry("RSE", v3.TerminalConfigRse),
		Entry("NRSE", v3.TerminalConfigNrse),
		Entry("Diff", v3.TerminalConfigDiff),
		Entry("PseudoDiff", v3.TerminalConfigPseudoDiff),
	)
})

var _ = Describe("MigrateAnalogReadConfig", func() {
	It("Should disable a thermocouple whose cold-junction port clamps",
		func(ctx SpecContext) {
			got := MustSucceed(v4.MigrateAnalogReadConfig(ctx, v3.AnalogReadConfig{
				Channels: []v3.AIChannel{
					{Variant: v3.AIThermocoupleChannel{
						BaseAIChannel: v3.BaseAIChannel{Key: "tc", Port: 1},
						Cjc:           v3.CJC{Variant: v3.ChanCJC{Port: 65536}},
					}},
					{Variant: v3.AIVoltageChannel{
						BaseAIChannel: v3.BaseAIChannel{Key: "v", Port: 2},
					}},
				},
			}))
			tc := got.Channels[0].Variant.(v4.AIThermocoupleChannel)
			Expect(tc.Disabled).To(BeTrue())
			Expect(tc.Port).To(Equal(uint16(1)))
			Expect(tc.Cjc.Variant).
				To(Equal(v4.CJCVariant(v4.ChanCJC{Port: math.MaxUint16})))
			v := got.Channels[1].Variant.(v4.AIVoltageChannel)
			Expect(v.Disabled).To(BeFalse())
		},
	)
})

func runStored[I, O gorp.Entry[uuid.UUID]](
	ctx SpecContext,
	namespace string,
	migration gorp.Migration,
	old I,
) (O, []any) {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, I]{DB: db}))
	Expect(gorp.NewCreate[uuid.UUID, I]().Entry(&old).Exec(ctx, db)).To(Succeed())
	core, logs := observer.New(zapcore.WarnLevel)
	logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
		ZapLogger: zap.New(core),
	}))
	Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
		Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
		DB:              db,
		Namespace:       namespace,
		Migrations:      []gorp.Migration{migration},
	})).To(Succeed())
	var got O
	Expect(gorp.NewRetrieve[uuid.UUID, O]().
		Where(gorp.MatchKeys[uuid.UUID, O](old.GorpKey())).
		Entry(&got).Exec(ctx, db)).To(Succeed())
	var fields []any
	for _, e := range logs.FilterMessage("narrowed an NI channel field").All() {
		fields = append(fields, e.ContextMap()["field"])
	}
	return got, fields
}

var noScale = v3.CustomScale{CustomScale: v3.Scale{Variant: v3.NoneScale{}}}

var _ = Describe("AnalogReadMigration", func() {
	It("Should clamp stored ports, disable their channels, and log each change",
		func(ctx SpecContext) {
			got, fields := runStored[v3.AnalogReadConfig, v4.AnalogReadConfig](
				ctx,
				"ni_analog_read_config",
				v4.AnalogReadMigration,
				v3.AnalogReadConfig{
					Key: uuid.New(),
					Channels: []v3.AIChannel{
						{Variant: v3.AIThermocoupleChannel{
							BaseAIChannel: v3.BaseAIChannel{Key: "tc", Port: 1},
							Cjc:           v3.CJC{Variant: v3.ChanCJC{Port: 65536}},
						}},
						{Variant: v3.AIVoltageChannel{
							BaseAIChannel: v3.BaseAIChannel{Key: "v", Port: 70000},
							CustomScale:   noScale,
						}},
					},
				},
			)
			tc := got.Channels[0].Variant.(v4.AIThermocoupleChannel)
			Expect(tc.Disabled).To(BeTrue())
			Expect(tc.Cjc.Variant).
				To(Equal(v4.CJCVariant(v4.ChanCJC{Port: math.MaxUint16})))
			v := got.Channels[1].Variant.(v4.AIVoltageChannel)
			Expect(v.Port).To(Equal(uint16(math.MaxUint16)))
			Expect(v.Disabled).To(BeTrue())
			Expect(fields).To(ConsistOf("Cjc.Port", "Disabled", "Port", "Disabled"))
		},
	)
})

var _ = Describe("MigrateDigitalReadConfig", func() {
	It("Should clamp each channel port and disable its channel", func(ctx SpecContext) {
		got := MustSucceed(v4.MigrateDigitalReadConfig(ctx, v3.DigitalReadConfig{
			Channels: []v3.DIChannel{
				{Key: "ok", Port: 1, Line: 2},
				{Key: "bad", Port: 70000, Line: 3},
			},
		}))
		Expect(got.Channels).To(Equal([]v4.DIChannel{
			{Key: "ok", Port: 1, Line: 2},
			{Key: "bad", Port: math.MaxUint16, Line: 3, Disabled: true},
		}))
	})
})

var _ = Describe("MigrateDigitalWriteConfig", func() {
	It("Should clamp each channel port and disable its channel", func(ctx SpecContext) {
		got := MustSucceed(v4.MigrateDigitalWriteConfig(ctx, v3.DigitalWriteConfig{
			Channels: []v3.DOChannel{
				{Key: "ok", Port: 1, Line: 2},
				{Key: "bad", Port: 70000, Line: 3},
			},
		}))
		Expect(got.Channels).To(Equal([]v4.DOChannel{
			{Key: "ok", Port: 1, Line: 2},
			{Key: "bad", Port: math.MaxUint16, Line: 3, Disabled: true},
		}))
	})
})

type storedPort struct {
	port     uint16
	disabled bool
	fields   []any
}

var _ = Describe("Stored migrations", func() {
	DescribeTable("Should clamp a stored port and log each change",
		func(ctx SpecContext, run func(SpecContext) storedPort) {
			got := run(ctx)
			Expect(got.port).To(Equal(uint16(math.MaxUint16)))
			Expect(got.disabled).To(BeTrue())
			Expect(got.fields).To(ConsistOf("Port", "Disabled"))
		},
		Entry("counter read", func(ctx SpecContext) storedPort {
			got, fields := runStored[v3.CounterReadConfig, v4.CounterReadConfig](
				ctx,
				"ni_counter_read_config",
				v4.CounterReadMigration,
				v3.CounterReadConfig{
					Key: uuid.New(),
					Channels: []v3.CIChannel{{Variant: v3.CIFrequencyChannel{
						BaseCIChannel: v3.BaseCIChannel{Key: "ctr", Port: 70000},
						CustomScale:   noScale,
						Divisor:       4,
					}}},
				},
			)
			base := got.Channels[0].Base()
			return storedPort{base.Port, base.Disabled, fields}
		}),
		Entry("analog write", func(ctx SpecContext) storedPort {
			got, fields := runStored[v3.AnalogWriteConfig, v4.AnalogWriteConfig](
				ctx,
				"ni_analog_write_config",
				v4.AnalogWriteMigration,
				v3.AnalogWriteConfig{
					Key: uuid.New(),
					Channels: []v3.AOChannel{{Variant: v3.AOVoltageChannel{
						BaseAOChannel: v3.BaseAOChannel{Key: "ao", Port: 70000},
						CustomScale:   noScale,
					}}},
				},
			)
			base := got.Channels[0].Base()
			return storedPort{base.Port, base.Disabled, fields}
		}),
		Entry("digital read", func(ctx SpecContext) storedPort {
			got, fields := runStored[v3.DigitalReadConfig, v4.DigitalReadConfig](
				ctx,
				"ni_digital_read_config",
				v4.DigitalReadMigration,
				v3.DigitalReadConfig{
					Key:      uuid.New(),
					Channels: []v3.DIChannel{{Key: "di", Port: 70000, Line: 4}},
				},
			)
			ch := got.Channels[0]
			return storedPort{ch.Port, ch.Disabled, fields}
		}),
		Entry("digital write", func(ctx SpecContext) storedPort {
			got, fields := runStored[v3.DigitalWriteConfig, v4.DigitalWriteConfig](
				ctx,
				"ni_digital_write_config",
				v4.DigitalWriteMigration,
				v3.DigitalWriteConfig{
					Key:      uuid.New(),
					Channels: []v3.DOChannel{{Key: "do", Port: 65536, Line: 3}},
				},
			)
			ch := got.Channels[0]
			return storedPort{ch.Port, ch.Disabled, fields}
		}),
	)
})
