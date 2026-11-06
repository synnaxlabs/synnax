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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
)

var _ = Describe("Base", func() {
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
