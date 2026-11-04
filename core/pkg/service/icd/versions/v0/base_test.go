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
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
)

var _ = Describe("Base", func() {
	Describe("TextField", func() {
		DescribeTable("Should get and set the base of each encoding",
			func(variant v0.TextFieldVariant) {
				f := v0.TextField{Variant: variant}
				b := v0.BaseField{Key: uuid.New(), Name: "rpm", Scale: 2}
				f.SetBase(b)
				Expect(f.Base()).To(Equal(b))
				Expect(f.Variant).To(BeAssignableToTypeOf(variant))
			},
			Entry("delimited", v0.DelimitedTextField{Position: 2}),
			Entry("tagged", v0.TaggedTextField{Tag: "T="}),
		)

		It("Should keep the rest of the variant when setting the base", func() {
			f := v0.TextField{Variant: v0.TaggedTextField{Tag: "T="}}
			f.SetBase(v0.BaseField{Name: "temp"})
			Expect(f.Variant.(v0.TaggedTextField).Tag).To(Equal("T="))
		})

		It("Should return a zero base for a field with no encoding", func() {
			Expect(v0.TextField{}.Base()).To(Equal(v0.BaseField{}))
		})

		It("Should panic when setting the base of a field with no encoding", func() {
			Expect(func() { (&v0.TextField{}).SetBase(v0.BaseField{}) }).
				To(PanicWith(MatchError("text field has no encoding: <nil>")))
		})
	})

	Describe("JSONField", func() {
		DescribeTable("Should get and set the base of each type",
			func(variant v0.JSONFieldVariant) {
				f := v0.JSONField{Variant: variant}
				b := v0.JSONBaseField{
					BaseField: v0.BaseField{Key: uuid.New(), Name: "rpm", Scale: 2},
					Pointer:   "/rpm",
				}
				f.SetBase(b)
				Expect(f.Base()).To(Equal(b))
				Expect(f.Variant).To(BeAssignableToTypeOf(variant))
			},
			Entry("number", v0.NumberJSONField{}),
			Entry("string", v0.StringJSONField{}),
			Entry("boolean", v0.BooleanJSONField{}),
			Entry("timestamp", v0.TimestampJSONField{Format: v0.TimeFormatUnixMs}),
		)

		It("Should keep the rest of the variant when setting the base", func() {
			f := v0.JSONField{
				Variant: v0.TimestampJSONField{Format: v0.TimeFormatUnixMs},
			}
			f.SetBase(v0.JSONBaseField{Pointer: "/t"})
			Expect(
				f.Variant.(v0.TimestampJSONField).Format,
			).To(Equal(v0.TimeFormatUnixMs))
		})

		It("Should return a zero base for a field with no type", func() {
			Expect(v0.JSONField{}.Base()).To(Equal(v0.JSONBaseField{}))
		})

		It("Should panic when setting the base of a field with no type", func() {
			Expect(func() { (&v0.JSONField{}).SetBase(v0.JSONBaseField{}) }).
				To(PanicWith(MatchError("JSON field has no type: <nil>")))
		})
	})

	Describe("Payload", func() {
		binary := v0.Payload{Variant: v0.BinaryPayload{Fields: []v0.BinaryField{
			{Name: "rpm"},
		}}}
		text := v0.Payload{Variant: v0.TextPayload{Fields: []v0.TextField{
			{Variant: v0.TaggedTextField{BaseField: v0.BaseField{Name: "temp"}}},
			{},
		}}}
		jsonPayload := v0.Payload{Variant: v0.JSONPayload{Fields: []v0.JSONField{
			{Variant: v0.NumberJSONField{JSONBaseField: v0.JSONBaseField{
				BaseField: v0.BaseField{Name: "volts"},
				Pointer:   "/v",
			}}},
			{},
		}}}
		rename := func(b *v0.BaseField) { b.Name += "!" }

		It("Should return the base of every field", func() {
			Expect(binary.FieldBases()).To(Equal([]v0.BaseField{{Name: "rpm"}}))
			Expect(text.FieldBases()).To(Equal([]v0.BaseField{{Name: "temp"}, {}}))
			Expect(
				jsonPayload.FieldBases(),
			).To(Equal([]v0.BaseField{{Name: "volts"}, {}}))
			Expect(v0.Payload{}.FieldBases()).To(BeNil())
		})

		It("Should update the base of every field without touching the source", func() {
			updated := binary
			updated.UpdateFieldBases(rename)
			Expect(updated.FieldBases()).To(Equal([]v0.BaseField{{Name: "rpm!"}}))
			Expect(binary.FieldBases()).To(Equal([]v0.BaseField{{Name: "rpm"}}))
		})

		It("Should skip a text field with no encoding", func() {
			updated := text
			updated.UpdateFieldBases(rename)
			Expect(updated.FieldBases()).To(Equal([]v0.BaseField{{Name: "temp!"}, {}}))
		})

		It("Should keep the pointer and skip a JSON field with no type", func() {
			updated := jsonPayload
			updated.UpdateFieldBases(rename)
			Expect(updated.FieldBases()).To(Equal([]v0.BaseField{{Name: "volts!"}, {}}))
			fields := updated.Variant.(v0.JSONPayload).Fields
			Expect(fields[0].Base().Pointer).To(Equal("/v"))
			Expect(jsonPayload.FieldBases()[0].Name).To(Equal("volts"))
		})

		DescribeTable("Should not decode a part of the other format",
			func(payload, msg string) {
				var p v0.Payload
				Expect(json.Unmarshal([]byte(payload), &p)).
					To(MatchError(ContainSubstring(msg)))
			},
			Entry(
				"text identifier in a binary payload",
				`{"format":"binary","identifier":{"type":"token","prefix":"$"}}`,
				`Identifier: unknown type "token"`,
			),
			Entry(
				"binary field in a text payload",
				`{"format":"text","fields":[{"encoding":"binary","name":"a"}]}`,
				`TextField: unknown encoding "binary"`,
			),
		)
	})
})
