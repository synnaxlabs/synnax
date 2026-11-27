// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Pointer", func() {
	Describe("ParsePointer", func() {
		DescribeTable("Should reject text that is not a JSON Pointer",
			func(text, expected string) {
				Expect(xjson.ParsePointer(text)).Error().
					To(MatchError(expected + ": validation error"))
			},
			Entry("no leading slash", "temperature",
				`json pointer "temperature" must be empty or start with "/"`,
			),
			Entry("a tilde that does not form an escape", "/a~2b",
				`json pointer "/a~2b" is malformed: "~" must be followed by "0" or "1"`,
			),
			Entry("a trailing tilde", "/a~",
				`json pointer "/a~" is malformed: "~" must be followed by "0" or "1"`,
			),
		)
	})

	Describe("String", func() {
		DescribeTable("Should round trip the source text",
			func(text string) {
				Expect(MustSucceed(xjson.ParsePointer(text)).String()).To(Equal(text))
			},
			Entry("root", ""),
			Entry("key", "/temperature"),
			Entry("escapes", "/nested/a~1b/m~0n"),
			Entry("empty token", "/nested/"),
		)
	})

	Describe("HasArrayIndex", func() {
		DescribeTable("Should return whether a token is an array index",
			func(pointer string, expected bool) {
				Expect(MustSucceed(xjson.ParsePointer(pointer)).HasArrayIndex()).
					To(Equal(expected))
			},
			Entry("zero", "/a/0", true),
			Entry("an index after a member", "/a/12/b", true),
			Entry("the root", "", false),
			Entry("members only", "/a/b", false),
			Entry("a leading zero", "/a/01", false),
			Entry("a sign", "/a/-1", false),
			Entry("an empty token", "/a/", false),
			Entry("digits past an int", "/a/99999999999999999999", false),
		)
	})

	Describe("HasPrefix", func() {
		DescribeTable("Should return whether a pointer equals or leads into a prefix",
			func(pointer, prefix string, expected bool) {
				p := MustSucceed(xjson.ParsePointer(pointer))
				Expect(p.HasPrefix(MustSucceed(xjson.ParsePointer(prefix)))).
					To(Equal(expected))
			},
			Entry("the same pointer", "/a/b", "/a/b", true),
			Entry("a pointer into the prefix", "/a/b/c", "/a/b", true),
			Entry("any pointer and the root", "/a", "", true),
			Entry("the root and itself", "", "", true),
			Entry("a pointer that holds the prefix", "/a", "/a/b", false),
			Entry("a sibling", "/a/c", "/a/b", false),
			Entry("a longer token", "/ab", "/a", false),
			Entry("the root and a key", "", "/a", false),
		)
	})
})
