// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("SanitizePathSegment check", func() {
	DescribeTable("Should return a legal name that a second sanitize keeps",
		func(r rune) {
			out := validate.SanitizePathSegment("a" + strings.Repeat(string(r), 31))
			Expect(validate.IsPathSegment(out)).To(BeTrue())
			Expect(validate.SanitizePathSegment(out)).To(Equal(out))
		},
		Entry("a combining acute accent", '\u0301'),
		Entry("a Hangul vowel jamo", '\u1161'),
		Entry("a Hangul final jamo", '\u11A8'),
		Entry("an Oriya vowel sign", '\u0B3E'),
		Entry("a Kannada vowel sign", '\u0CC2'),
		Entry("a Kannada length mark", '\u0CD5'),
		Entry("a Myanmar vowel sign", '\u102E'),
	)
})
