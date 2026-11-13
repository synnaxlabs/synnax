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
		Entry("positive", 12.5, 12.5),
	)

	It("Should keep a commanded output", func(ctx SpecContext) {
		got := MustSucceed(v4.MigrateAnalogWriteConfig(ctx, v3.AnalogWriteConfig{
			Channels: []v3.AOChannel{{Variant: v3.AOVoltageChannel{
				BaseAOChannel: v3.BaseAOChannel{Key: "ao", Port: 2},
			}}},
		}))
		Expect(got.Channels[0].Variant).
			To(Equal(v4.AOChannelVariant(v4.AOVoltageChannel{
				Key: "ao", Port: 2,
			})))
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
