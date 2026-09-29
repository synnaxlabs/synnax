// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/library/versions/v0"
)

var _ = Describe("Base", func() {
	Describe("Entry", func() {
		DescribeTable("Should get and set the base of each kind",
			func(variant v0.EntryVariant) {
				e := v0.Entry{Variant: variant}
				b := v0.BaseEntry{Key: uuid.New(), Name: "State"}
				e.SetBase(b)
				Expect(e.Base()).To(Equal(b))
				Expect(e.Variant).To(BeAssignableToTypeOf(variant))
			},
			Entry("enum", v0.EnumEntry{}),
			Entry("message", v0.MessageEntry{Format: v0.FormatText}),
		)

		It("Should keep the rest of the variant when setting the base", func() {
			e := v0.Entry{Variant: v0.MessageEntry{Format: v0.FormatText}}
			e.SetBase(v0.BaseEntry{Name: "Status"})
			Expect(e.Variant.(v0.MessageEntry).Format).To(Equal(v0.FormatText))
		})

		It("Should return a zero base for an entry with no kind", func() {
			Expect(v0.Entry{}.Base()).To(Equal(v0.BaseEntry{}))
		})

		It("Should panic when setting the base of an entry with no kind", func() {
			Expect(func() { (&v0.Entry{}).SetBase(v0.BaseEntry{}) }).
				To(PanicWith(MatchError("entry has no kind: <nil>")))
		})
	})

	Describe("Field", func() {
		DescribeTable("Should get and set the base of each encoding",
			func(variant v0.FieldVariant) {
				f := v0.Field{Variant: variant}
				b := v0.BaseField{Key: uuid.New(), Name: "rpm", Scale: 2}
				f.SetBase(b)
				Expect(f.Base()).To(Equal(b))
				Expect(f.Variant).To(BeAssignableToTypeOf(variant))
			},
			Entry("binary", v0.BinaryField{BitLength: 16}),
			Entry("delimited", v0.DelimitedField{Position: 2}),
			Entry("tagged", v0.TaggedField{Tag: "T="}),
		)

		It("Should keep the rest of the variant when setting the base", func() {
			f := v0.Field{Variant: v0.TaggedField{Tag: "T="}}
			f.SetBase(v0.BaseField{Name: "temp"})
			Expect(f.Variant.(v0.TaggedField).Tag).To(Equal("T="))
		})

		It("Should return a zero base for a field with no encoding", func() {
			Expect(v0.Field{}.Base()).To(Equal(v0.BaseField{}))
		})

		It("Should panic when setting the base of a field with no encoding", func() {
			Expect(func() { (&v0.Field{}).SetBase(v0.BaseField{}) }).
				To(PanicWith(MatchError("field has no encoding: <nil>")))
		})
	})
})
