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
	v4 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v5"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
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
})
