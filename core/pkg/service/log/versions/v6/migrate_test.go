// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v6_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v5 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v5"
	v6 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v6"
	color "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migrate", func() {
	It("Should store a channel color as a fixed color", func(ctx SpecContext) {
		c := color.Color{R: 1, G: 2, A: 128}
		e := MustSucceed(v6.MigrateChannelEntry(ctx, v5.ChannelEntry{
			Channel: 1,
			Color:   &c,
		}))
		Expect(e.Channel).To(BeEquivalentTo(1))
		Expect(e.Color).To(Equal(&servicecolor.Ref{
			Variant: servicecolor.FixedRef{Value: c},
		}))
	})

	It("Should keep an absent channel color absent", func(ctx SpecContext) {
		e := MustSucceed(v6.MigrateChannelEntry(ctx, v5.ChannelEntry{Channel: 1}))
		Expect(e.Color).To(BeNil())
	})

	It("Should lift logs that v0.62 stored to v6", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v5.Key, v5.Log]{DB: db}))
		group, project := uuid.New(), uuid.New()
		old := v5.Log{
			Key:  uuid.New(),
			Name: "Engine",
			Channels: []v5.ChannelEntry{
				{
					Channel:   1,
					Color:     &color.Color{R: 1, G: 2, B: 3, A: 128},
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
		Expect(gorp.NewCreate[v5.Key, v5.Log]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v6.Key, v6.Log]{
			DB:         db,
			Migrations: []gorp.Migration{v6.Migration},
		}))
		var got v6.Log
		Expect(gorp.NewRetrieve[v6.Key, v6.Log]().
			Where(gorp.MatchKeys[v6.Key, v6.Log](old.Key)).
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
		Expect(colored.Color).To(Equal(&servicecolor.Ref{
			Variant: servicecolor.FixedRef{
				Value: color.Color{R: 1, G: 2, B: 3, A: 128},
			},
		}))
		Expect(colored.Notation).To(BeEquivalentTo("scientific"))
		Expect(colored.Precision).To(HaveValue(BeEquivalentTo(2)))
		Expect(colored.Alias).To(Equal("Pressure"))
		Expect(got.Channels[1].Channel).To(BeEquivalentTo(2))
		Expect(got.Channels[1].Color).To(BeNil())
	})
})
