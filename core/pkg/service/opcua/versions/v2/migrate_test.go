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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/opcua/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/opcua/versions/v2"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
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
					{
						Key:      "ch",
						NodeID:   "NS=2;I=8",
						DataType: telem.Int16T,
						Channel:  7,
						IsIndex:  true,
					},
				},
			}
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v1.ReadConfig]{DB: db}),
			)
			Expect(
				gorp.NewCreate[uuid.UUID, v1.ReadConfig]().Entry(&stored).Exec(ctx, db),
			).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "opc_read_config",
				Migrations: []gorp.Migration{v2.ReadMigration},
			})).To(Succeed())
			var got v2.ReadConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v2.ReadConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v2.ReadConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.ReadConfig).To(Equal(stored.ReadConfig))
			Expect(got.Device).To(Equal(stored.Device))
			Expect(got.SampleRate).To(Equal(stored.SampleRate))
			Expect(got.StreamRate).To(Equal(stored.StreamRate))
			Expect(got.ArrayMode).To(BeTrue())
			Expect(got.ArraySize).To(Equal(stored.ArraySize))
			Expect(got.NanFillEnabled).To(BeFalse())
			Expect(got.Channels).To(HaveLen(1))
			Expect(got.Channels[0].BaseChannel).To(Equal(v2.BaseChannel{
				Key:    "ch",
				NodeID: "NS=2;I=8",
			}))
			Expect(got.Channels[0].DataType).To(Equal(telem.Int16T))
			Expect(got.Channels[0].Channel).To(Equal(stored.Channels[0].Channel))
			Expect(got.Channels[0].IsIndex).To(BeTrue())
			Expect(got.Channels[0].Field).To(BeEmpty())
		},
	)

	It(
		"Should keep every write config field and write node values",
		func(ctx SpecContext) {
			stored := v1.WriteConfig{
				Key:       uuid.New(),
				AutoStart: true,
				Device:    "dev",
				Channels: []v1.WriteChannel{
					{
						Key:        "ch",
						NodeID:     "NS=2;I=8",
						DataType:   telem.Int16T,
						CmdChannel: 7,
					},
				},
			}
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(
				gorp.OpenTable(
					ctx,
					gorp.TableConfig[uuid.UUID, v1.WriteConfig]{DB: db},
				),
			)
			Expect(
				gorp.NewCreate[uuid.UUID, v1.WriteConfig]().Entry(&stored).
					Exec(ctx, db),
			).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "opc_write_config",
				Migrations: []gorp.Migration{v2.WriteMigration},
			})).To(Succeed())
			var got v2.WriteConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v2.WriteConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v2.WriteConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.WriteConfig).To(Equal(stored.WriteConfig))
			Expect(got.Channels).To(HaveExactElements(v2.WriteChannel{
				Variant: v2.NodeWriteChannel{
					BaseWriteChannel: v2.BaseWriteChannel{
						BaseChannel: v2.BaseChannel{Key: "ch", NodeID: "NS=2;I=8"},
						CmdChannel:  7,
					},
					DataType: telem.Int16T,
				},
			}))
		},
	)
})
