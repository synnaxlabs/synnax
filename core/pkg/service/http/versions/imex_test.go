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
		g := cfg.Endpoints[0].Fields[0].Variant.(versions.GeneratedWriteField)
		Expect(g.Generator).To(Equal(payload.GeneratorTypeTimestamp))
		Expect(*g.TimeFormat).To(Equal(payload.TimeFormatUnixNs))
	})
})
