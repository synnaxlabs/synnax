// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy/v1"
	notation "github.com/synnaxlabs/x/notation/versions/v0"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

var _ = Describe("Normalize", func() {
	DescribeTable("Should coerce per-channel enums to their valid value or the standard default",
		func(in v1.ChannelEntry, notat notation.Notation, format telem.TimestampFormat, tz telem.TimeZone) {
			out := v1.Data{Channels: []v1.ChannelEntry{in}}.Normalize()
			Expect(out.Channels).To(HaveLen(1))
			Expect(out.Channels[0].Notation).To(Equal(notat))
			Expect(out.Channels[0].Timestamp.Format).To(Equal(format))
			Expect(out.Channels[0].Timestamp.Tz).To(Equal(tz))
		},
		Entry("out-of-set values fall back to the standard defaults",
			v1.ChannelEntry{
				Channel:   channel.Key(1),
				Notation:  notation.Notation("unsupported"),
				Timestamp: v1.TimestampConfig{Format: telem.TimestampFormat("calendar"), Tz: telem.TimeZone("MST")},
			},
			notation.NotationStandard, telem.TimestampFormatPreciseDate, telem.TimeZoneLocal),
		Entry("empty values fall back to the standard defaults",
			v1.ChannelEntry{Channel: channel.Key(1)},
			notation.NotationStandard, telem.TimestampFormatPreciseDate, telem.TimeZoneLocal),
		Entry("in-set values are preserved",
			v1.ChannelEntry{
				Channel:   channel.Key(1),
				Notation:  notation.NotationScientific,
				Timestamp: v1.TimestampConfig{Format: telem.TimestampFormatISO, Tz: telem.TimeZoneUTC},
			},
			notation.NotationScientific, telem.TimestampFormatISO, telem.TimeZoneUTC),
	)

	It("Should leave the raw color string untouched", func() {
		out := v1.Data{Channels: []v1.ChannelEntry{
			{Channel: channel.Key(1), Color: "not-a-hex"},
		}}.Normalize()
		Expect(out.Channels[0].Color).To(Equal("not-a-hex"))
	})

	It("Should normalize each channel independently in a multi-channel body", func() {
		out := v1.Data{Channels: []v1.ChannelEntry{
			{Channel: channel.Key(1), Notation: notation.Notation("bad")},
			{Channel: channel.Key(2), Notation: notation.NotationScientific},
		}}.Normalize()
		Expect(out.Channels[0].Notation).To(Equal(notation.NotationStandard))
		Expect(out.Channels[1].Notation).To(Equal(notation.NotationScientific))
	})

	It("Should be a no-op when there are no channels", func() {
		Expect(v1.Data{}.Normalize().Channels).To(BeEmpty())
	})
})
