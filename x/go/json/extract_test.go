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
	"encoding/json/jsontext"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	. "github.com/synnaxlabs/x/testutil"
)

// pointers parses each text form into a Pointer.
func pointers(texts ...string) []xjson.Pointer {
	GinkgoHelper()
	parsed := make([]xjson.Pointer, len(texts))
	for i, p := range texts {
		parsed[i] = MustSucceed(xjson.ParsePointer(p))
	}
	return parsed
}

// extract resolves the pointers in doc and returns each value's text, with "-" for a
// pointer that leads to no value.
func extract(doc string, texts ...string) []string {
	GinkgoHelper()
	values := make([]jsontext.Value, len(texts))
	Expect(xjson.NewExtractor(pointers(texts...)).Extract([]byte(doc), values)).
		To(Succeed())
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = "-"
		if v != nil {
			out[i] = string(v)
		}
	}
	return out
}

var _ = Describe("Extractor", func() {
	It("Should resolve each pointer to the text of its value", func() {
		doc := `{"n":9007199254740993,"s":"o\nn","t":true,"z":null,"o":{"a": [1]}}`
		Expect(extract(doc, "/n", "/s", "/t", "/z", "/o")).To(Equal([]string{
			"9007199254740993", `"o\nn"`, "true", "null", `{"a": [1]}`,
		}))
	})

	It("Should resolve the root pointer to the whole document", func() {
		Expect(extract(`[1,"a"]`, "")).To(Equal([]string{`[1,"a"]`}))
	})

	It("Should resolve a value and a value nested inside it", func() {
		Expect(extract(`{"a":{"b":2}}`, "/a/b", "/a")).
			To(Equal([]string{"2", `{"b":2}`}))
	})

	It("Should resolve array elements by index", func() {
		Expect(extract(`{"a":[10,20,{"b":30}]}`, "/a/1", "/a/2/b")).
			To(Equal([]string{"20", "30"}))
	})

	It("Should resolve escaped tokens", func() {
		Expect(extract(`{"a/b":1,"c~d":2}`, "/a~1b", "/c~0d")).
			To(Equal([]string{"1", "2"}))
	})

	It("Should match an object member whose name holds an escape", func() {
		Expect(extract(`{"\u0061":1}`, "/a")).To(Equal([]string{"1"}))
	})

	DescribeTable("Should find no value for a pointer that leads to none",
		func(doc, pointer string) {
			Expect(extract(doc, pointer)).To(Equal([]string{"-"}))
		},
		Entry("a missing member", `{"a":1}`, "/b"),
		Entry("an index past the end", `[1]`, "/1"),
		Entry("an index with a leading zero", `[1,2]`, "/01"),
		Entry("a member of a scalar", `{"a":1}`, "/a/b"),
		Entry("an index into an object", `{"a":{"0":1}}`, "/a/1"),
	)

	It("Should resolve a member name made of digits in an object", func() {
		Expect(extract(`{"a":{"01":1}}`, "/a/01")).To(Equal([]string{"1"}))
	})

	It("Should allow repeated names that no pointer addresses", func() {
		Expect(extract(`{"x":1,"x":2,"a":3}`, "/a")).To(Equal([]string{"3"}))
	})

	It("Should clear the values of the last document", func() {
		e := xjson.NewExtractor(pointers("/a"))
		values := make([]jsontext.Value, 1)
		Expect(e.Extract([]byte(`{"a":1}`), values)).To(Succeed())
		Expect(e.Extract([]byte(`{"b":1}`), values)).To(Succeed())
		Expect(values[0]).To(BeNil())
	})

	It("Should keep the values of a document valid while it reads a nested one",
		func() {
			Expect(extract(`{"a":{"b":[1,{"c":2}]},"d":3}`, "/a", "/a/b/1/c", "/d")).
				To(Equal([]string{`{"b":[1,{"c":2}]}`, "2", "3"}))
		},
	)

	DescribeTable("Should reject a document it cannot read",
		func(doc, message string) {
			values := make([]jsontext.Value, 1)
			Expect(xjson.NewExtractor(pointers("/a")).Extract([]byte(doc), values)).
				To(MatchError(ContainSubstring(message)))
		},
		Entry("malformed JSON", `{"a":`, "unexpected EOF"),
		Entry("data after the document", `{"a":1} 2`,
			"unexpected data after the top-level value"),
		Entry("a repeated addressed name", `{"a":1,"a":2}`, "duplicate value at /a"),
		Entry("invalid UTF-8", "{\"a\":\"\xff\"}", "invalid UTF-8"),
	)
})
