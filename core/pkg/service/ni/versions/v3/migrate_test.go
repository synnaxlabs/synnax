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
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("ScanMigration", func() {
	It("Should keep every scan config field and drop the ignored models",
		func(ctx SpecContext) {
			stored := v2.ScanConfig{
				Key:           uuid.New(),
				Rate:          5,
				Disabled:      true,
				IgnoredModels: []string{"^cRIO.*", "^nown.*"},
			}
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[uuid.UUID, v2.ScanConfig]{DB: db},
			))
			Expect(
				gorp.NewCreate[uuid.UUID, v2.ScanConfig]().Entry(&stored).Exec(ctx, db),
			).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "ni_scanner_config",
				Migrations: []gorp.Migration{v3.ScanMigration},
			})).To(Succeed())
			var got v3.ScanConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v3.ScanConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v3.ScanConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.ScanConfig).To(Equal(stored.ScanConfig))
		},
	)
})

var _ = Describe("MigrateDIChannel", func() {
	DescribeTable("Should clamp the port and line",
		func(
			ctx SpecContext,
			port, line int32,
			wantPort uint32,
			wantLine uint8,
			disabled bool,
		) {
			got := MustSucceed(v3.MigrateDIChannel(
				ctx,
				v2.DIChannel{Key: "a", Port: port, Line: line},
			))
			Expect(got.Port).To(Equal(wantPort))
			Expect(got.Line).To(Equal(wantLine))
			Expect(got.Disabled).To(Equal(disabled))
		},
		Entry("in range", int32(2), int32(31), uint32(2), uint8(31), false),
		Entry("negative port", int32(-1), int32(3), uint32(0), uint8(3), true),
		Entry("negative line", int32(0), int32(-4), uint32(0), uint8(0), true),
		Entry("line above 31", int32(0), int32(40), uint32(0), uint8(31), true),
	)
})

var _ = Describe("MigrateDOChannel", func() {
	It("Should clamp a line above 31 and disable the channel", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateDOChannel(
			ctx,
			v2.DOChannel{Key: "a", Port: 1, Line: 32},
		))
		Expect(got.Port).To(Equal(uint32(1)))
		Expect(got.Line).To(Equal(uint8(31)))
		Expect(got.Disabled).To(BeTrue())
	})
})

var _ = Describe("MigrateBaseAIChannel", func() {
	It("Should clamp a negative port and disable the channel", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateBaseAIChannel(ctx, v2.BaseAIChannel{Port: -3}))
		Expect(got.Port).To(BeZero())
		Expect(got.Disabled).To(BeTrue())
	})

	It("Should keep an in-range port and the channel enabled", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateBaseAIChannel(ctx, v2.BaseAIChannel{Port: 7}))
		Expect(got.Port).To(Equal(uint32(7)))
		Expect(got.Disabled).To(BeFalse())
	})
})

var _ = Describe("MigrateBaseAOChannel", func() {
	It("Should clamp a negative port and disable the channel", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateBaseAOChannel(ctx, v2.BaseAOChannel{Port: -1}))
		Expect(got.Port).To(BeZero())
		Expect(got.Disabled).To(BeTrue())
	})
})

