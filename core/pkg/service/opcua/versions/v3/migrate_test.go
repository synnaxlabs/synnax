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
	v2 "github.com/synnaxlabs/synnax/pkg/service/opcua/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/opcua/versions/v3"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	It(
		"Should keep every read config field and read whole node values",
		func(ctx SpecContext) {
			stored := v2.ReadConfig{
				Key:        uuid.New(),
				AutoStart:  true,
				Device:     "dev",
				SampleRate: 100,
				StreamRate: 20,
				ArrayMode:  true,
				ArraySize:  5,
				Channels: []v2.ReadChannel{
					{Key: "ch", NodeID: "NS=2;I=8", Channel: 7, IsIndex: true},
				},
				NanFillEnabled: true,
			}
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v2.ReadConfig]{DB: db}),
			)
			Expect(
				gorp.NewCreate[uuid.UUID, v2.ReadConfig]().Entry(&stored).Exec(ctx, db),
			).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "opc_read_config",
				Migrations: []gorp.Migration{v3.ReadMigration},
			})).To(Succeed())
			var got v3.ReadConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v3.ReadConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v3.ReadConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.ReadConfig).To(Equal(stored.ReadConfig))
			Expect(got.Device).To(Equal(stored.Device))
			Expect(got.SampleRate).To(Equal(stored.SampleRate))
			Expect(got.StreamRate).To(Equal(stored.StreamRate))
			Expect(got.ArrayMode).To(BeTrue())
			Expect(got.ArraySize).To(Equal(stored.ArraySize))
			Expect(got.NanFillEnabled).To(BeTrue())
			Expect(got.Channels).To(HaveLen(1))
			Expect(got.Channels[0].BaseChannel).To(Equal(stored.Channels[0].BaseChannel))
			Expect(got.Channels[0].Channel).To(Equal(stored.Channels[0].Channel))
			Expect(got.Channels[0].IsIndex).To(BeTrue())
			Expect(got.Channels[0].Field).To(BeEmpty())
		},
	)

	It(
		"Should keep every write config field and write node values",
		func(ctx SpecContext) {
			stored := v2.WriteConfig{
				Key:    uuid.New(),
				Device: "dev",
				Channels: []v2.WriteChannel{
					{Key: "ch", NodeID: "NS=2;I=8", CmdChannel: 7},
				},
			}
			stored.AutoStart = true
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v2.WriteConfig]{DB: db}),
			)
			Expect(
				gorp.NewCreate[uuid.UUID, v2.WriteConfig]().Entry(&stored).Exec(ctx, db),
			).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "opc_write_config",
				Migrations: []gorp.Migration{v3.WriteMigration},
			})).To(Succeed())
			var got v3.WriteConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v3.WriteConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v3.WriteConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.WriteConfig).To(Equal(stored.WriteConfig))
			Expect(got.Channels).To(HaveLen(1))
			Expect(got.Channels[0].BaseChannel).To(Equal(stored.Channels[0].BaseChannel))
			Expect(got.Channels[0].CmdChannel).To(Equal(stored.Channels[0].CmdChannel))
			Expect(got.Channels[0].Type).To(Equal(v3.WriteTypeNode))
			Expect(got.Channels[0].ObjectID).To(BeEmpty())
			Expect(got.Channels[0].Arguments).To(BeEmpty())
		},
	)
})
