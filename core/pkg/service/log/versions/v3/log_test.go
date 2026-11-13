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
	"io"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/orc"
	notation "github.com/synnaxlabs/x/notation/versions/v0"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

var _ = Describe("Log", func() {
	Describe("GorpKey", func() {
		It("Should return the log's key", func() {
			k := uuid.New()
			Expect(v3.Log{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("ResourceID", func() {
		It("Should return the log resource identifier", func() {
			k := uuid.New()
			Expect(v3.Log{Key: k}.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeLog, Key: k.String(),
			}))
		})
	})

	Describe("ApplyDefaults", func() {
		It("Should fill schema-declared defaults on every channel", func() {
			l := v3.Log{Channels: []v3.ChannelEntry{{}}}
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
			l := v3.Log{Channels: []v3.ChannelEntry{{
				Notation:        "scientific",
				TimestampConfig: v3.TimestampConfig{Format: "ISO", TimeZone: "UTC"},
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
			l := v3.Log{Name: "valid", Channels: []v3.ChannelEntry{{}}}
			l.ApplyDefaults()
			Expect(l.Validate()).To(MatchError(ContainSubstring("channels.0.channel")))
		})

		It("Should accept a defaulted log with an absent precision", func() {
			l := v3.Log{Name: "valid", Channels: []v3.ChannelEntry{{Channel: 1}}}
			l.ApplyDefaults()
			Expect(l.Validate()).To(Succeed())
		})

		DescribeTable("Should accept a precision in range",
			func(precision uint8) {
				l := v3.Log{
					Name: "valid",
					Channels: []v3.ChannelEntry{
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
				l := v3.Log{
					Name: "valid",
					Channels: []v3.ChannelEntry{
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
			l := v3.Log{
				Name: "valid",
				Channels: []v3.ChannelEntry{
					{Channel: 1, Alias: strings.Repeat("a", 64)},
				},
			}
			l.ApplyDefaults()
			Expect(l.Validate()).To(Succeed())
		})

		It("Should reject an alias past the maximum length", func() {
			l := v3.Log{
				Name: "valid",
				Channels: []v3.ChannelEntry{
					{Channel: 1, Alias: strings.Repeat("a", 65)},
				},
			}
			l.ApplyDefaults()
			Expect(l.Validate()).To(MatchError(ContainSubstring("channels.0.alias")))
		})

		It("Should path an invalid notation to its channel", func() {
			l := v3.Log{
				Name:     "valid",
				Channels: []v3.ChannelEntry{{Channel: 1, Notation: "bogus"}},
			}
			l.ApplyDefaults()
			Expect(l.Validate()).To(MatchError(ContainSubstring("channels.0.notation")))
		})
	})

	Describe("DecodeOrc", func() {
		encode := func() []byte {
			GinkgoHelper()
			w := orc.NewWriter(0)
			Expect(v3.Log{
				Key:  uuid.New(),
				Name: "log",
				Channels: []v3.ChannelEntry{{
					Channel:         channel.Key(1),
					Notation:        "standard",
					Precision:       new(uint8(3)),
					Alias:           "alias",
					TimestampConfig: v3.TimestampConfig{Format: "ISO", TimeZone: "UTC"},
				}},
				TimestampPrecision: 2,
			}.EncodeOrc(w)).To(Succeed())
			return w.Bytes()
		}

		It("Should encode the truncation fixture in 90 bytes", func() {
			Expect(encode()).To(HaveLen(90))
		})

		DescribeTable("Should reject a log truncated inside a field",
			func(kept int) {
				r := orc.NewReader(nil)
				r.ResetBytes(encode()[:kept])
				var decoded v3.Log
				Expect(decoded.DecodeOrc(r)).To(Or(
					MatchError(io.EOF),
					MatchError(io.ErrUnexpectedEOF),
				))
			},
			Entry("key", 8),
			Entry("name", 16),
			Entry("channels presence", 23),
			Entry("channel count", 24),
			Entry("channel key", 28),
			Entry("channel color", 32),
			Entry("channel notation", 43),
			Entry("channel precision presence", 55),
			Entry("channel precision", 56),
			Entry("channel alias", 57),
			Entry("channel timestamp", 66),
			Entry("timestamp precision", 80),
			Entry("channel names hidden", 81),
			Entry("timestamps hidden", 82),
			Entry("time zone", 85),
			Entry("group presence", 87),
			Entry("project presence", 88),
			Entry("range presence", 89),
		)
	})
})
