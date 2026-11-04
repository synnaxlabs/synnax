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

// extract resolves pointers in doc and returns the values and whether each was found.
func extract(doc string, pointers ...string) ([]any, []bool) {
	GinkgoHelper()
	parsed := make([]xjson.Pointer, len(pointers))
	for i, p := range pointers {
		parsed[i] = MustSucceed(xjson.ParsePointer(p))
	}
	values, found := make([]any, len(pointers)), make([]bool, len(pointers))
	Expect(xjson.NewExtractor(parsed).Extract([]byte(doc), values, found)).
		To(Succeed())
	return values, found
}

var _ = Describe("Extractor", func() {
	It("Should resolve each pointer to the value that Decode gives", func() {
		doc := `{"n":9007199254740993,"s":"on","t":true,"z":null,"o":{"a":[1]}}`
		values, found := extract(doc, "/n", "/s", "/t", "/z", "/o")
		Expect(found).To(HaveEach(BeTrue()))
		Expect(values).To(Equal([]any{
			xjson.NumberText("9007199254740993"),
			"on",
			true,
			nil,
			map[string]any{"a": []any{xjson.NumberText("1")}},
		}))
	})

	It("Should resolve the root pointer to the whole document", func() {
		values, found := extract(`[1,"a"]`, "")
		Expect(found).To(Equal([]bool{true}))
		Expect(values[0]).To(Equal([]any{xjson.NumberText("1"), "a"}))
	})

	It("Should resolve a value and a value nested inside it", func() {
		values, found := extract(`{"a":{"b":2}}`, "/a/b", "/a")
		Expect(found).To(Equal([]bool{true, true}))
		Expect(values[0]).To(Equal(xjson.NumberText("2")))
		Expect(values[1]).To(Equal(map[string]any{"b": xjson.NumberText("2")}))
	})

	It("Should resolve array elements by index", func() {
		values, found := extract(`{"a":[10,20,{"b":30}]}`, "/a/1", "/a/2/b")
		Expect(found).To(Equal([]bool{true, true}))
		Expect(values).To(Equal([]any{xjson.NumberText("20"), xjson.NumberText("30")}))
	})

	It("Should resolve escaped tokens", func() {
		values, found := extract(`{"a/b":1,"c~d":2}`, "/a~1b", "/c~0d")
		Expect(found).To(Equal([]bool{true, true}))
		Expect(values).To(Equal([]any{xjson.NumberText("1"), xjson.NumberText("2")}))
	})

	DescribeTable("Should not find a pointer that leads to no value",
		func(doc, pointer string) {
			_, found := extract(doc, pointer)
			Expect(found).To(Equal([]bool{false}))
		},
		Entry("a missing member", `{"a":1}`, "/b"),
		Entry("an index past the end", `[1]`, "/1"),
		Entry("an index with a leading zero", `[1,2]`, "/01"),
		Entry("a member of a scalar", `{"a":1}`, "/a/b"),
		Entry("an index into an object", `{"a":{"0":1}}`, "/a/1"),
	)

	It("Should resolve a member name made of digits in an object", func() {
		values, found := extract(`{"a":{"01":1}}`, "/a/01")
		Expect(found).To(Equal([]bool{true}))
		Expect(values[0]).To(Equal(xjson.NumberText("1")))
	})

	It("Should keep the last of repeated names that no pointer addresses", func() {
		values, found := extract(`{"x":1,"x":2,"a":3}`, "/a")
		Expect(found).To(Equal([]bool{true}))
		Expect(values[0]).To(Equal(xjson.NumberText("3")))
	})

	It("Should replace invalid UTF-8 in a string", func() {
		values, _ := extract("{\"a\":\"\xff\"}", "/a")
		Expect(values[0]).To(Equal("�"))
	})

	It("Should clear found values between documents", func() {
		e := xjson.NewExtractor([]xjson.Pointer{{"a"}})
		values, found := make([]any, 1), make([]bool, 1)
		Expect(e.Extract([]byte(`{"a":1}`), values, found)).To(Succeed())
		Expect(found[0]).To(BeTrue())
		Expect(e.Extract([]byte(`{"b":1}`), values, found)).To(Succeed())
		Expect(found[0]).To(BeFalse())
	})

	DescribeTable("Should reject a document it cannot read",
		func(doc, message string) {
			e := xjson.NewExtractor([]xjson.Pointer{{"a"}})
			values, found := make([]any, 1), make([]bool, 1)
			Expect(e.Extract([]byte(doc), values, found)).
				To(MatchError(ContainSubstring(message)))
		},
		Entry("malformed JSON", `{"a":`, "unexpected EOF"),
		Entry("data after the document", `{"a":1} 2`,
			"unexpected data after the top-level value"),
		Entry("a repeated addressed name", `{"a":1,"a":2}`, "duplicate value at /a"),
	)
})
