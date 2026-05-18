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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/notation"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Data", func() {
	Describe("Parse", func() {
		It("Should parse a fully populated v1 payload via json.Unmarshal", func() {
			data := map[string]any{
				"channels": []any{
					map[string]any{
						"channel":   1,
						"color":     "#ff0000",
						"notation":  "scientific",
						"precision": 2,
						"alias":     "temp",
						"timestamp": map[string]any{
							"format": "ISO",
							"tz":     "UTC",
						},
					},
				},
				"remote_created":         true,
				"timestamp_precision":    3,
				"show_channel_names":     false,
				"show_receipt_timestamp": true,
			}
			d := MustSucceed(v1.Parse(data))
			Expect(d.Channels).To(HaveLen(1))
			Expect(d.Channels[0].Channel).To(Equal(channel.Key(1)))
			Expect(d.Channels[0].Color).To(Equal(color.MustFromHex("#ff0000")))
			Expect(d.Channels[0].Notation).To(Equal(notation.NotationScientific))
			Expect(d.Channels[0].Precision).To(Equal(int32(2)))
			Expect(d.Channels[0].Alias).To(Equal("temp"))
		})

		It("Should accept channel entries with only the channel field", func() {
			data := map[string]any{
				"channels": []any{map[string]any{"channel": 5}},
			}
			d := MustSucceed(v1.Parse(data))
			Expect(d.Channels[0].Channel).To(Equal(channel.Key(5)))
		})

		It("Should coerce json.Number values throughout the payload", func() {
			data := map[string]any{
				"channels": []any{
					map[string]any{
						"channel":   json.Number("7"),
						"precision": json.Number("4"),
					},
				},
				"timestamp_precision": json.Number("2"),
			}
			d := MustSucceed(v1.Parse(data))
			Expect(d.Channels[0].Channel).To(Equal(channel.Key(7)))
			Expect(d.Channels[0].Precision).To(Equal(int32(4)))
			Expect(d.TimestampPrecision).To(Equal(int32(2)))
		})

		It("Should fail on a malformed color hex", func() {
			data := map[string]any{
				"channels": []any{map[string]any{"channel": 1, "color": "not-a-hex"}},
			}
			Expect(v1.Parse(data)).Error().To(MatchError(ContainSubstring("invalid hex color")))
		})

		It("Should accept any string for a typed enum (Validate enforces closed sets)", func() {
			data := map[string]any{
				"channels": []any{map[string]any{"channel": 1, "notation": "garbage"}},
			}
			d := MustSucceed(v1.Parse(data))
			Expect(d.Channels[0].Notation).To(Equal(notation.Notation("garbage")))
		})
	})

	Describe("ParseLenient", func() {
		It("Should drop an invalid color hex pre-unmarshal", func() {
			data := map[string]any{
				"channels": []any{
					map[string]any{"channel": 1, "color": "not-a-hex"},
				},
			}
			d := MustSucceed(v1.ParseLenient(data))
			Expect(d.Channels).To(HaveLen(1))
			Expect(d.Channels[0].Color).To(Equal(color.Color{}))
		})

		It("Should leave enum strings unchanged for the latest-Log lift to handle", func() {
			data := map[string]any{
				"channels": []any{
					map[string]any{"channel": 1, "notation": "garbage"},
				},
			}
			d := MustSucceed(v1.ParseLenient(data))
			Expect(d.Channels[0].Notation).To(Equal(notation.Notation("garbage")))
		})
	})

	Describe("Validate", func() {
		It("Should accept a Data with no enum violations", func() {
			d := v1.Data{Channels: []v1.ChannelEntry{{
				Channel:  channel.Key(1),
				Notation: notation.NotationScientific,
			}}}
			Expect(v1.Validate(d)).To(Succeed())
		})

		It("Should reject an unknown notation value", func() {
			d := v1.Data{Channels: []v1.ChannelEntry{{
				Channel:  channel.Key(1),
				Notation: notation.Notation("logarithmic"),
			}}}
			Expect(v1.Validate(d)).To(MatchError(ContainSubstring("invalid value \"logarithmic\"")))
		})
	})
})
