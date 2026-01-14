// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package primitives_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/primitives"
)

func TestPrimitives(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Primitives Suite")
}

var _ = Describe("Primitives", func() {
	Describe("IsPrimitive", func() {
		It("Should return true for registered primitives", func() {
			Expect(primitives.IsPrimitive("string")).To(BeTrue())
			Expect(primitives.IsPrimitive("uuid")).To(BeTrue())
			Expect(primitives.IsPrimitive("int32")).To(BeTrue())
			Expect(primitives.IsPrimitive("float64")).To(BeTrue())
			Expect(primitives.IsPrimitive("bytes")).To(BeTrue())
		})

		It("Should return false for non-primitives", func() {
			Expect(primitives.IsPrimitive("MyStruct")).To(BeFalse())
			Expect(primitives.IsPrimitive("CustomType")).To(BeFalse())
			Expect(primitives.IsPrimitive("")).To(BeFalse())
		})
	})

	Describe("Get", func() {
		It("Should return primitive and true for registered types", func() {
			p, ok := primitives.Get("string")
			Expect(ok).To(BeTrue())
			Expect(p.Name).To(Equal("string"))
			Expect(p.Category).To(Equal(primitives.CategoryString))
		})

		It("Should return false for unknown types", func() {
			_, ok := primitives.Get("unknown")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Category checks", func() {
		It("Should correctly identify string types", func() {
			Expect(primitives.IsString("string")).To(BeTrue())
			Expect(primitives.IsString("uuid")).To(BeTrue())
			Expect(primitives.IsString("int32")).To(BeFalse())
		})

		It("Should correctly identify number types", func() {
			Expect(primitives.IsNumber("int8")).To(BeTrue())
			Expect(primitives.IsNumber("int64")).To(BeTrue())
			Expect(primitives.IsNumber("uint32")).To(BeTrue())
			Expect(primitives.IsNumber("float32")).To(BeTrue())
			Expect(primitives.IsNumber("float64")).To(BeTrue())
			Expect(primitives.IsNumber("string")).To(BeFalse())
		})

		It("Should correctly identify temporal types", func() {
			Expect(primitives.IsTemporal("int64")).To(BeFalse())
		})

		It("Should correctly identify boolean types", func() {
			Expect(primitives.IsBoolean("bool")).To(BeTrue())
			Expect(primitives.IsBoolean("string")).To(BeFalse())
		})

		It("Should correctly identify binary types", func() {
			Expect(primitives.IsBinary("bytes")).To(BeTrue())
			Expect(primitives.IsBinary("string")).To(BeFalse())
		})
	})

	Describe("All", func() {
		It("Should return all registered primitives", func() {
			all := primitives.All()
			Expect(len(all)).To(BeNumerically(">=", 15))

			names := make(map[string]bool)
			for _, p := range all {
				names[p.Name] = true
			}

			Expect(names).To(HaveKey("string"))
			Expect(names).To(HaveKey("uuid"))
			Expect(names).To(HaveKey("bytes"))
		})
	})
})

// NOTE: Language-specific mapping tests have been moved to per-language test files:
// - oracle/plugin/go/primitives/mapping_test.go
// - oracle/plugin/py/primitives/mapping_test.go
// - oracle/plugin/ts/primitives/mapping_test.go
// - oracle/plugin/cpp/primitives/mapping_test.go
// - oracle/plugin/pb/primitives/mapping_test.go