var _ = Describe("MigrateAOChannel", func() {
	oldBase := v2.BaseAOChannel{
		Key:              "ch",
		Port:             3,
		CmdChannel:       11,
		StateChannel:     12,
		CmdChannelName:   "ch_cmd",
		StateChannelName: "ch_state",
	}
	newBase := v3.BaseAOChannel{Key: "ch", Port: 3}
	commanded := v3.CommandedAOChannel{
		CmdChannel:       11,
		StateChannel:     12,
		CmdChannelName:   "ch_cmd",
		StateChannelName: "ch_state",
	}

	It("Should drop the command and state channels of a function generator",
		func(ctx SpecContext) {
			got := MustSucceed(v3.MigrateAOChannel(ctx, v2.AOChannel{
				Variant: v2.AOFuncGenChannel{
					BaseAOChannel: oldBase,
					WaveType:      v2.WaveTypeTriangle,
					Frequency:     50,
					Amplitude:     2,
					Offset:        0.5,
				},
			}))
			Expect(got.Variant).To(Equal(v3.AOFuncGenChannel{
				BaseAOChannel: newBase,
				WaveType:      v3.WaveTypeTriangle,
				Frequency:     50,
				Amplitude:     2,
				Offset:        0.5,
			}))
		},
	)

	It("Should keep the command and state channels of a voltage output",
		func(ctx SpecContext) {
			got := MustSucceed(v3.MigrateAOChannel(ctx, v2.AOChannel{
				Variant: v2.AOVoltageChannel{BaseAOChannel: oldBase},
			}))
			Expect(got.Variant).To(Equal(v3.AOVoltageChannel{
				BaseAOChannel:      newBase,
				CommandedAOChannel: commanded,
			}))
		},
	)

	It("Should keep the command and state channels of a current output",
		func(ctx SpecContext) {
			got := MustSucceed(v3.MigrateAOChannel(ctx, v2.AOChannel{
				Variant: v2.AOCurrentChannel{BaseAOChannel: oldBase},
			}))
			Expect(got.Variant).To(Equal(v3.AOCurrentChannel{
				BaseAOChannel:      newBase,
				CommandedAOChannel: commanded,
			}))
		},
	)

	It("Should clamp a negative port and disable the channel", func(ctx SpecContext) {
		base := oldBase
		base.Port = -1
		got := MustSucceed(v3.MigrateAOChannel(ctx, v2.AOChannel{
			Variant: v2.AOVoltageChannel{BaseAOChannel: base},
		}))
		Expect(got.Base().Port).To(BeZero())
		Expect(got.Base().Disabled).To(BeTrue())
	})

	DescribeTable("Should make a function generator frequency positive",
		func(ctx SpecContext, frequency, want float64) {
			got := MustSucceed(v3.MigrateAOChannel(ctx, v2.AOChannel{
				Variant: v2.AOFuncGenChannel{Frequency: frequency},
			}))
			Expect(got.Variant.(v3.AOFuncGenChannel).Frequency).To(Equal(want))
		},
		Entry("zero", 0.0, 1.0),
		Entry("negative", -20.0, 1.0),
		Entry("NaN", math.NaN(), 1.0),
		Entry("positive", 12.5, 12.5),
	)
})

var _ = Describe("MigrateBaseCIChannel", func() {
	It("Should clamp a negative port and disable the channel", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateBaseCIChannel(ctx, v2.BaseCIChannel{Port: -1}))
		Expect(got.Port).To(BeZero())
		Expect(got.Disabled).To(BeTrue())
	})
})

var _ = Describe("MigrateCJC", func() {
	It("Should clamp a negative channel port", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateCJC(ctx, v2.CJC{Variant: v2.ChanCJC{Port: -2}}))
		Expect(got.Variant).To(Equal(v3.ChanCJC{Port: 0}))
	})

	It("Should keep a constant source", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateCJC(ctx, v2.CJC{Variant: v2.ConstValCJC{Val: 25}}))
		Expect(got.Variant).To(Equal(v3.ConstValCJC{Val: 25}))
	})
})

