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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Pointer", func() {
	doc := map[string]any{
		"temperature": 23.5,
		"nested":      map[string]any{"a/b": "slash", "m~n": "tilde", "": "empty"},
		"list":        []any{"zero", map[string]any{"k": "one"}},
	}

	Describe("ParsePointer", func() {
		It("Should reject a pointer that does not start with a slash", func() {
			Expect(xjson.ParsePointer("temperature")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`must be empty or start with "/"`)),
			))
		})
		It("Should reject a tilde that does not form an escape", func() {
			Expect(xjson.ParsePointer("/a~2b")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`"~" must be followed by "0" or "1"`)),
			))
		})
		It("Should reject a trailing tilde", func() {
			Expect(xjson.ParsePointer("/a~")).Error().To(
				MatchError(validate.ErrValidation),
			)
		})
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

	Describe("Get", func() {
		DescribeTable("Should resolve a value",
			func(text string, expected any) {
				p := MustSucceed(xjson.ParsePointer(text))
				Expect(MustBeOk(p.Get(doc))).To(Equal(expected))
			},
			Entry("the whole document", "", doc),
			Entry("a top-level key", "/temperature", 23.5),
			Entry("an escaped slash", "/nested/a~1b", "slash"),
			Entry("an escaped tilde", "/nested/m~0n", "tilde"),
			Entry("an empty key", "/nested/", "empty"),
			Entry("an array element", "/list/0", "zero"),
			Entry("a key inside an array element", "/list/1/k", "one"),
		)
		DescribeTable("Should report a missing value",
			func(text string) {
				p := MustSucceed(xjson.ParsePointer(text))
				_, ok := p.Get(doc)
				Expect(ok).To(BeFalse())
			},
			Entry("a missing key", "/pressure"),
			Entry("an index past the end", "/list/2"),
			Entry("a negative index", "/list/-1"),
			Entry("the append marker", "/list/-"),
			Entry("an index with a leading zero", "/list/01"),
			Entry("a non-numeric index", "/list/first"),
			Entry("a path through a scalar", "/temperature/unit"),
		)
	})

	Describe("Set", func() {
		set := func(doc, pointer, value string) string {
			GinkgoHelper()
			var root any
			if doc != "" {
				root = decode(doc)
			}
			p := MustSucceed(xjson.ParsePointer(pointer))
			out := MustSucceed(p.Set(root, decode(value)))
			return string(MustSucceed(json.Marshal(out)))
		}
		DescribeTable("Should place a value",
			func(doc, pointer, value, expected string) {
				Expect(set(doc, pointer, value)).To(MatchJSON(expected))
			},
			Entry("the root of an empty document", "", "", `42`, `42`),
			Entry("a new key of an empty document", "", "/a", `1`, `{"a":1}`),
			Entry("a nested path of an empty document",
				"", "/a/b/c", `"x"`, `{"a":{"b":{"c":"x"}}}`),
			Entry("a new key next to an existing one",
				`{"a":1}`, "/b", `2`, `{"a":1,"b":2}`),
			Entry("an existing key", `{"a":1}`, "/a", `2`, `{"a":2}`),
			Entry("a key below an existing object",
				`{"a":{"b":1}}`, "/a/c", `2`, `{"a":{"b":1,"c":2}}`),
			Entry("an element of an array", `{"a":[1,2]}`, "/a/1", `9`, `{"a":[1,9]}`),
			Entry("an appended element", `{"a":[1]}`, "/a/-", `2`, `{"a":[1,2]}`),
			Entry("an escaped key", "", "/a~1b", `1`, `{"a/b":1}`),
		)
		DescribeTable("Should reject a blocked path",
			func(doc, pointer, message string) {
				p := MustSucceed(xjson.ParsePointer(pointer))
				Expect(p.Set(decode(doc), 1)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
			},
			Entry("a scalar in the path",
				`{"a":1}`, "/a/b", "cannot set /a/b: a scalar value blocks the path"),
			Entry("an index past the end of an array",
				`{"a":[1]}`, "/a/3", `"3" is not an index of an array of 1 elements`),
			Entry("a key into an array",
				`{"a":[1]}`, "/a/b", `"b" is not an index of an array of 1 elements`),
		)
	})
})
