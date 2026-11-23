// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
)

var _ = Describe("Base", func() {
	Describe("Payload", func() {
		binary := v1.Payload{Variant: v1.BinaryPayload{Fields: []v1.BinaryField{
			{Variant: v1.UnsignedBinaryField{BinaryBaseField: v1.BinaryBaseField{
				BaseField: v1.BaseField{Name: "rpm"},
				BitLength: 8,
			}}},
			{},
		}}}
		text := v1.Payload{Variant: v1.TextPayload{Fields: []v1.TextField{
			{Variant: v1.TaggedTextField{BaseField: v1.BaseField{Name: "temp"}}},
			{},
		}}}
		jsonPayload := v1.Payload{Variant: v1.JSONPayload{Fields: []v1.JSONField{
			{Variant: v1.FloatJSONField{JSONBaseField: v1.JSONBaseField{
				BaseField: v1.BaseField{Name: "volts"},
				Pointer:   "/v",
			}}},
			{},
		}}}
		rename := func(b *v1.BaseField) { b.Name += "!" }

		It("Should return the base of every field", func() {
			Expect(binary.FieldBases()).To(Equal([]v1.BaseField{{Name: "rpm"}, {}}))
			Expect(text.FieldBases()).To(Equal([]v1.BaseField{{Name: "temp"}, {}}))
			Expect(
				jsonPayload.FieldBases(),
			).To(Equal([]v1.BaseField{{Name: "volts"}, {}}))
			Expect(v1.Payload{}.FieldBases()).To(BeNil())
		})

		It("Should keep the bits and skip a binary field with no kind", func() {
			updated := binary
			updated.UpdateFieldBases(rename)
			Expect(updated.FieldBases()).To(Equal([]v1.BaseField{{Name: "rpm!"}, {}}))
			Expect(binary.FieldBases()).To(Equal([]v1.BaseField{{Name: "rpm"}, {}}))
			payload, ok := updated.Variant.(v1.BinaryPayload)
			Expect(ok).To(BeTrue())
			Expect(payload.Fields[0].Base().BitLength).To(Equal(uint8(8)))
		})

		It("Should skip a text field with no encoding", func() {
			updated := text
			updated.UpdateFieldBases(rename)
			Expect(updated.FieldBases()).To(Equal([]v1.BaseField{{Name: "temp!"}, {}}))
		})

		It("Should keep the pointer and skip a JSON field with no type", func() {
			updated := jsonPayload
			updated.UpdateFieldBases(rename)
			Expect(updated.FieldBases()).To(Equal([]v1.BaseField{{Name: "volts!"}, {}}))
			payload, ok := updated.Variant.(v1.JSONPayload)
			Expect(ok).To(BeTrue())
			fields := payload.Fields
			Expect(fields[0].Base().Pointer).To(Equal("/v"))
			Expect(jsonPayload.FieldBases()[0].Name).To(Equal("volts"))
		})

		DescribeTable("Should not decode a part of the other format",
			func(payload, msg string) {
				var p v1.Payload
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
