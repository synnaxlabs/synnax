// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/http/versions"
	v3 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/payload"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("WriteConfigLadder", func() {
	It("Should keep the payload fields of a v3 body", func(ctx SpecContext) {
		cfg := MustSucceed(versions.WriteConfigLadder.Decode(
			ctx,
			v3.Version,
			msgpack.EncodedJSON{
				"device": "dev",
				"endpoints": []any{map[string]any{
					"key": "ep",
					"channel": map[string]any{
						"json_type":   "string",
						"time_format": "unix_ms",
						"enum_values": []any{
							map[string]any{"label": "ON", "value": 1},
						},
					},
					"fields": []any{
						map[string]any{
							"type":        "generated",
							"key":         "g",
							"generator":   "timestamp",
							"time_format": "unix_ns",
						},
					},
				}},
			},
		))
		Expect(cfg.Endpoints).To(HaveLen(1))
		ch := cfg.Endpoints[0].Channel
		Expect(ch.JSONType).To(Equal(payload.JSONTypeString))
		Expect(*ch.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
		Expect(ch.EnumValues).To(Equal([]payload.EnumEntry{{Label: "ON", Value: 1}}))
		Expect(cfg.Endpoints[0].Fields).To(HaveLen(1))
		g, ok := cfg.Endpoints[0].Fields[0].Variant.(versions.GeneratedWriteField)
		Expect(ok).To(BeTrue())
		Expect(g.Generator).To(Equal(payload.GeneratorTypeTimestamp))
		Expect(*g.TimeFormat).To(Equal(payload.TimeFormatUnixNs))
	})
})

var _ = Describe("ReadConfigLadder", func() {
	It("Should stage the fields of a v4 body and apply the NaN fill rule",
		func(ctx SpecContext) {
			cfg := MustSucceed(versions.ReadConfigLadder.Decode(
				ctx,
				v4.Version,
				msgpack.EncodedJSON{
					"device": "dev",
					"rate":   2,
					"endpoints": []any{map[string]any{
						"key":   "ep",
						"path":  "/values",
						"index": "t",
						"fields": []any{
							map[string]any{"key": "t", "channel": 6, "pointer": "/t"},
							map[string]any{
								"key":      "p",
								"channel":  7,
								"pointer":  "/p",
								"optional": true,
							},
						},
					}},
				},
			))
			Expect(cfg.NanFillEnabled).To(BeTrue())
			Expect(cfg.Endpoints).To(HaveLen(1))
			Expect(cfg.Endpoints[0].Path).To(Equal("/values"))
			Expect(cfg.Endpoints[0].ICD).To(BeNil())
			Expect(cfg.Inline).ToNot(BeNil())
			Expect(cfg.Inline.Endpoints).To(HaveLen(1))
			staged := cfg.Inline.Endpoints[0]
			Expect(staged.Key).To(Equal("ep"))
			Expect(staged.Index).To(Equal("t"))
			Expect(staged.Fields).To(HaveLen(2))
			Expect(staged.Fields[1].Optional).To(BeTrue())
		},
	)

	It("Should decode a v5 body with no inline fields", func(ctx SpecContext) {
		cfg := MustSucceed(versions.ReadConfigLadder.Decode(
			ctx,
			v5.Version,
			msgpack.EncodedJSON{"device": "dev", "rate": 2, "endpoints": []any{}},
		))
		Expect(cfg.Device).To(BeEquivalentTo("dev"))
		Expect(cfg.Inline).To(BeNil())
	})
})