var _ = Describe("MigrateCIChannel", func() {
	DescribeTable("Should clamp the divisor to at least 1 or default it",
		func(ctx SpecContext, divisor int32, want uint32) {
			got := MustSucceed(v3.MigrateCIChannel(ctx, v2.CIChannel{
				Variant: v2.CIFrequencyChannel{Divisor: divisor},
			}))
			Expect(got.Variant.(v3.CIFrequencyChannel).Divisor).To(Equal(want))
		},
		Entry("absent", int32(0), uint32(4)),
		Entry("negative", int32(-5), uint32(1)),
		Entry("at the maximum", int32(math.MaxInt32), uint32(math.MaxInt32)),
	)

	It("Should clamp the period divisor to at least 1", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateCIChannel(ctx, v2.CIChannel{
			Variant: v2.CIPeriodChannel{Divisor: -1},
		}))
		Expect(got.Variant.(v3.CIPeriodChannel).Divisor).To(Equal(uint32(1)))
	})

	DescribeTable("Should clamp and truncate pulses per revolution",
		func(ctx SpecContext, pulses float64, want uint32) {
			got := MustSucceed(v3.MigrateCIChannel(ctx, v2.CIChannel{
				Variant: v2.CIPositionAngularChannel{PulsesPerRev: pulses},
			}))
			Expect(got.Variant.(v3.CIPositionAngularChannel).PulsesPerRev).
				To(Equal(want))
		},
		Entry("absent", 0.0, uint32(24)),
		Entry("below 1", 0.5, uint32(1)),
		Entry("a fraction", 24.9, uint32(24)),
		Entry("above uint32", 1e12, uint32(math.MaxUint32)),
		Entry("NaN", math.NaN(), uint32(1)),
	)

	It("Should clamp angular velocity pulses per revolution", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateCIChannel(ctx, v2.CIChannel{
			Variant: v2.CIVelocityAngularChannel{PulsesPerRev: -3},
		}))
		Expect(got.Variant.(v3.CIVelocityAngularChannel).PulsesPerRev).
			To(Equal(uint32(1)))
	})

	DescribeTable("Should clamp and truncate the initial count",
		func(ctx SpecContext, count float64, want uint32) {
			got := MustSucceed(v3.MigrateCIChannel(ctx, v2.CIChannel{
				Variant: v2.CIEdgeCountChannel{InitialCount: count},
			}))
			Expect(got.Variant.(v3.CIEdgeCountChannel).InitialCount).To(Equal(want))
		},
		Entry("negative", -5.0, uint32(0)),
		Entry("a fraction", 3.7, uint32(3)),
		Entry("above uint32", 5e10, uint32(math.MaxUint32)),
	)
})

var _ = Describe("MigrateAnalogReadConfig", func() {
	It("Should disable a thermocouple whose cold-junction port clamps",
		func(ctx SpecContext) {
			got := MustSucceed(v3.MigrateAnalogReadConfig(ctx, v2.AnalogReadConfig{
				Channels: []v2.AIChannel{
					{Variant: v2.AIThermocoupleChannel{
						BaseAIChannel: v2.BaseAIChannel{Key: "tc", Port: 1},
						Cjc:           v2.CJC{Variant: v2.ChanCJC{Port: -1}},
					}},
					{Variant: v2.AIVoltageChannel{
						BaseAIChannel: v2.BaseAIChannel{Key: "v", Port: 2},
					}},
				},
			}))
			tc := got.Channels[0].Variant.(v3.AIThermocoupleChannel)
			Expect(tc.Disabled).To(BeTrue())
			Expect(tc.Port).To(Equal(uint32(1)))
			Expect(tc.Cjc.Variant).To(Equal(v3.ChanCJC{Port: 0}))
			v := got.Channels[1].Variant.(v3.AIVoltageChannel)
			Expect(v.Disabled).To(BeFalse())
		},
	)
})

var _ = Describe("DigitalReadMigration", func() {
	It("Should clamp and disable stored channels and log each change",
		func(ctx SpecContext) {
			stored := v2.DigitalReadConfig{
				Key: uuid.New(),
				Channels: []v2.DIChannel{
					{Key: "ok", Port: 0, Line: 3},
					{Key: "bad", Port: 1, Line: 40},
				},
			}
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[uuid.UUID, v2.DigitalReadConfig]{DB: db},
			))
			Expect(gorp.NewCreate[uuid.UUID, v2.DigitalReadConfig]().
				Entry(&stored).Exec(ctx, db)).To(Succeed())
			core, logs := observer.New(zapcore.WarnLevel)
			logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
				ZapLogger: zap.New(core),
			}))
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
				DB:              db,
				Namespace:       "ni_digital_read_config",
				Migrations:      []gorp.Migration{v3.DigitalReadMigration},
			})).To(Succeed())
			var got v3.DigitalReadConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v3.DigitalReadConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v3.DigitalReadConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Channels).To(Equal([]v3.DIChannel{
				{Key: "ok", Port: 0, Line: 3},
				{Key: "bad", Port: 1, Line: 31, Disabled: true},
			}))
			changes := logs.FilterMessage("narrowed an NI channel field").All()
			Expect(changes).To(HaveLen(2))
			line := changes[0].ContextMap()
			Expect(line).To(HaveKeyWithValue("config", stored.Key.String()))
			Expect(line).To(HaveKeyWithValue("channel", "bad"))
			Expect(line).To(HaveKeyWithValue("field", "Line"))
			Expect(line["old"]).To(BeNumerically("==", 40))
			Expect(line["new"]).To(BeNumerically("==", 31))
			Expect(changes[1].ContextMap()).To(HaveKeyWithValue("field", "Disabled"))
		},
	)
})

