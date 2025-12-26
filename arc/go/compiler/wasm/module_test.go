// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/arc/compiler/wasm"
)

var _ = Describe("WASM Module", func() {
	Describe("Module Creation", func() {
		It("Should create an empty module", func() {
			mod := wasm.NewModule()
			Expect(mod).ToNot(BeNil())
			types, functions, exports := mod.Debug()
			Expect(types).To(Equal(0))
			Expect(functions).To(Equal(0))
			Expect(exports).To(Equal(0))
		})

		It("Should generate valid WASM magic number and version", func() {
			mod := wasm.NewModule()
			bytes := mod.Generate()
			// Check magic number: 0x00 0x61 0x73 0x6d (\0asm)
			Expect(bytes[0:4]).To(Equal([]byte{0x00, 0x61, 0x73, 0x6d}))
			// Check version: 1 (little-endian)
			Expect(bytes[4:8]).To(Equal([]byte{0x01, 0x00, 0x00, 0x00}))
		})
	})

	Describe("Function Types", func() {
		It("Should add a simple function type", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32, wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			idx := mod.AddType(ft)
			Expect(idx).To(Equal(uint32(0)))
			types, _, _ := mod.Debug()
			Expect(types).To(Equal(1))
		})

		It("Should deduplicate identical function types", func() {
			mod := wasm.NewModule()
			ft1 := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			ft2 := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			idx1 := mod.AddType(ft1)
			idx2 := mod.AddType(ft2)
			Expect(idx1).To(Equal(idx2))
			types, _, _ := mod.Debug()
			Expect(types).To(Equal(1)) // Should only have one type
		})

		It("Should distinguish different function types", func() {
			mod := wasm.NewModule()
			ft1 := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			ft2 := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I64},
				Results: []wasm.ValueType{wasm.I64},
			}
			idx1 := mod.AddType(ft1)
			idx2 := mod.AddType(ft2)
			Expect(idx1).ToNot(Equal(idx2))
			types, _, _ := mod.Debug()
			Expect(types).To(Equal(2))
		})

		It("Should handle function type with no parameters", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{},
				Results: []wasm.ValueType{wasm.I32},
			}
			idx := mod.AddType(ft)
			Expect(idx).To(Equal(uint32(0)))
		})

		It("Should handle function type with no results", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{},
			}
			idx := mod.AddType(ft)
			Expect(idx).To(Equal(uint32(0)))
		})

		It("Should handle function type with multiple parameters and results", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32, wasm.I64, wasm.F32, wasm.F64},
				Results: []wasm.ValueType{wasm.I32, wasm.I64},
			}
			idx := mod.AddType(ft)
			Expect(idx).To(Equal(uint32(0)))
		})
	})

	Describe("Imports", func() {
		It("Should add an imported function", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			idx := mod.AddImport("env", "imported_func", ft)
			Expect(idx).To(Equal(uint32(0)))
		})

		It("Should assign sequential indices to imports", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			idx1 := mod.AddImport("env", "func1", ft)
			idx2 := mod.AddImport("env", "func2", ft)
			Expect(idx1).To(Equal(uint32(0)))
			Expect(idx2).To(Equal(uint32(1)))
		})

		It("Should automatically add function type when adding import", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{wasm.I64},
			}
			mod.AddImport("env", "func", ft)
			types, _, _ := mod.Debug()
			Expect(types).To(Equal(1))
		})
	})

	Describe("ImportCount", func() {
		It("Should return zero for module with no imports", func() {
			mod := wasm.NewModule()
			Expect(mod.ImportCount()).To(Equal(uint32(0)))
		})

		It("Should return correct count after adding imports", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			mod.AddImport("env", "func1", ft)
			Expect(mod.ImportCount()).To(Equal(uint32(1)))

			mod.AddImport("env", "func2", ft)
			Expect(mod.ImportCount()).To(Equal(uint32(2)))

			mod.AddImport("env", "func3", ft)
			Expect(mod.ImportCount()).To(Equal(uint32(3)))
		})

		It("Should not be affected by adding local functions", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			mod.AddImport("env", "imported_func", ft)
			Expect(mod.ImportCount()).To(Equal(uint32(1)))

			// Add local function
			typeIdx := mod.AddType(ft)
			mod.AddFunction(typeIdx, []wasm.ValueType{}, []byte{0x20, 0x00})
			Expect(mod.ImportCount()).To(Equal(uint32(1))) // Still 1 import
		})

		It("Should not be affected by adding exports", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{},
				Results: []wasm.ValueType{wasm.I32},
			}
			mod.AddImport("env", "func", ft)
			Expect(mod.ImportCount()).To(Equal(uint32(1)))

			typeIdx := mod.AddType(ft)
			funcIdx := mod.AddFunction(typeIdx, []wasm.ValueType{}, []byte{0x41, 0x2a})
			mod.AddExport("exported", wasm.ExportFunc, funcIdx)
			Expect(mod.ImportCount()).To(Equal(uint32(1))) // Still 1 import
		})
	})

	Describe("Functions", func() {
		It("Should add a function with empty body", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{},
				Results: []wasm.ValueType{},
			}
			typeIdx := mod.AddType(ft)
			funcIdx := mod.AddFunction(typeIdx, []wasm.ValueType{}, []byte{})
			Expect(funcIdx).To(Equal(uint32(0)))
			_, functions, _ := mod.Debug()
			Expect(functions).To(Equal(1))
		})

		It("Should add function with locals", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			typeIdx := mod.AddType(ft)
			locals := []wasm.ValueType{wasm.I32, wasm.I64}
			body := []byte{0x20, 0x00, 0x0b} // local.get 0, end
			funcIdx := mod.AddFunction(typeIdx, locals, body)
			Expect(funcIdx).To(Equal(uint32(0)))
		})

		It("Should assign function indices after imports", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{},
				Results: []wasm.ValueType{},
			}
			// Add 2 imports
			mod.AddImport("env", "func1", ft)
			mod.AddImport("env", "func2", ft)
			// Add a local function
			typeIdx := mod.AddType(ft)
			funcIdx := mod.AddFunction(typeIdx, []wasm.ValueType{}, []byte{})
			// Function index should be after imports
			Expect(funcIdx).To(Equal(uint32(2)))
		})
	})

	Describe("Exports", func() {
		It("Should add a function export", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{},
				Results: []wasm.ValueType{wasm.I32},
			}
			typeIdx := mod.AddType(ft)
			funcIdx := mod.AddFunction(typeIdx, []wasm.ValueType{}, []byte{})
			mod.AddExport("exported_func", wasm.ExportFunc, funcIdx)
			_, _, exports := mod.Debug()
			Expect(exports).To(Equal(1))
		})

		It("Should add multiple exports", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{},
				Results: []wasm.ValueType{},
			}
			typeIdx := mod.AddType(ft)
			func1 := mod.AddFunction(typeIdx, []wasm.ValueType{}, []byte{})
			func2 := mod.AddFunction(typeIdx, []wasm.ValueType{}, []byte{})
			mod.AddExport("func1", wasm.ExportFunc, func1)
			mod.AddExport("func2", wasm.ExportFunc, func2)
			_, _, exports := mod.Debug()
			Expect(exports).To(Equal(2))
		})

		It("Should support memory export", func() {
			mod := wasm.NewModule()
			mod.EnableMemory()
			mod.AddExport("memory", wasm.ExportMemory, 0)
			_, _, exports := mod.Debug()
			Expect(exports).To(Equal(1))
		})
	})

	Describe("Memory", func() {
		It("Should enable memory", func() {
			mod := wasm.NewModule()
			mod.EnableMemory()
			bytes := mod.Generate()
			// Memory section ID is 0x05 - verify it's present
			Expect(bytes).To(ContainElement(byte(0x05)))
		})

		It("Should generate valid module without memory", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{},
				Results: []wasm.ValueType{},
			}
			typeIdx := mod.AddType(ft)
			mod.AddFunction(typeIdx, []wasm.ValueType{}, []byte{})
			bytes := mod.Generate()
			// Should have magic, version, type section, function section, code section
			Expect(len(bytes)).To(BeNumerically(">", 8))
		})
	})

	Describe("Code Generation", func() {
		It("Should generate complete module with all sections", func() {
			mod := wasm.NewModule()

			// Add function type
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32, wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			typeIdx := mod.AddType(ft)

			// Add import
			mod.AddImport("env", "log", wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{},
			})

			// Add function
			body := []byte{
				0x20, 0x00, // local.get 0
				0x20, 0x01, // local.get 1
				0x6a, // i32.add
			}
			funcIdx := mod.AddFunction(typeIdx, []wasm.ValueType{}, body)

			// Add export
			mod.AddExport("add", wasm.ExportFunc, funcIdx)

			// Enable memory
			mod.EnableMemory()

			bytes := mod.Generate()

			// Verify magic and version
			Expect(bytes[0:4]).To(Equal([]byte{0x00, 0x61, 0x73, 0x6d}))
			Expect(bytes[4:8]).To(Equal([]byte{0x01, 0x00, 0x00, 0x00}))

			// Verify sections are present
			// Type section (0x01), Import (0x02), Function (0x03), Memory (0x05), Export (0x07), Code (0x0a)
			Expect(bytes).To(ContainElement(byte(0x01))) // Type section
			Expect(bytes).To(ContainElement(byte(0x02))) // Import section
			Expect(bytes).To(ContainElement(byte(0x03))) // Function section
			Expect(bytes).To(ContainElement(byte(0x05))) // Memory section
			Expect(bytes).To(ContainElement(byte(0x07))) // Export section
			Expect(bytes).To(ContainElement(byte(0x0a))) // Code section
		})

		It("Should group locals by type efficiently", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{},
				Results: []wasm.ValueType{},
			}
			typeIdx := mod.AddType(ft)

			// Add function with grouped locals: 3 i32s, 2 i64s, 1 f32
			locals := []wasm.ValueType{
				wasm.I32, wasm.I32, wasm.I32,
				wasm.I64, wasm.I64,
				wasm.F32,
			}
			mod.AddFunction(typeIdx, locals, []byte{})

			bytes := mod.Generate()
			// Verify module generates successfully
			Expect(bytes).ToNot(BeEmpty())
		})

		It("Should handle function with no locals", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			typeIdx := mod.AddType(ft)
			body := []byte{0x20, 0x00, 0x0b} // local.get 0, end
			mod.AddFunction(typeIdx, []wasm.ValueType{}, body)

			bytes := mod.Generate()
			Expect(bytes).ToNot(BeEmpty())
		})
	})

	Describe("Edge Cases", func() {
		It("Should handle module with only imports", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			mod.AddImport("env", "func", ft)

			bytes := mod.Generate()
			Expect(bytes).ToNot(BeEmpty())
			// Should have magic, version, type section, import section
			Expect(bytes[0:4]).To(Equal([]byte{0x00, 0x61, 0x73, 0x6d}))
		})

		It("Should handle module with only types", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{wasm.I32},
				Results: []wasm.ValueType{wasm.I32},
			}
			mod.AddType(ft)

			bytes := mod.Generate()
			// Should have magic, version, type section
			Expect(bytes[0:4]).To(Equal([]byte{0x00, 0x61, 0x73, 0x6d}))
		})

		It("Should handle multiple functions with same type", func() {
			mod := wasm.NewModule()
			ft := wasm.FunctionType{
				Params:  []wasm.ValueType{},
				Results: []wasm.ValueType{wasm.I32},
			}
			typeIdx := mod.AddType(ft)

			// Add 5 functions with same type
			for i := 0; i < 5; i++ {
				mod.AddFunction(typeIdx, []wasm.ValueType{}, []byte{0x41, 0x00})
			}

			types, functions, _ := mod.Debug()
			Expect(types).To(Equal(1))     // Should reuse type
			Expect(functions).To(Equal(5)) // Should have 5 functions
		})
	})
})
