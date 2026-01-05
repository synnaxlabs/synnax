// Copyright 2025 Synnax Labs, Inc.
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
			Expect(primitives.IsPrimitive("timestamp")).To(BeTrue())
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
			p, ok := primitives.Get("timestamp")
			Expect(ok).To(BeTrue())
			Expect(p.Name).To(Equal("timestamp"))
			Expect(p.Category).To(Equal(primitives.CategoryTemporal))
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
			Expect(primitives.IsString("color")).To(BeTrue())
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
			Expect(primitives.IsTemporal("timestamp")).To(BeTrue())
			Expect(primitives.IsTemporal("timespan")).To(BeTrue())
			Expect(primitives.IsTemporal("time_range")).To(BeTrue())
			Expect(primitives.IsTemporal("time_range_bounded")).To(BeTrue())
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
			Expect(len(all)).To(BeNumerically(">=", 20))

			names := make(map[string]bool)
			for _, p := range all {
				names[p.Name] = true
			}

			Expect(names).To(HaveKey("string"))
			Expect(names).To(HaveKey("uuid"))
			Expect(names).To(HaveKey("timestamp"))
			Expect(names).To(HaveKey("bytes"))
		})
	})
})

var _ = Describe("Mapping", func() {
	Describe("GetMapping", func() {
		It("Should return Go mapping for primitives", func() {
			m := primitives.GetMapping("uuid", "go")
			Expect(m.TargetType).To(Equal("uuid.UUID"))
			Expect(m.Imports).To(HaveLen(1))
			Expect(m.Imports[0].Path).To(Equal("github.com/google/uuid"))
		})

		It("Should return Python mapping for primitives", func() {
			m := primitives.GetMapping("uuid", "py")
			Expect(m.TargetType).To(Equal("UUID"))
			Expect(m.Imports).To(HaveLen(1))
			Expect(m.Imports[0].Name).To(Equal("UUID"))
		})

		It("Should return TypeScript Zod mapping for primitives", func() {
			m := primitives.GetMapping("uuid", "ts")
			Expect(m.TargetType).To(Equal("z.string().uuid()"))
		})

		It("Should return C++ mapping for primitives", func() {
			m := primitives.GetMapping("uuid", "cpp")
			Expect(m.TargetType).To(Equal("std::string"))
		})

		It("Should return Protobuf mapping for primitives", func() {
			m := primitives.GetMapping("uuid", "pb")
			Expect(m.TargetType).To(Equal("string"))
		})

		It("Should return any for unknown language", func() {
			m := primitives.GetMapping("uuid", "unknown")
			Expect(m.TargetType).To(Equal("any"))
		})

		It("Should return any for unknown primitive", func() {
			m := primitives.GetMapping("unknown", "go")
			Expect(m.TargetType).To(Equal("any"))
		})
	})

	Describe("Go mappings", func() {
		It("Should have correct temporal type mappings", func() {
			Expect(primitives.GoMapping["timestamp"].TargetType).To(Equal("telem.TimeStamp"))
			Expect(primitives.GoMapping["timespan"].TargetType).To(Equal("telem.TimeSpan"))
			Expect(primitives.GoMapping["time_range"].TargetType).To(Equal("telem.TimeRange"))
		})

		It("Should have correct number type mappings", func() {
			Expect(primitives.GoMapping["int8"].TargetType).To(Equal("int8"))
			Expect(primitives.GoMapping["int64"].TargetType).To(Equal("int64"))
			Expect(primitives.GoMapping["float64"].TargetType).To(Equal("float64"))
		})
	})

	Describe("Python mappings", func() {
		It("Should map integers to int", func() {
			Expect(primitives.PythonMapping["int8"].TargetType).To(Equal("int"))
			Expect(primitives.PythonMapping["int64"].TargetType).To(Equal("int"))
			Expect(primitives.PythonMapping["uint32"].TargetType).To(Equal("int"))
		})

		It("Should map floats to float", func() {
			Expect(primitives.PythonMapping["float32"].TargetType).To(Equal("float"))
			Expect(primitives.PythonMapping["float64"].TargetType).To(Equal("float"))
		})
	})

	Describe("C++ mappings", func() {
		It("Should use fixed-width integer types", func() {
			Expect(primitives.CppMapping["int8"].TargetType).To(Equal("std::int8_t"))
			Expect(primitives.CppMapping["int64"].TargetType).To(Equal("std::int64_t"))
			Expect(primitives.CppMapping["uint32"].TargetType).To(Equal("std::uint32_t"))
		})

		It("Should have correct system includes", func() {
			Expect(primitives.CppMapping["string"].Imports[0].Path).To(Equal("string"))
			Expect(primitives.CppMapping["string"].Imports[0].Category).To(Equal("system"))
		})
	})
})
