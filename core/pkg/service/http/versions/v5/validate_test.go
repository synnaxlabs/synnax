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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Validate", func() {
	DescribeTable("Should reject an invalid field",
		func(v interface{ Validate() error }, path string) {
			Expect(v.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(path)),
			))
		},
		Entry("a header with no name", v5.Header{Value: "json"}, "name"),
		Entry("a header name with a colon", v5.Header{Name: "Accept:"}, "name"),
		Entry("a header name with a space", v5.Header{Name: "X Key"}, "name"),
		Entry(
			"a header value with a line break",
			v5.Header{Name: "Accept", Value: "a\r\nb"},
			"value",
		),
		Entry("a query parameter with no name", v5.QueryParam{Value: "1"}, "parameter"),
		Entry(
			"a read pointer with no leading slash",
			v5.ReadField{Pointer: "t"},
			"pointer",
		),
		Entry(
			"a channel pointer with a bad escape",
			v5.ChannelField{Pointer: "/a~2"},
			"pointer",
		),
		Entry("a read path with a query", v5.ReadEndpoint{Path: "/a?b=1"}, "path"),
		Entry("a write path with a fragment", v5.WriteEndpoint{Path: "/a#b"}, "path"),
		Entry(
			"a read rate that is not finite",
			v5.ReadConfig{Rate: telem.Rate(math.Inf(1))},
			"rate",
		),
		Entry("a read rate above 10 kHz", v5.ReadConfig{Rate: 10001}, "rate"),
		Entry(
			"a header value with a NUL",
			v5.Header{Name: "Accept", Value: "a\x00b"},
			"value",
		),
		Entry(
			"a header value with a DEL",
			v5.Header{Name: "Accept", Value: "a\x7fb"},
			"value",
		),
		Entry(
			"a static write field pointer with no leading slash",
			v5.WriteField{Variant: v5.StaticWriteField{
				BaseWriteField: v5.BaseWriteField{Pointer: "value"},
			}},
			"pointer",
		),
		Entry(
			"a generated write field pointer with a bad escape",
			v5.WriteField{Variant: v5.GeneratedWriteField{
				BaseWriteField: v5.BaseWriteField{Pointer: "/id~3"},
			}},
			"pointer",
		),
		Entry(
			"a write endpoint with a header that has no name",
			v5.WriteEndpoint{Headers: []v5.Header{{Value: "1"}}},
			"headers",
		),
		Entry(
			"a write endpoint with a query parameter that has no name",
			v5.WriteEndpoint{QueryParams: []v5.QueryParam{{Value: "1"}}},
			"query_params",
		),
	)

	DescribeTable("Should accept a valid field",
		func(v interface{ Validate() error }) { Expect(v.Validate()).To(Succeed()) },
		Entry("a header", v5.Header{Name: "X-API-Key", Value: "a\tb"}),
		Entry("a query parameter", v5.QueryParam{Parameter: "units", Value: "si"}),
		Entry("an escaped pointer", v5.ReadField{Pointer: "/a~1b/~0c/0"}),
		Entry("a whole body pointer", v5.ReadField{}),
	)
})
