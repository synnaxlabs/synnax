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
	http "github.com/synnaxlabs/x/http/versions/v0"
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
		Entry("a header with no name", http.Header{Value: "json"}, "name"),
		Entry("a header name with a colon", http.Header{Name: "Accept:"}, "name"),
		Entry("a header name with a space", http.Header{Name: "X Key"}, "name"),
		Entry(
			"a header value with a line break",
			http.Header{Name: "Accept", Value: "a\r\nb"},
			"value",
		),
		Entry(
			"a header value with a NUL",
			http.Header{Name: "Accept", Value: "a\x00b"},
			"value",
		),
		Entry(
			"a query parameter with no name",
			http.QueryParam{Value: "1"},
			"parameter",
		),
	)

	DescribeTable("Should accept a valid field",
		func(v interface{ Validate() error }) { Expect(v.Validate()).To(Succeed()) },
		Entry("a header", http.Header{Name: "X-API-Key", Value: "a\tb"}),
		Entry("a query parameter", http.QueryParam{Parameter: "units", Value: "si"}),
	)
})
