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
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/payload"
	payloadv0 "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/telem"
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
						map[string]any{
							"type":      "static",
							"key":       "s",
							"pointer":   "/s",
							"json_type": "number",
							"value":     5,
						},
					},
				}},
			},
		))
		Expect(cfg.Endpoints).To(HaveLen(1))
		Expect(cfg.Endpoints[0].Channel.Variant).To(Equal(v5.StringChannelField{
			BaseChannelField: v5.BaseChannelField{DataType: new(telem.Float64T)},
			EnumValues:       []payloadv0.StringEntry{{Label: "ON", Value: "1"}},
		}))
		Expect(cfg.Endpoints[0].Fields).To(Equal([]payload.WriteField{
			{Variant: payload.TimestampWriteField{
				BaseWriteField: payload.BaseWriteField{Key: "g"},
				TimeFormat:     payload.TimeFormatUnixNs,
			}},
			{Variant: payload.IntegerWriteField{
				BaseWriteField: payload.BaseWriteField{Key: "s", Pointer: "/s"},
				Value:          5,
			}},
		}))
	})
})
