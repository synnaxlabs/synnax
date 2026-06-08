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
	"reflect"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
)

var _ = Describe("Data", func() {
	Describe("ToMap", func() {
		It("Should produce a key for every json-tagged field on Data", func() {
			m := v1.Data{}.ToMap()
			t := reflect.TypeFor[v1.Data]()
			for field := range t.Fields() {
				tag, _, _ := strings.Cut(field.Tag.Get("json"), ",")
				Expect(m).To(HaveKey(tag), "field %s missing from Data.ToMap", tag)
			}
		})

		It("Should preserve typed values without coercing through JSON", func() {
			d := v1.Data{
				Channels:             []v1.ChannelEntry{{Channel: 7, Color: "red"}},
				RemoteCreated:        true,
				TimestampPrecision:   3,
				ShowChannelNames:     false,
				ShowReceiptTimestamp: true,
			}
			m := d.ToMap()
			Expect(m["channels"]).To(Equal(d.Channels))
			Expect(m["remote_created"]).To(Equal(true))
			Expect(m["timestamp_precision"]).To(Equal(3))
			Expect(m["show_channel_names"]).To(Equal(false))
			Expect(m["show_receipt_timestamp"]).To(Equal(true))
		})
	})

	Describe("Schema", func() {
		It("Should parse a valid v1 payload with all fields populated", func() {
			data := map[string]any{
				"channels": []any{
					map[string]any{
						"channel":   1,
						"color":     "red",
						"notation":  "scientific",
						"precision": 2,
						"alias":     "temp",
					},
				},
				"remote_created":         true,
				"timestamp_precision":    3,
				"show_channel_names":     false,
				"show_receipt_timestamp": true,
			}
			var d v1.Data
			Expect(v1.Schema.Parse(data, &d)).To(Succeed())
			Expect(d.Channels).To(HaveLen(1))
			Expect(d.Channels[0].Channel).To(Equal(1))
			Expect(d.Channels[0].Color).To(Equal("red"))
			Expect(d.Channels[0].Notation).To(Equal("scientific"))
			Expect(d.Channels[0].Precision).To(Equal(2))
			Expect(d.Channels[0].Alias).To(Equal("temp"))
			Expect(d.RemoteCreated).To(BeTrue())
			Expect(d.TimestampPrecision).To(Equal(3))
			Expect(d.ShowChannelNames).To(BeFalse())
			Expect(d.ShowReceiptTimestamp).To(BeTrue())
		})

		It("Should accept channel entries with only the required channel field", func() {
			data := map[string]any{
				"channels": []any{map[string]any{"channel": 5}},
			}
			var d v1.Data
			Expect(v1.Schema.Parse(data, &d)).To(Succeed())
			Expect(d.Channels[0].Channel).To(Equal(5))
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
			var d v1.Data
			Expect(v1.Schema.Parse(data, &d)).To(Succeed())
			Expect(d.Channels[0].Channel).To(Equal(7))
			Expect(d.Channels[0].Precision).To(Equal(4))
			Expect(d.TimestampPrecision).To(Equal(2))
		})

		It("Should reject a missing channels field", func() {
			data := map[string]any{"remote_created": true}
			var d v1.Data
			Expect(v1.Schema.Parse(data, &d)).To(MatchError(ContainSubstring("channels")))
		})

		It("Should reject a channel entry missing the required channel key", func() {
			data := map[string]any{
				"channels": []any{map[string]any{"color": "red"}},
			}
			var d v1.Data
			Expect(v1.Schema.Parse(data, &d)).To(MatchError(ContainSubstring("channel")))
		})

		It("Should reject channels containing a non-object entry", func() {
			data := map[string]any{"channels": []any{1, 2, 3}}
			var d v1.Data
			Expect(v1.Schema.Parse(data, &d)).To(MatchError(ContainSubstring("channels")))
		})

		It("Should reject a non-bool show_channel_names", func() {
			data := map[string]any{
				"channels":           []any{map[string]any{"channel": 1}},
				"show_channel_names": "yes",
			}
			var d v1.Data
			Expect(v1.Schema.Parse(data, &d)).To(
				MatchError(ContainSubstring("show_channel_names")),
			)
		})

		It("Should reject a fractional timestamp_precision", func() {
			data := map[string]any{
				"channels":            []any{map[string]any{"channel": 1}},
				"timestamp_precision": 1.5,
			}
			var d v1.Data
			Expect(v1.Schema.Parse(data, &d)).To(
				MatchError(ContainSubstring("timestamp_precision")),
			)
		})
	})
})
