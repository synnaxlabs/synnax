// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package base64_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/encoding/base64"
)

var _ = Describe("base64", func() {
	Describe("MustDecode", func() {
		DescribeTable("Success Cases", func(encoded, decoded string) {
			Expect(base64.MustDecode(encoded)).To(Equal(decoded))
		},
			Entry("Hello, World!", "SGVsbG8sIFdvcmxkIQ==", "Hello, World!"),
			Entry("Empty String", "", ""),
			Entry("Non-ASCII Characters", "SGVsbG8sIOS4lueVjA==", "Hello, 世界"),
		)
		It("should panic if the string is not valid base64", func() {
			Expect(func() { base64.MustDecode("invalid") }).To(Panic())
		})
	})
})
