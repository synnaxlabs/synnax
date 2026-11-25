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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	http "github.com/synnaxlabs/x/http/versions/v0"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Endpoint", func() {
	DescribeTable("Should report a bad header or query parameter at its path",
		func(v interface{ Validate() error }, path string) {
			Expect(v.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(path)),
			))
		},
		Entry(
			"a read endpoint header with no name",
			v5.ReadEndpoint{Headers: []http.Header{{Value: "1"}}},
			"headers.0.name",
		),
		Entry(
			"a read endpoint query parameter with no name",
			v5.ReadEndpoint{QueryParams: []http.QueryParam{{Value: "1"}}},
			"query_params.0.parameter",
		),
		Entry(
			"a write endpoint header value with a line break",
			v5.WriteEndpoint{Headers: []http.Header{{Name: "A", Value: "a\nb"}}},
			"headers.0.value",
		),
		Entry(
			"a write endpoint query parameter with no name",
			v5.WriteEndpoint{QueryParams: []http.QueryParam{{Value: "1"}}},
			"query_params.0.parameter",
		),
	)
})
