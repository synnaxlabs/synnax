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
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It(
		"Should keep every read config field and leave NaN fill disabled",
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
					{Key: "ch", NodeID: "NS=2;I=8", Channel: 7},
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
			Expect(got.Channels).To(Equal(stored.Channels))
			Expect(got.NanFillEnabled).To(BeFalse())
		},
	)
})
