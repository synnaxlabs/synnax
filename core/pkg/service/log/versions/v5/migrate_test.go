// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v5"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migrate", func() {
	DescribeTable("Should store the channel color alpha as a byte",
		func(ctx SpecContext, a float64, expected uint8) {
			old := v4.ChannelEntry{Channel: 1, Color: color.Color{R: 1, G: 2, A: a}}
			e := MustSucceed(v5.MigrateChannelEntry(ctx, old))
			Expect(e.Channel).To(BeEquivalentTo(1))
			Expect(e.Color).To(HaveValue(Equal(colorv2.Color{R: 1, G: 2, A: expected})))
		},
		Entry("a transparent alpha", 0.0, uint8(0)),
		Entry("a faint alpha", 0.003, uint8(1)),
		Entry("a half alpha", 0.5, uint8(128)),
		Entry("an opaque alpha", 1.0, uint8(255)),
	)

	It("Should store a zero channel color as no color", func(ctx SpecContext) {
		e := MustSucceed(v5.MigrateChannelEntry(ctx, v4.ChannelEntry{Channel: 1}))
		Expect(e.Channel).To(BeEquivalentTo(1))
		Expect(e.Color).To(BeNil())
	})

	It("Should convert the color of every channel of a log", func(ctx SpecContext) {
		key := uuid.New()
		group := uuid.New()
		old := v4.Log{
			Key:  key,
			Name: "log",
			Channels: []v4.ChannelEntry{
				{Channel: 1, Color: color.Color{R: 1, A: 0.5}},
				{Channel: 2, Color: color.Color{B: 2, A: 1}},
			},
			Group: &group,
		}
		l := MustSucceed(v5.MigrateLog(ctx, old))
		Expect(l.Key).To(Equal(key))
		Expect(l.Name).To(Equal("log"))
		Expect(l.Group).To(HaveValue(Equal(group)))
		Expect(l.Channels).To(HaveLen(2))
		Expect(l.Channels[0].Color).To(HaveValue(Equal(colorv2.Color{R: 1, A: 128})))
		Expect(l.Channels[1].Color).To(HaveValue(Equal(colorv2.Color{B: 2, A: 255})))
	})

	// v0.61 stored logs as v3, so the stored rows pass through v4 on the way to v5.
	It("Should lift logs that v0.61 stored to v5", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v3.Key, v3.Log]{DB: db}))
		group, project := uuid.New(), uuid.New()
		old := v3.Log{
			Key:  uuid.New(),
			Name: "Engine",
			Channels: []v3.ChannelEntry{
				{
					Channel:   1,
					Color:     color.Color{R: 1, G: 2, B: 3, A: 0.5},
					Notation:  "scientific",
					Precision: new(uint8(2)),
					Alias:     "Pressure",
				},
				{Channel: 2},
			},
			TimestampPrecision: 3,
			ChannelNamesHidden: true,
			TimeZone:           "UTC",
			Group:              &group,
			Project:            &project,
		}
		Expect(gorp.NewCreate[v3.Key, v3.Log]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v5.Key, v5.Log]{
			DB: db,
			Migrations: []gorp.Migration{
				v4.RangeMigration,
				v4.Migration,
				v5.Migration,
			},
		}))
		var got v5.Log
		Expect(gorp.NewRetrieve[v5.Key, v5.Log]().
			Where(gorp.MatchKeys[v5.Key, v5.Log](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Name).To(Equal("Engine"))
		Expect(got.TimestampPrecision).To(BeEquivalentTo(3))
		Expect(got.ChannelNamesHidden).To(BeTrue())
		Expect(got.TimeZone).To(BeEquivalentTo("UTC"))
		Expect(got.Group).To(HaveValue(Equal(group)))
		Expect(got.Project).To(HaveValue(Equal(project)))
		Expect(got.Channels).To(HaveLen(2))
		colored := got.Channels[0]
		Expect(colored.Channel).To(BeEquivalentTo(1))
		Expect(colored.Color).
			To(HaveValue(Equal(colorv2.Color{R: 1, G: 2, B: 3, A: 128})))
		Expect(colored.Notation).To(BeEquivalentTo("scientific"))
		Expect(colored.Precision).To(HaveValue(BeEquivalentTo(2)))
		Expect(colored.Alias).To(Equal("Pressure"))
		Expect(got.Channels[1].Channel).To(BeEquivalentTo(2))
		Expect(got.Channels[1].Color).To(BeNil())
	})
})
