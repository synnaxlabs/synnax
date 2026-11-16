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
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v2 "github.com/synnaxlabs/synnax/pkg/service/labjack/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/labjack/versions/v3"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func analog(negChan int32) v2.ReadChannel {
	return v2.ReadChannel{Variant: v2.AnalogReadChannel{
		BaseReadChannel: v2.BaseReadChannel{Key: "ai", Port: "AIN2"},
		Range:           10,
		NegChan:         negChan,
		Scale:           task.Scale{Variant: task.NoneScale{}},
	}}
}

func thermocouple(negChan int32) v2.ReadChannel {
	return v2.ReadChannel{Variant: v2.ThermocoupleReadChannel{
		BaseReadChannel: v2.BaseReadChannel{Key: "tc", Port: "AIN2"},
		NegChan:         negChan,
		Scale:           task.Scale{Variant: task.NoneScale{}},
	}}
}

func negChanOf(ch v3.ReadChannel) *uint16 {
	GinkgoHelper()
	switch v := ch.Variant.(type) {
	case v3.AnalogReadChannel:
		return v.NegChan
	case v3.ThermocoupleReadChannel:
		return v.NegChan
	}
	Fail("channel has no negative channel")
	return nil
}

var _ = Describe("MigrateReadChannel", func() {
	DescribeTable(
		"Should map ground to absent and clamp an out-of-range negative channel",
		func(
			ctx SpecContext,
			old v2.ReadChannel,
			negChan types.GomegaMatcher,
			disabled bool,
		) {
			ch := MustSucceed(v3.MigrateReadChannel(ctx, old))
			Expect(negChanOf(ch)).To(negChan)
			Expect(ch.Base().Disabled).To(Equal(disabled))
		},
		Entry("analog ground", analog(199), BeNil(), false),
		Entry("thermocouple ground", thermocouple(199), BeNil(), false),
		Entry("v2 zero, which v2 filled with ground", analog(0), BeNil(), false),
		Entry("analog pair", analog(3), HaveValue(BeEquivalentTo(3)), false),
		Entry("lowest pair", analog(1), HaveValue(BeEquivalentTo(1)), false),
		Entry("highest port", analog(127), HaveValue(BeEquivalentTo(127)), false),
		Entry("below the lowest port", analog(-1), HaveValue(BeEquivalentTo(0)), true),
		Entry(
			"above the highest port",
			analog(128),
			HaveValue(BeEquivalentTo(127)),
			true,
		),
		Entry(
			"thermocouple above the highest port",
			thermocouple(200),
			HaveValue(BeEquivalentTo(127)),
			true,
		),
	)

	It("Should keep a digital channel", func(ctx SpecContext) {
		base := v2.BaseReadChannel{Key: "di", Port: "DIO4"}
		Expect(v3.MigrateReadChannel(ctx, v2.ReadChannel{
			Variant: v2.DigitalReadChannel{BaseReadChannel: base},
		})).To(Equal(v3.ReadChannel{
			Variant: v3.DigitalReadChannel{BaseReadChannel: base},
		}))
	})
})

var _ = Describe("MigrateReadConfig", func() {
	It("Should map a backlog count of 0 to absent", func(ctx SpecContext) {
		cfg := MustSucceed(v3.MigrateReadConfig(ctx, v2.ReadConfig{}))
		Expect(cfg.DeviceScanBacklogWarnOnCount).To(BeNil())
		Expect(cfg.LjmScanBacklogWarnOnCount).To(BeNil())
	})

	It("Should keep a set backlog count", func(ctx SpecContext) {
		cfg := MustSucceed(v3.MigrateReadConfig(ctx, v2.ReadConfig{
			DeviceScanBacklogWarnOnCount: 7,
			LjmScanBacklogWarnOnCount:    1,
		}))
		Expect(cfg.DeviceScanBacklogWarnOnCount).To(HaveValue(BeEquivalentTo(7)))
		Expect(cfg.LjmScanBacklogWarnOnCount).To(HaveValue(BeEquivalentTo(1)))
	})
})

var _ = Describe("MigrateScanConfig", func() {
	DescribeTable(
		"Should fill a TCP scan multiplier of 0 and clamp a negative one",
		func(ctx SpecContext, old int32, expected uint32) {
			cfg := MustSucceed(v3.MigrateScanConfig(ctx, v2.ScanConfig{
				TCPScanMultiplier: old,
			}))
			Expect(cfg.TCPScanMultiplier).To(Equal(expected))
		},
		Entry("negative", int32(-4), uint32(1)),
		Entry("zero", int32(0), uint32(10)),
		Entry("the minimum", int32(1), uint32(1)),
		Entry("the default", int32(10), uint32(10)),
	)
})

func observe() (alamos.Instrumentation, *observer.ObservedLogs) {
	GinkgoHelper()
	core, logs := observer.New(zapcore.WarnLevel)
	logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
		ZapLogger: zap.New(core),
	}))
	return alamos.New("test", alamos.WithLogger(logger)), logs
}

var _ = Describe("ReadMigration", func() {
	It("Should clamp, disable, and log a stored negative channel", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		stored := v2.ReadConfig{
			Key:      uuid.New(),
			Channels: []v2.ReadChannel{analog(199), analog(300)},
		}
		Expect(gorp.NewCreate[uuid.UUID, v2.ReadConfig]().Entry(&stored).Exec(ctx, db)).
			To(Succeed())
		ins, logs := observe()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: ins,
			DB:              db,
			Namespace:       "labjack_read_config",
			Migrations:      []gorp.Migration{v3.ReadMigration},
		})).To(Succeed())
		var got v3.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v3.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v3.ReadConfig](stored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(negChanOf(got.Channels[0])).To(BeNil())
		Expect(got.Channels[0].Base().Disabled).To(BeFalse())
		Expect(negChanOf(got.Channels[1])).To(HaveValue(BeEquivalentTo(127)))
		Expect(got.Channels[1].Base().Disabled).To(BeTrue())
		clamped := logs.FilterField(zap.String("field", "neg_chan")).All()
		Expect(clamped).To(HaveLen(1))
		Expect(clamped[0].ContextMap()).To(And(
			HaveKeyWithValue("config", stored.Key.String()),
			HaveKeyWithValue("channel", "ai"),
			HaveKeyWithValue("old", BeEquivalentTo(300)),
			HaveKeyWithValue("new", BeEquivalentTo(127)),
		))
	})
})

var _ = Describe("ScanMigration", func() {
	It("Should clamp and log a stored TCP scan multiplier", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		stored := v2.ScanConfig{
			Key:               uuid.New(),
			TCPScanMultiplier: -2,
		}
		Expect(gorp.NewCreate[uuid.UUID, v2.ScanConfig]().Entry(&stored).Exec(ctx, db)).
			To(Succeed())
		ins, logs := observe()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: ins,
			DB:              db,
			Namespace:       "labjack_scan_config",
			Migrations:      []gorp.Migration{v3.ScanMigration},
		})).To(Succeed())
		var got v3.ScanConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v3.ScanConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v3.ScanConfig](stored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.TCPScanMultiplier).To(BeEquivalentTo(1))
		clamped := logs.FilterField(zap.String("field", "tcp_scan_multiplier")).All()
		Expect(clamped).To(HaveLen(1))
		Expect(clamped[0].ContextMap()).To(HaveKeyWithValue("old", BeEquivalentTo(-2)))
	})
})
