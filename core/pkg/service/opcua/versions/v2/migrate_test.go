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
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/opcua/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/opcua/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const clampMessage = "clamped an OPC UA array size"

var _ = Describe("Migration", func() {
	migrate := func(
		ctx SpecContext,
		stored v1.ReadConfig,
	) (v2.ReadConfig, *observer.ObservedLogs) {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(
			gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v1.ReadConfig]{DB: db}),
		)
		Expect(
			gorp.NewCreate[uuid.UUID, v1.ReadConfig]().Entry(&stored).Exec(ctx, db),
		).To(Succeed())
		core, logs := observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			DB:              db,
			Namespace:       "opc_read_config",
			Migrations:      []gorp.Migration{v2.ReadMigration},
		})).To(Succeed())
		var got v2.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v2.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v2.ReadConfig](stored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got, logs
	}

	It(
		"Should keep every read config field, read whole node values, and leave NaN fill disabled",
		func(ctx SpecContext) {
			stored := v1.ReadConfig{
				Key:        uuid.MustParse("1a2b3c4d-0000-0000-0000-000000000000"),
				AutoStart:  true,
				Device:     "dev",
				SampleRate: 100,
				StreamRate: 20,
				ArrayMode:  true,
				ArraySize:  5,
				Channels: []v1.ReadChannel{
					{Key: "ch", NodeID: "NS=2;I=8", Channel: 7, IsIndex: true},
				},
			}
			got, logs := migrate(ctx, stored)
			Expect(got.ReadConfig).To(Equal(stored.ReadConfig))
			Expect(got.Device).To(Equal(stored.Device))
			Expect(got.SampleRate).To(Equal(stored.SampleRate))
			Expect(got.StreamRate).To(Equal(stored.StreamRate))
			Expect(got.Sampling.Variant).To(Equal(v2.ArraySampling{Size: 5}))
			Expect(got.NanFillEnabled).To(BeFalse())
			Expect(got.Channels).To(HaveLen(1))
			ch, want := got.Channels[0], stored.Channels[0]
			Expect(ch.BaseChannel).To(Equal(want.BaseChannel))
			Expect(ch.Channel).To(Equal(want.Channel))
			Expect(ch.IsIndex).To(BeTrue())
			Expect(ch.Field).To(BeEmpty())
			Expect(logs.FilterMessage(clampMessage).Len()).To(BeZero())
		},
	)

	It(
		"Should lift a config out of array mode to unary sampling",
		func(ctx SpecContext) {
			got, logs := migrate(ctx, v1.ReadConfig{
				Key:       uuid.New(),
				ArrayMode: false,
				ArraySize: -3,
			})
			Expect(got.Sampling.Variant).To(Equal(v2.UnarySampling{}))
			Expect(logs.FilterMessage(clampMessage).Len()).To(BeZero())
		},
	)

	DescribeTable(
		"Should clamp an array size into range and log each clamp",
		func(ctx SpecContext, size int32, want uint32) {
			stored := v1.ReadConfig{Key: uuid.New(), ArrayMode: true, ArraySize: size}
			got, logs := migrate(ctx, stored)
			Expect(got.Sampling.Variant).To(Equal(v2.ArraySampling{Size: want}))
			clamps := logs.FilterMessage(clampMessage).All()
			Expect(clamps).To(HaveLen(1))
			Expect(clamps[0].ContextMap()).To(And(
				HaveKeyWithValue("config", stored.Key.String()),
				HaveKeyWithValue("field", "sampling.array.size"),
				HaveKeyWithValue("old", size),
				HaveKeyWithValue("new", want),
			))
		},
		Entry("zero", int32(0), uint32(1)),
		Entry("the lowest stored size", int32(math.MinInt32), uint32(1)),
	)

	It("Should keep the highest stored array size", func(ctx SpecContext) {
		got, logs := migrate(ctx, v1.ReadConfig{
			Key:       uuid.New(),
			ArrayMode: true,
			ArraySize: math.MaxInt32,
		})
		Expect(got.Sampling.Variant).To(Equal(v2.ArraySampling{Size: math.MaxInt32}))
		Expect(logs.FilterMessage(clampMessage).Len()).To(BeZero())
	})

	It("Should clamp an imported array size without a logger", func(ctx SpecContext) {
		got := MustSucceed(v2.MigrateReadConfig(ctx, v1.ReadConfig{
			ArrayMode: true,
			ArraySize: -1,
		}))
		Expect(got.Sampling.Variant).To(Equal(v2.ArraySampling{Size: 1}))
	})
})
