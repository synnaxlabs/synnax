// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/parser"
	. "github.com/synnaxlabs/x/testutil"
)

func TestMainCompiler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Compiler Suite")
}

func compile(source string) (*compiler.Module, error) {
	prog := MustSucceed(parser.Parse(source))
	analysis := analyzer.Analyze(analyzer.Config{Program: prog})
	return compiler.Compile(prog, &analysis)
}

var _ = Describe("Main Compiler", func() {
	Describe("Function Compilation", func() {
		It("should compile a simple function", func() {
			source := `
				func add(a i32, b i32) i32 {
					return a + b
				}
			`
			mod := MustSucceed(compile(source))
			Expect(mod).ToNot(BeNil())
			Expect(mod.Functions).To(HaveLen(1))
			Expect(mod.Functions[0].Name).To(Equal("add"))
			Expect(mod.Functions[0].Params).To(HaveLen(2))
			Expect(*mod.Functions[0].Return).To(ContainSubstring("I32"))

			// Check WASM mod bytes were generated
			Expect(len(mod.Module)).To(BeNumerically(">", 0))

			// WASM magic number
			Expect(mod.Module[0:4]).To(Equal([]byte{0x00, 0x61, 0x73, 0x6d}))
		})

		It("should compile a function with local variables", func() {
			source := `
				func calculate(x i32) i32 {
					y := x * 2
					z := y + 10
					return z
				}
			`

			module, err := compile(source)
			Expect(err).To(BeNil())
			Expect(module).ToNot(BeNil())

			// Check function metadata
			Expect(module.Functions).To(HaveLen(1))
			Expect(module.Functions[0].Name).To(Equal("calculate"))

			// Check WASM module was generated
			Expect(len(module.Module)).To(BeNumerically(">", 0))
		})

		It("should compile a void function", func() {
			source := `
				func doNothing() {
					x := 42
					return
				}
			`

			module, err := compile(source)
			Expect(err).To(BeNil())
			Expect(module).ToNot(BeNil())

			// Check function has no return type
			Expect(module.Functions).To(HaveLen(1))
			Expect(module.Functions[0].Return).To(BeNil())
		})
	})

	Describe("Task Compilation", func() {
		It("should compile a simple task", func() {
			source := `
				task process(input chan i32, output chan i32) {
					value := <-input
					result := value * 2
					result -> output
				}
			`

			module, err := compile(source)
			Expect(err).To(BeNil())
			Expect(module).ToNot(BeNil())

			// Check task metadata
			Expect(module.Tasks).To(HaveLen(1))
			Expect(module.Tasks[0].Name).To(Equal("process"))
			Expect(module.Tasks[0].Args).To(HaveLen(2))

			// Check WASM module was generated
			Expect(len(module.Module)).To(BeNumerically(">", 0))
		})

		It("should compile a task with stateful variables", func() {
			source := `
				task counter(increment chan i32, output chan i32) {
					count $= 0
					inc := <-increment
					count = count + inc
					count -> output
				}
			`

			module, err := compile(source)
			Expect(err).To(BeNil())
			Expect(module).ToNot(BeNil())

			// Check stateful variables
			Expect(module.Tasks).To(HaveLen(1))
			Expect(module.Tasks[0].StatefulVars).To(HaveLen(1))
			Expect(module.Tasks[0].StatefulVars[0].Name).To(Equal("count"))
		})
	})

	Describe("Multiple Items", func() {
		It("should compile multiple functions and tasks", func() {
			source := `
				func helper(x i32) i32 {
					return x * 2
				}

				task processor(input chan i32, output chan i32) {
					value := <-input
					result := helper(value)
					result -> output
				}

				func add(a i32, b i32) i32 {
					return a + b
				}
			`

			module, err := compile(source)
			Expect(err).To(BeNil())
			Expect(module).ToNot(BeNil())

			// Check we have 2 functions and 1 task
			Expect(module.Functions).To(HaveLen(2))
			Expect(module.Tasks).To(HaveLen(1))

			// Check function names
			funcNames := []string{}
			for _, f := range module.Functions {
				funcNames = append(funcNames, f.Name)
			}
			Expect(funcNames).To(ConsistOf("helper", "add"))

			// Check task name
			Expect(module.Tasks[0].Name).To(Equal("processor"))
		})
	})

	Describe("Error Handling", func() {
		It("should fail compilation if analysis has errors", func() {
			source := `
				func broken() i32 {
					return undefined
				}
			`

			prog := MustSucceed(parser.Parse(source))
			analysis := analyzer.Analyze(analyzer.Config{
				Program: prog,
			})

			// This should have analysis errors for undefined variable
			module, err := compiler.Compile(prog, &analysis)
			Expect(err).ToNot(BeNil())
			Expect(module).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("cannot compile program with errors"))
		})
	})
})
