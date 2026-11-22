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

var _ = Describe("SanitizePathSegment breaker", func() {
	DescribeTable("Should return a name that passes IsPathSegment",
		func(s string) {
			out := validate.SanitizePathSegment(s)
			Expect(out).ToNot(BeEmpty())
			Expect(validate.IsPathSegment(out)).To(BeTrue(), "sanitized %+q", out)
		},
		Entry(
			"30 combining marks and then a Hangul vowel",
			"a"+strings.Repeat("\u0316", 30)+"\u1161",
		),
		Entry(
			"31 combining marks and then a Hangul vowel",
			"a"+strings.Repeat("\u0316", 31)+"\u1161",
		),
		Entry(
			"30 combining marks and then a Hangul final consonant",
			"a"+strings.Repeat("\u0316", 30)+"\u11A8",
		),
		Entry(
			"30 combining marks and then a Bengali vowel sign",
			"\u0995"+strings.Repeat("\u0316", 30)+"\u09BE",
		),
		Entry(
			"30 combining marks and then a Tamil vowel sign",
			"\u0B95"+strings.Repeat("\u0316", 30)+"\u0BBE",
		),
		Entry(
			"30 combining marks and then a Hangul compatibility vowel",
			"a"+strings.Repeat("\u0316", 30)+"\u3160",
		),
		Entry("31 Hangul vowels and no combining mark", strings.Repeat("\u1161", 31)),
		Entry(
			"31 Hangul compatibility vowels and no combining mark",
			strings.Repeat("\u3160", 31),
		),
	)
})
