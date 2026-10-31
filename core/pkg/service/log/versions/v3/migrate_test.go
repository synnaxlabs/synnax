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
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	DescribeTable("MigrateChannelEntry",
		func(ctx SpecContext, precision int32, expected *int32) {
			old := v2.ChannelEntry{Channel: 1, Precision: precision}
			Expect(MustSucceed(v3.MigrateChannelEntry(ctx, old)).Precision).
				To(Equal(expected))
		},
		Entry("Should drop the auto precision sentinel", int32(-1), nil),
		Entry("Should keep a zero precision", int32(0), new(int32(0))),
		Entry("Should keep a set precision", int32(4), new(int32(4))),
	)

	It("Should lift every channel entry of a log", func(ctx SpecContext) {
		old := v2.Log{
			Name: "log",
			Channels: []v2.ChannelEntry{
				{Channel: 1, Precision: -1},
				{Channel: 2, Precision: 5},
			},
		}
		got := MustSucceed(v3.MigrateLog(ctx, old))
		Expect(got.Name).To(Equal("log"))
		Expect(got.Channels).To(Equal([]v3.ChannelEntry{
			{Channel: 1},
			{Channel: 2, Precision: new(int32(5))},
		}))
	})

	It("Should lift stored logs and keep every other field", func(ctx SpecContext) {
		stored := v2.Log{
			Key:                uuid.New(),
			Name:               "log",
			TimestampPrecision: 2,
			ChannelNamesHidden: true,
			Channels: []v2.ChannelEntry{
				{
					Channel:   7,
					Notation:  notation.NotationScientific,
					Precision: -1,
					Alias:     "pressure",
					Timestamp: v2.TimestampConfig{
						Format: telem.TimestampFormatPreciseDate,
						Tz:     telem.TimeZoneUTC,
					},
				},
				{Channel: 8, Precision: 3},
			},
		}
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v2.Log]{DB: db}))
		Expect(gorp.NewCreate[uuid.UUID, v2.Log]().Entry(&stored).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "log",
			Migrations: []gorp.Migration{v3.Migration},
		})).To(Succeed())
		var got v3.Log
		Expect(gorp.NewRetrieve[uuid.UUID, v3.Log]().
			Where(gorp.MatchKeys[uuid.UUID, v3.Log](stored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Name).To(Equal(stored.Name))
		Expect(got.TimestampPrecision).To(Equal(stored.TimestampPrecision))
		Expect(got.ChannelNamesHidden).To(BeTrue())
		Expect(got.Channels).To(Equal([]v3.ChannelEntry{
			{
				Channel:   7,
				Notation:  notation.NotationScientific,
				Alias:     "pressure",
				Timestamp: stored.Channels[0].Timestamp,
			},
			{Channel: 8, Precision: new(int32(3))},
		}))
	})
})
