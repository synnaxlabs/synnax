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
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
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

var _ = Describe("AnalogWriteMigration", func() {
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
			newBase := func(key string, port int32) v3.BaseAOChannel {
				return v3.BaseAOChannel{Key: key, Port: port}
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
			scale := v2.CustomScale{CustomScale: v2.Scale{Variant: v2.NoneScale{}}}
			stored := v2.AnalogWriteConfig{Channels: []v2.AOChannel{
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
					CustomScale:   scale,
				}},
				{Variant: v2.AOCurrentChannel{
					BaseAOChannel: oldBase("current", 2),
					MinMaxVal:     bounds,
					CustomScale:   scale,
				}},
			}}
			stored.Key = uuid.New()
			stored.StateRate = 25
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[uuid.UUID, v2.AnalogWriteConfig]{DB: db},
			))
			Expect(
				gorp.NewCreate[uuid.UUID, v2.AnalogWriteConfig]().
					Entry(&stored).
					Exec(ctx, db),
			).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "ni_analog_write_config",
				Migrations: []gorp.Migration{v3.AnalogWriteMigration},
			})).To(Succeed())
			var got v3.AnalogWriteConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v3.AnalogWriteConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v3.AnalogWriteConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.WriteConfig).To(Equal(stored.WriteConfig))
			Expect(got.Channels).To(Equal([]v3.AOChannel{
				{Variant: v3.AOFuncGenChannel{
					BaseAOChannel: newBase("gen", 0),
					WaveType:      v3.WaveTypeSquare,
					Frequency:     100,
					Amplitude:     2,
					Offset:        1,
				}},
				{Variant: v3.AOVoltageChannel{
					BaseAOChannel:      newBase("voltage", 1),
					MinMaxVal:          bounds,
					CustomScale:        scale,
					CommandedAOChannel: commanded("voltage", 1),
				}},
				{Variant: v3.AOCurrentChannel{
					BaseAOChannel:      newBase("current", 2),
					MinMaxVal:          bounds,
					CustomScale:        scale,
					CommandedAOChannel: commanded("current", 2),
				}},
			}))
		},
	)
})
