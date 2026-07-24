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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v1 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v1"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Status", func() {
	Describe("String", func() {
		DescribeTable("Should render the variant icon for each variant",
			func(variant v1.Variant, expected string) {
				Expect(v1.Status[any]{Variant: variant}.String()).To(Equal(expected))
			},
			Entry("info", v1.VariantInfo, "[ℹ info]"),
			Entry("success", v1.VariantSuccess, "[✓ success]"),
			Entry("error", v1.VariantError, "[✗ error]"),
			Entry("warning", v1.VariantWarning, "[⚠ warning]"),
			Entry("disabled", v1.VariantDisabled, "[⊘ disabled]"),
			Entry("loading", v1.VariantLoading, "[◌ loading]"),
			Entry("unknown falls back to a bullet", v1.Variant("custom"), "[• custom]"),
		)

		It("Should render the name when present", func() {
			s := v1.Status[any]{Variant: v1.VariantInfo, Name: "My Status"}
			Expect(s.String()).To(Equal("[ℹ info] My Status"))
		})

		It("Should render the key in parentheses when it differs from the name", func() {
			s := v1.Status[any]{Variant: v1.VariantInfo, Name: "My Status", Key: "abc"}
			Expect(s.String()).To(Equal("[ℹ info] My Status (abc)"))
		})

		It("Should not render the key when it equals the name", func() {
			s := v1.Status[any]{Variant: v1.VariantInfo, Name: "same", Key: "same"}
			Expect(s.String()).To(Equal("[ℹ info] same"))
		})

		It("Should not render a key in parentheses when the key is empty", func() {
			s := v1.Status[any]{Variant: v1.VariantInfo, Name: "n"}
			Expect(s.String()).ToNot(ContainSubstring("("))
		})

		It("Should render the message after a colon", func() {
			s := v1.Status[any]{Variant: v1.VariantError, Message: "boom"}
			Expect(s.String()).To(Equal("[✗ error]: boom"))
		})

		It("Should render the description on its own line", func() {
			s := v1.Status[any]{
				Variant:     v1.VariantInfo,
				Message:     "m",
				Description: "more detail",
			}
			Expect(s.String()).To(ContainSubstring("\n  more detail"))
		})

		It("Should render the time when non-zero", func() {
			s := v1.Status[any]{Variant: v1.VariantInfo, Time: telem.Now()}
			Expect(s.String()).To(ContainSubstring("\n  @ "))
		})

		It("Should not render the time when zero", func() {
			s := v1.Status[any]{Variant: v1.VariantInfo}
			Expect(s.String()).ToNot(ContainSubstring("@"))
		})

		It("Should render a fully-populated status with every field", func() {
			s := v1.Status[map[string]any]{
				Variant:     v1.VariantWarning,
				Name:        "Acquire",
				Key:         "task-1",
				Message:     "acquiring",
				Description: "5 channels",
				Details:     map[string]any{"running": true},
			}
			out := s.String()
			Expect(out).To(HavePrefix("[⚠ warning] Acquire (task-1): acquiring"))
			Expect(out).To(ContainSubstring("\n  5 channels"))
			Expect(out).To(ContainSubstring("\n  Details: map[running:true]"))
		})

		It("Should not render a Details line when Details is the zero value", func() {
			s := v1.Status[any]{Variant: v1.VariantInfo, Message: "hello"}
			Expect(s.String()).ToNot(ContainSubstring("Details"))
		})

		It("Should suppress a zero numeric Details value", func() {
			s := v1.Status[int]{Variant: v1.VariantInfo, Message: "hello", Details: 0}
			Expect(s.String()).ToNot(ContainSubstring("Details"))
		})

		It("Should render a Details line when Details is non-zero", func() {
			s := v1.Status[map[string]any]{
				Variant: v1.VariantInfo,
				Message: "hello",
				Details: map[string]any{"running": true},
			}
			Expect(s.String()).To(ContainSubstring("Details: map[running:true]"))
		})

		It("Should render a non-zero string Details that stringifies to \"0\"", func() {
			s := v1.Status[string]{
				Variant: v1.VariantInfo,
				Message: "hello",
				Details: "0",
			}
			Expect(s.String()).To(ContainSubstring("Details: 0"))
		})
	})

	Describe("GorpKey", func() {
		It("Should return the status's key", func() {
			Expect(v1.Status[any]{Key: "st-1"}.GorpKey()).To(Equal("st-1"))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v1.Status[any]{}.SetOptions()).To(BeNil())
		})
	})
	Describe("OntologyID", func() {
		It("Should return the status ontology identifier", func() {
			Expect(v1.Status[any]{Key: "st-1"}.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeStatus, Key: "st-1",
			}))
		})
	})

	Describe("CustomTypeName", func() {
		It("Should return the Status gorp type name", func() {
			Expect(v1.Status[any]{}.CustomTypeName()).To(Equal("Status"))
		})
	})
})
