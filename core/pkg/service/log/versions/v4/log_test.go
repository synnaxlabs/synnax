// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v4 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	notation "github.com/synnaxlabs/x/notation/versions/v0"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

var _ = Describe("Log", func() {
	Describe("GorpKey", func() {
		It("Should return the log's key", func() {
			k := uuid.New()
			Expect(v4.Log{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("ResourceID", func() {
		It("Should return the log resource identifier", func() {
			k := uuid.New()
			Expect(v4.Log{Key: k}.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeLog, Key: k.String(),
			}))
		})
	})

	Describe("ApplyDefaults", func() {
		It("Should fill schema-declared defaults on every channel", func() {
			l := v4.Log{Channels: []v4.ChannelEntry{{}}}
			l.ApplyDefaults()
			Expect(l.Channels[0].Notation).To(Equal(notation.Notation("standard")))
			Expect(l.Channels[0].TimestampConfig.Format).To(
				Equal(telem.TimestampFormat("preciseDate")),
			)
			Expect(
				l.Channels[0].TimestampConfig.TimeZone,
			).To(Equal(telem.TimeZone("local")))
			Expect(l.Channels[0].Precision).To(BeNil())
		})

		It("Should keep values that are already set", func() {
			l := v4.Log{Channels: []v4.ChannelEntry{{
				Notation:        "scientific",
				TimestampConfig: v4.TimestampConfig{Format: "ISO", TimeZone: "UTC"},
			}}}
			l.ApplyDefaults()
			Expect(l.Channels[0].Notation).To(Equal(notation.Notation("scientific")))
			Expect(
				l.Channels[0].TimestampConfig.Format,
			).To(Equal(telem.TimestampFormat("ISO")))
			Expect(
				l.Channels[0].TimestampConfig.TimeZone,
			).To(Equal(telem.TimeZone("UTC")))
		})
	})

	Describe("Validate", func() {
		It("Should reject a channel of 0", func() {
			l := v4.Log{Name: "valid", Channels: []v4.ChannelEntry{{}}}
			l.ApplyDefaults()
			Expect(l.Validate()).To(MatchError(ContainSubstring("channels.0.channel")))
		})

		It("Should accept a defaulted log with an absent precision", func() {
			l := v4.Log{Name: "valid", Channels: []v4.ChannelEntry{{Channel: 1}}}
			l.ApplyDefaults()
			Expect(l.Validate()).To(Succeed())
		})

		DescribeTable("Should accept a precision in range",
			func(precision uint8) {
				l := v4.Log{
					Name: "valid",
					Channels: []v4.ChannelEntry{
						{Channel: 1, Precision: new(precision)},
					},
				}
				l.ApplyDefaults()
				Expect(l.Validate()).To(Succeed())
			},
			Entry("lower bound", uint8(0)),
			Entry("upper bound", uint8(17)),
		)

		DescribeTable("Should reject a precision out of range",
			func(precision uint8) {
				l := v4.Log{
					Name: "valid",
					Channels: []v4.ChannelEntry{
						{Channel: 1, Precision: new(precision)},
					},
				}
				l.ApplyDefaults()
				Expect(
					l.Validate(),
				).To(MatchError(ContainSubstring("channels.0.precision")))
			},
			Entry("above the maximum", uint8(18)),
		)

		It("Should accept an alias at the maximum length", func() {
			l := v4.Log{
				Name: "valid",
				Channels: []v4.ChannelEntry{
					{Channel: 1, Alias: strings.Repeat("a", 64)},
				},
			}
			l.ApplyDefaults()
			Expect(l.Validate()).To(Succeed())
		})

		It("Should reject an alias past the maximum length", func() {
			l := v4.Log{
				Name: "valid",
				Channels: []v4.ChannelEntry{
					{Channel: 1, Alias: strings.Repeat("a", 65)},
				},
			}
			l.ApplyDefaults()
			Expect(l.Validate()).To(MatchError(ContainSubstring("channels.0.alias")))
		})

		It("Should reject an empty name and an invalid timestamp precision", func() {
			l := v4.Log{TimestampPrecision: 4}
			l.ApplyDefaults()
			Expect(l.Validate()).To(SatisfyAll(
				MatchError(ContainSubstring("name")),
				MatchError(ContainSubstring("timestamp_precision")),
			))
		})

		It("Should path an invalid notation to its channel", func() {
			l := v4.Log{
				Name:     "valid",
				Channels: []v4.ChannelEntry{{Channel: 1, Notation: "bogus"}},
			}
			l.ApplyDefaults()
			Expect(l.Validate()).To(MatchError(ContainSubstring("channels.0.notation")))
		})
	})
})
