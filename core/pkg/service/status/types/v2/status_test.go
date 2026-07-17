// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/synnax/pkg/service/status/types/v2"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Status.String", func() {
	DescribeTable("Should render the variant icon for each variant",
		func(variant v2.Variant, expected string) {
			Expect(v2.Status[any]{Variant: variant}.String()).To(Equal(expected))
		},
		Entry("info", v2.VariantInfo, "[ℹ info]"),
		Entry("success", v2.VariantSuccess, "[✓ success]"),
		Entry("error", v2.VariantError, "[✗ error]"),
		Entry("warning", v2.VariantWarning, "[⚠ warning]"),
		Entry("disabled", v2.VariantDisabled, "[⊘ disabled]"),
		Entry("loading", v2.VariantLoading, "[◌ loading]"),
		Entry("unknown falls back to a bullet", v2.Variant("custom"), "[• custom]"),
	)

	It("Should render the name when present", func() {
		s := v2.Status[any]{Variant: v2.VariantInfo, Name: "My Status"}
		Expect(s.String()).To(Equal("[ℹ info] My Status"))
	})

	It("Should render the key in parentheses when it differs from the name", func() {
		s := v2.Status[any]{Variant: v2.VariantInfo, Name: "My Status", Key: "abc"}
		Expect(s.String()).To(Equal("[ℹ info] My Status (abc)"))
	})

	It("Should not render the key when it equals the name", func() {
		s := v2.Status[any]{Variant: v2.VariantInfo, Name: "same", Key: "same"}
		Expect(s.String()).To(Equal("[ℹ info] same"))
	})

	It("Should not render a key in parentheses when the key is empty", func() {
		s := v2.Status[any]{Variant: v2.VariantInfo, Name: "n"}
		Expect(s.String()).ToNot(ContainSubstring("("))
	})

	It("Should render the message after a colon", func() {
		s := v2.Status[any]{Variant: v2.VariantError, Message: "boom"}
		Expect(s.String()).To(Equal("[✗ error]: boom"))
	})

	It("Should render the description on its own line", func() {
		s := v2.Status[any]{
			Variant:     v2.VariantInfo,
			Message:     "m",
			Description: "more detail",
		}
		Expect(s.String()).To(ContainSubstring("\n  more detail"))
	})

	It("Should render the time when non-zero", func() {
		s := v2.Status[any]{Variant: v2.VariantInfo, Time: telem.Now()}
		Expect(s.String()).To(ContainSubstring("\n  @ "))
	})

	It("Should not render the time when zero", func() {
		s := v2.Status[any]{Variant: v2.VariantInfo}
		Expect(s.String()).ToNot(ContainSubstring("@"))
	})

	It("Should render a fully-populated status with every field", func() {
		s := v2.Status[map[string]any]{
			Variant:     v2.VariantWarning,
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
		s := v2.Status[any]{Variant: v2.VariantInfo, Message: "hello"}
		Expect(s.String()).ToNot(ContainSubstring("Details"))
	})

	It("Should suppress a zero numeric Details value", func() {
		s := v2.Status[int]{Variant: v2.VariantInfo, Message: "hello", Details: 0}
		Expect(s.String()).ToNot(ContainSubstring("Details"))
	})

	It("Should render a Details line when Details is non-zero", func() {
		s := v2.Status[map[string]any]{
			Variant: v2.VariantInfo,
			Message: "hello",
			Details: map[string]any{"running": true},
		}
		Expect(s.String()).To(ContainSubstring("Details: map[running:true]"))
	})

	It("Should render a non-zero string Details that stringifies to \"0\"", func() {
		s := v2.Status[string]{
			Variant: v2.VariantInfo,
			Message: "hello",
			Details: "0",
		}
		Expect(s.String()).To(ContainSubstring("Details: 0"))
	})
})
