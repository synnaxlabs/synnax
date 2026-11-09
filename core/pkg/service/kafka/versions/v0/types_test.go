// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	http "github.com/synnaxlabs/synnax/pkg/service/http/versions/v3"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

var _ = Describe("StartOffset", func() {
	DescribeTable("IsValid",
		func(s v0.StartOffset, valid bool) {
			Expect(s.IsValid()).To(Equal(valid))
		},
		Entry("latest", v0.StartOffsetLatest, true),
		Entry("earliest", v0.StartOffsetEarliest, true),
		Entry("none", v0.StartOffsetNone, true),
		Entry("unknown", v0.StartOffset("middle"), false),
		Entry("empty", v0.StartOffset(""), false),
	)
})

var _ = Describe("RecordKey", func() {
	DescribeTable("IsValid",
		func(k v0.RecordKey, valid bool) {
			Expect(k.IsValid()).To(Equal(valid))
		},
		Entry("none", v0.RecordKeyNone, true),
		Entry("channel name", v0.RecordKeyChannelName, true),
		Entry("unknown", v0.RecordKey("task_name"), false),
		Entry("empty", v0.RecordKey(""), false),
	)
})

var _ = Describe("ScanConfig", func() {
	It("Should default the scan rate", func() {
		cfg := v0.ScanConfig{}
		cfg.ApplyDefaults()
		Expect(cfg.Rate).To(BeNumerically(">", 0))
	})
})

var _ = Describe("ReadConfig", func() {
	Describe("ApplyDefaults", func() {
		It("Should default the start offset and field data types", func() {
			cfg := v0.ReadConfig{Records: []v0.ReadRecord{{
				Key:    "r",
				Fields: []v0.ReadField{{Key: "f"}},
			}}}
			cfg.ApplyDefaults()
			Expect(cfg.StartOffset).To(Equal(v0.StartOffsetLatest))
			Expect(cfg.Records[0].Fields[0].DataType).To(Equal(telem.Float64T))
		})

		It("Should keep the values the caller set", func() {
			cfg := v0.ReadConfig{
				StartOffset: v0.StartOffsetNone,
				Records: []v0.ReadRecord{{
					Key:    "r",
					Fields: []v0.ReadField{{Key: "f", DataType: telem.Uint8T}},
				}},
			}
			cfg.ApplyDefaults()
			Expect(cfg.StartOffset).To(Equal(v0.StartOffsetNone))
			Expect(cfg.Records[0].Fields[0].DataType).To(Equal(telem.Uint8T))
		})
	})

	Describe("Validate", func() {
		It("Should accept a defined start offset", func() {
			Expect(v0.ReadConfig{StartOffset: v0.StartOffsetEarliest}.Validate()).
				To(Succeed())
		})

		It("Should reject an unknown start offset", func() {
			Expect(v0.ReadConfig{StartOffset: "middle"}.Validate()).
				To(MatchError(ContainSubstring("invalid start_offset: middle")))
		})
	})
})

var _ = Describe("WriteConfig", func() {
	Describe("ApplyDefaults", func() {
		It("Should default the record key, record shape, and channel JSON types",
			func() {
				cfg := v0.WriteConfig{
					Channels: []v0.WriteChannel{{Key: "c"}},
					Record: v0.Record{
						Fields: []http.WriteField{{Variant: http.StaticWriteField{}}},
					},
				}
				cfg.ApplyDefaults()
				Expect(cfg.RecordKey).To(Equal(v0.RecordKeyChannelName))
				Expect(cfg.Record.ValuePointer).To(Equal("/value"))
				Expect(cfg.Record.TimeFormat).To(Equal(http.TimeFormatUnixNs))
				Expect(cfg.Channels[0].JSONType).To(Equal(http.JSONTypeNumber))
			},
		)

		It("Should keep the values the caller set", func() {
			cfg := v0.WriteConfig{
				RecordKey: v0.RecordKeyNone,
				Record: v0.Record{
					ValuePointer: "/v",
					TimeFormat:   http.TimeFormatISO8601,
				},
				Channels: []v0.WriteChannel{{Key: "c", JSONType: http.JSONTypeString}},
			}
			cfg.ApplyDefaults()
			Expect(cfg.RecordKey).To(Equal(v0.RecordKeyNone))
			Expect(cfg.Record.ValuePointer).To(Equal("/v"))
			Expect(cfg.Record.TimeFormat).To(Equal(http.TimeFormatISO8601))
			Expect(cfg.Channels[0].JSONType).To(Equal(http.JSONTypeString))
		})
	})

	Describe("Validate", func() {
		valid := func() v0.WriteConfig {
			cfg := v0.WriteConfig{Channels: []v0.WriteChannel{{Key: "c"}}}
			cfg.ApplyDefaults()
			return cfg
		}

		It("Should accept a defaulted config", func() {
			Expect(valid().Validate()).To(Succeed())
		})

		DescribeTable("Should reject a field outside its enum",
			func(mutate func(cfg *v0.WriteConfig), msg string) {
				cfg := valid()
				mutate(&cfg)
				Expect(cfg.Validate()).To(MatchError(ContainSubstring(msg)))
			},
			Entry("record key",
				func(cfg *v0.WriteConfig) { cfg.RecordKey = "task_name" },
				"record_key: invalid record_key: task_name"),
			Entry("record time format",
				func(cfg *v0.WriteConfig) { cfg.Record.TimeFormat = "unix_hours" },
				"record.time_format: invalid time_format: unix_hours"),
			Entry("record field",
				func(cfg *v0.WriteConfig) {
					cfg.Record.Fields = []http.WriteField{{
						Variant: http.StaticWriteField{JSONType: "integer"},
					}}
				},
				"record.fields.0.json_type: invalid json_type: integer"),
			Entry("channel json type",
				func(cfg *v0.WriteConfig) { cfg.Channels[0].JSONType = "integer" },
				"channels.0.json_type: invalid json_type: integer"),
		)
	})
})