var _ = Describe("CounterReadMigration", func() {
	It("Should migrate a NaN pulse count and log the change", func(ctx SpecContext) {
		stored := v2.CounterReadConfig{
			Key: uuid.New(),
			Channels: []v2.CIChannel{{Variant: v2.CIPositionAngularChannel{
				BaseCIChannel: v2.BaseCIChannel{Key: "enc", Port: -1},
				CustomScale: v2.CustomScale{
					CustomScale: v2.Scale{Variant: v2.NoneScale{}},
				},
				PulsesPerRev: math.NaN(),
			}}},
		}
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(
			ctx,
			gorp.TableConfig[uuid.UUID, v2.CounterReadConfig]{DB: db},
		))
		Expect(gorp.NewCreate[uuid.UUID, v2.CounterReadConfig]().
			Entry(&stored).Exec(ctx, db)).To(Succeed())
		core, logs := observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			DB:              db,
			Namespace:       "ni_counter_read_config",
			Migrations:      []gorp.Migration{v3.CounterReadMigration},
		})).To(Succeed())
		var got v3.CounterReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v3.CounterReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v3.CounterReadConfig](stored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		ch := got.Channels[0].Variant.(v3.CIPositionAngularChannel)
		Expect(ch.PulsesPerRev).To(Equal(uint32(1)))
		Expect(ch.Port).To(BeZero())
		Expect(ch.Disabled).To(BeTrue())
		fields := []any{}
		for _, e := range logs.FilterMessage("narrowed an NI channel field").All() {
			fields = append(fields, e.ContextMap()["field"])
		}
		Expect(fields).To(ConsistOf("Port", "PulsesPerRev", "Disabled"))
	})

	It("Should log a narrowed divisor and initial count", func(ctx SpecContext) {
		_, fields := runStored[v2.CounterReadConfig, v3.CounterReadConfig](
			ctx,
			"ni_counter_read_config",
			v3.CounterReadMigration,
			v2.CounterReadConfig{Key: uuid.New(), Channels: []v2.CIChannel{
				{Variant: v2.CIFrequencyChannel{
					BaseCIChannel: v2.BaseCIChannel{Key: "freq"},
					CustomScale:   noScale,
					Divisor:       -2,
				}},
				{Variant: v2.CIEdgeCountChannel{
					BaseCIChannel: v2.BaseCIChannel{Key: "edge"},
					InitialCount:  2.5,
				}},
			}},
		)
		Expect(fields).To(Equal([]any{"Divisor", "InitialCount"}))
	})
})

var _ = Describe("AnalogReadMigration", func() {
	It("Should log a clamped cold-junction port", func(ctx SpecContext) {
		stored := v2.AnalogReadConfig{
			Key: uuid.New(),
			Channels: []v2.AIChannel{{Variant: v2.AIThermocoupleChannel{
				BaseAIChannel: v2.BaseAIChannel{Key: "tc", Port: 1},
				Cjc:           v2.CJC{Variant: v2.ChanCJC{Port: -1}},
			}}},
		}
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(
			ctx,
			gorp.TableConfig[uuid.UUID, v2.AnalogReadConfig]{DB: db},
		))
		Expect(gorp.NewCreate[uuid.UUID, v2.AnalogReadConfig]().
			Entry(&stored).Exec(ctx, db)).To(Succeed())
		core, logs := observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			DB:              db,
			Namespace:       "ni_analog_read_config",
			Migrations:      []gorp.Migration{v3.AnalogReadMigration},
		})).To(Succeed())
		fields := []any{}
		for _, e := range logs.FilterMessage("narrowed an NI channel field").All() {
			fields = append(fields, e.ContextMap()["field"])
		}
		Expect(fields).To(ConsistOf("Cjc.Port", "Disabled"))
	})

	It("Should log a clamped port and skip a built-in temperature channel",
		func(ctx SpecContext) {
			_, fields := runStored[v2.AnalogReadConfig, v3.AnalogReadConfig](
				ctx,
				"ni_analog_read_config",
				v3.AnalogReadMigration,
				v2.AnalogReadConfig{Key: uuid.New(), Channels: []v2.AIChannel{
					{Variant: v2.AITempBuiltinChannel{Key: "temp"}},
					{Variant: v2.AIVoltageChannel{
						BaseAIChannel: v2.BaseAIChannel{Key: "volt", Port: -1},
						CustomScale:   noScale,
					}},
				}},
			)
			Expect(fields).To(Equal([]any{"Port", "Disabled"}))
		},
	)
})

