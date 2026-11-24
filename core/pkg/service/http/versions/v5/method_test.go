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
)

var _ = Describe("Method", func() {
	DescribeTable("Should accept a supported method",
		func(m v5.Method) { Expect(m.IsValid()).To(BeTrue()) },
		Entry("QUERY", v5.MethodQuery),
		Entry("GET", v5.MethodGet),
		Entry("PATCH", v5.MethodPatch),
	)

	DescribeTable("Should refuse a removed method",
		func(m v5.Method) { Expect(m.IsValid()).To(BeFalse()) },
		Entry("TRACE", v5.Method("TRACE")),
		Entry("CONNECT", v5.Method("CONNECT")),
	)
})
