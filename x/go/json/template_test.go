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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// numberedSlots fills the slot of field i with i.
type numberedSlots struct{}

func (numberedSlots) AppendSlot(dst []byte, field int) ([]byte, error) {
	return strconv.AppendInt(dst, int64(field), 10), nil
}

// field returns a TemplateField at pointer, a slot when literal is empty.
func field(pointer, literal string) xjson.TemplateField {
	GinkgoHelper()
	f := xjson.TemplateField{Pointer: MustSucceed(xjson.ParsePointer(pointer))}
	if literal != "" {
		f.Literal = []byte(literal)
	}
	return f
}

func render(fields ...xjson.TemplateField) string {
	GinkgoHelper()
	t := MustSucceed(xjson.NewTemplate(fields))
	return string(MustSucceed(t.Append(nil, numberedSlots{})))
}

var _ = Describe("Template", func() {
	DescribeTable("Should render the fields in their order",
		func(expected string, fields ...xjson.TemplateField) {
			Expect(render(fields...)).To(Equal(expected))
		},
		Entry("one slot at the root", "0", field("", "")),
		Entry("one literal at the root", `"x"`, field("", `"x"`)),
		Entry("members in field order", `{"z":0,"a":1}`, field("/z", ""),
			field("/a", "")),
		Entry("nested objects", `{"v":{"x":0,"y":"c"},"t":2}`, field("/v/x", ""),
			field("/v/y", `"c"`), field("/t", "")),
		Entry("an index token as a member name", `{"a":{"0":0}}`, field("/a/0", "")),
		Entry("names that need escapes", `{"a\"b":{"c/d":0}}`, field("/a\"b/c~1d", "")),
		Entry("no fields", `{}`),
	)

	It("Should render the same text on every append", func() {
		t := MustSucceed(xjson.NewTemplate([]xjson.TemplateField{
			field("/a", ""), field("/b", "true"),
		}))
		first := MustSucceed(t.Append(nil, numberedSlots{}))
		Expect(string(MustSucceed(t.Append(first, numberedSlots{})))).
			To(Equal(`{"a":0,"b":true}{"a":0,"b":true}`))
	})

	It("Should return the error of a slot", func() {
		t := MustSucceed(xjson.NewTemplate([]xjson.TemplateField{field("/a", "")}))
		Expect(t.Append(nil, failingSlots{})).Error().To(MatchError("no value"))
	})

	DescribeTable("Should reject pointers that overlap",
		func(message string, fields ...xjson.TemplateField) {
			Expect(xjson.NewTemplate(fields)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(message)),
			))
		},
		Entry("two equal pointers", `pointers "/a" and "/a" overlap`,
			field("/a", ""), field("/a", "")),
		Entry("a pointer inside another", `pointers "/a" and "/a/b" overlap`,
			field("/a", ""), field("/a/b", "")),
		Entry("a pointer that holds another", `pointers "/a" and "/a/b" overlap`,
			field("/a/b", ""), field("/a", "")),
		Entry("the root and another pointer", `pointers "" and "/a" overlap`,
			field("", ""), field("/a", "")),
		Entry("another pointer and the root", `pointers "" and "/a" overlap`,
			field("/a", ""), field("", "")),
	)
})

type failingSlots struct{}

func (failingSlots) AppendSlot([]byte, int) ([]byte, error) {
	return nil, errors.New("no value")
}