// runStored stores old under namespace, runs migration over it, and returns the
// migrated record with the field of each change the migration logged.
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
	fields := []any{}
	for _, e := range logs.FilterMessage("narrowed an NI channel field").All() {
		fields = append(fields, e.ContextMap()["field"])
	}
	return got, fields
}

var noScale = v2.CustomScale{CustomScale: v2.Scale{Variant: v2.NoneScale{}}}

var _ = Describe("MigrateAnalogWriteConfig", func() {
	It("Should clamp a negative port and disable the channel", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateAnalogWriteConfig(ctx, v2.AnalogWriteConfig{
			Channels: []v2.AOChannel{{Variant: v2.AOVoltageChannel{
				BaseAOChannel: v2.BaseAOChannel{Key: "ao", Port: -1},
			}}},
		}))
		ch := got.Channels[0].Variant.(v3.AOVoltageChannel)
		Expect(ch.Port).To(BeZero())
		Expect(ch.Disabled).To(BeTrue())
	})
})

var _ = Describe("MigrateCounterReadConfig", func() {
	It("Should clamp each counter channel", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateCounterReadConfig(ctx, v2.CounterReadConfig{
			Channels: []v2.CIChannel{{Variant: v2.CIFrequencyChannel{Divisor: -2}}},
		}))
		Expect(got.Channels[0].Variant.(v3.CIFrequencyChannel).Divisor).
			To(Equal(uint32(1)))
	})
})

var _ = Describe("MigrateDigitalReadConfig", func() {
	It("Should clamp and disable each digital channel", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateDigitalReadConfig(ctx, v2.DigitalReadConfig{
			Device:   "dev",
			Channels: []v2.DIChannel{{Key: "a", Line: 33}},
		}))
		Expect(string(got.Device)).To(Equal("dev"))
		Expect(got.Channels).To(Equal([]v3.DIChannel{
			{Key: "a", Line: 31, Disabled: true},
		}))
	})
})

var _ = Describe("MigrateDigitalWriteConfig", func() {
	It("Should clamp and disable each digital channel", func(ctx SpecContext) {
		got := MustSucceed(v3.MigrateDigitalWriteConfig(ctx, v2.DigitalWriteConfig{
			Channels: []v2.DOChannel{{Key: "a", Port: -1, Line: 2}},
		}))
		Expect(got.Channels).To(Equal([]v3.DOChannel{
			{Key: "a", Line: 2, Disabled: true},
		}))
	})
})

