// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolver_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/resolver"
)

var _ = Describe("ConfigurableFormatter", func() {
	Describe("Go configuration", func() {
		var f resolver.TypeFormatter

		BeforeEach(func() {
			f = resolver.NewFormatter(resolver.GoFormatterConfig)
		})

		It("Should qualify names with the configured separator", func() {
			Expect(f.FormatQualified("telem", "Rate")).To(Equal("telem.Rate"))
		})

		It("Should return the bare name for an empty qualifier", func() {
			Expect(f.FormatQualified("", "Rate")).To(Equal("Rate"))
		})

		It("Should format generics with the configured brackets", func() {
			Expect(f.FormatGeneric("Series", []string{"K", "V"})).
				To(Equal("Series[K, V]"))
		})

		It("Should return the base name when there are no type args", func() {
			Expect(f.FormatGeneric("Series", nil)).To(Equal("Series"))
		})

		It("Should format dynamic and fixed arrays", func() {
			Expect(f.FormatArray("int32")).To(Equal("[]int32"))
			Expect(f.FormatFixedArray("byte", 16)).To(Equal("[16]byte"))
		})

		It("Should format maps", func() {
			Expect(f.FormatMap("string", "int32")).To(Equal("map[string]int32"))
		})

		It("Should expose the configured fallback type", func() {
			Expect(f.FallbackType()).To(Equal("any"))
		})
	})

	Describe("Default configuration", func() {
		var f resolver.TypeFormatter

		BeforeEach(func() {
			f = resolver.NewFormatter(resolver.FormatterConfig{})
		})

		It("Should default arrays to the Go-like []T form", func() {
			Expect(f.FormatArray("string")).To(Equal("[]string"))
		})

		It("Should fall back to the dynamic array form for fixed arrays", func() {
			Expect(f.FormatFixedArray("string", 4)).To(Equal("[]string"))
		})

		It("Should default maps to the Go-like map[K]V form", func() {
			Expect(f.FormatMap("string", "bool")).To(Equal("map[string]bool"))
		})
	})

	Describe("SkipGenerics", func() {
		It("Should drop type arguments entirely", func() {
			f := resolver.NewFormatter(resolver.FormatterConfig{SkipGenerics: true})
			Expect(f.FormatGeneric("Series", []string{"T"})).To(Equal("Series"))
		})
	})
})