var _ = Describe("AnalogWriteMigration", func() {
	It("Should clamp a stored output port and log each change", func(ctx SpecContext) {
		got, fields := runStored[v2.AnalogWriteConfig, v3.AnalogWriteConfig](
			ctx,
			"ni_analog_write_config",
			v3.AnalogWriteMigration,
			v2.AnalogWriteConfig{
				Key: uuid.New(),
				Channels: []v2.AOChannel{{Variant: v2.AOVoltageChannel{
					BaseAOChannel: v2.BaseAOChannel{Key: "ao", Port: -4},
					CustomScale:   noScale,
				}}},
			},
		)
		ch := got.Channels[0].Variant.(v3.AOVoltageChannel)
		Expect(ch.Port).To(BeZero())
		Expect(ch.Disabled).To(BeTrue())
		Expect(fields).To(ConsistOf("Port", "Disabled"))
	})

	It("Should make a stored frequency positive and log the change",
		func(ctx SpecContext) {
			got, fields := runStored[v2.AnalogWriteConfig, v3.AnalogWriteConfig](
				ctx,
				"ni_analog_write_config",
				v3.AnalogWriteMigration,
				v2.AnalogWriteConfig{
					Key: uuid.New(),
					Channels: []v2.AOChannel{{Variant: v2.AOFuncGenChannel{
						BaseAOChannel: v2.BaseAOChannel{Key: "gen"},
					}}},
				},
			)
			ch := got.Channels[0].Variant.(v3.AOFuncGenChannel)
			Expect(ch.Frequency).To(Equal(1.0))
			Expect(fields).To(ConsistOf("Frequency"))
		},
	)

	It("Should drop the Synnax channels of a function generator and keep the others",
		func(ctx SpecContext) {
			oldBase := func(key string, port int32) v2.BaseAOChannel {
				return v2.BaseAOChannel{
					Key:              key,
					Port:             port,
					CmdChannel:       channel.Key(port + 10),
					StateChannel:     channel.Key(port + 20),
					CmdChannelName:   key + "_cmd",
					StateChannelName: key + "_state",
				}
			}
			commanded := func(key string, port int32) v3.CommandedAOChannel {
				return v3.CommandedAOChannel{
					CmdChannel:       channel.Key(port + 10),
					StateChannel:     channel.Key(port + 20),
					CmdChannelName:   key + "_cmd",
					StateChannelName: key + "_state",
				}
			}
			bounds := v2.MinMaxVal{MinVal: -5, MaxVal: 5}
			got, fields := runStored[v2.AnalogWriteConfig, v3.AnalogWriteConfig](
				ctx,
				"ni_analog_write_config",
				v3.AnalogWriteMigration,
				v2.AnalogWriteConfig{Key: uuid.New(), Channels: []v2.AOChannel{
					{Variant: v2.AOFuncGenChannel{
						BaseAOChannel: oldBase("gen", 0),
						WaveType:      v2.WaveTypeSquare,
						Frequency:     100,
						Amplitude:     2,
						Offset:        1,
					}},
					{Variant: v2.AOVoltageChannel{
						BaseAOChannel: oldBase("voltage", 1),
						MinMaxVal:     bounds,
						CustomScale:   noScale,
					}},
					{Variant: v2.AOCurrentChannel{
						BaseAOChannel: oldBase("current", 2),
						MinMaxVal:     bounds,
						CustomScale:   noScale,
					}},
				}},
			)
			Expect(got.Channels).To(Equal([]v3.AOChannel{
				{Variant: v3.AOFuncGenChannel{
					BaseAOChannel: v3.BaseAOChannel{Key: "gen", Port: 0},
					WaveType:      v3.WaveTypeSquare,
					Frequency:     100,
					Amplitude:     2,
					Offset:        1,
				}},
				{Variant: v3.AOVoltageChannel{
					BaseAOChannel:      v3.BaseAOChannel{Key: "voltage", Port: 1},
					MinMaxVal:          bounds,
					CustomScale:        noScale,
					CommandedAOChannel: commanded("voltage", 1),
				}},
				{Variant: v3.AOCurrentChannel{
					BaseAOChannel:      v3.BaseAOChannel{Key: "current", Port: 2},
					MinMaxVal:          bounds,
					CustomScale:        noScale,
					CommandedAOChannel: commanded("current", 2),
				}},
			}))
			Expect(fields).To(BeEmpty())
		},
	)
})

var _ = Describe("DigitalWriteMigration", func() {
	It("Should clamp a stored line and log each change", func(ctx SpecContext) {
		got, fields := runStored[v2.DigitalWriteConfig, v3.DigitalWriteConfig](
			ctx,
			"ni_digital_write_config",
			v3.DigitalWriteMigration,
			v2.DigitalWriteConfig{
				Key:      uuid.New(),
				Channels: []v2.DOChannel{{Key: "do", Port: 1, Line: -1}},
			},
		)
		Expect(got.Channels).To(Equal([]v3.DOChannel{
			{Key: "do", Port: 1, Line: 0, Disabled: true},
		}))
		Expect(fields).To(ConsistOf("Line", "Disabled"))
	})
})
