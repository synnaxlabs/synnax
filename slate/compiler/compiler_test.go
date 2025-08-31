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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/resolver"
)

var _ = Describe("Compiler", func() {
	var (
		c *compiler.Compiler
		r resolver.GlobalResolver
	)

	BeforeEach(func() {
		r = resolver.NewStubResolver()
		c = compiler.New(r)
	})

	Describe("Basic Compilation", func() {
		It("should compile an empty program", func() {
			source := ""
			module, err := c.Compile(source)
			Expect(err).ToNot(HaveOccurred())
			Expect(module).ToNot(BeNil())
			Expect(module.Version).To(Equal("1.0.0"))
			Expect(module.Module).ToNot(BeEmpty())
		})

		It("should compile a simple task", func() {
			source := `
task double(value f64) f64 {
	return value * 2
}
`
			module, err := c.Compile(source)
			Expect(err).ToNot(HaveOccurred())
			Expect(module).ToNot(BeNil())
			Expect(module.Tasks).To(HaveLen(1))
			Expect(*module.Tasks[0].Name).To(Equal("double"))
			Expect(module.Tasks[0].Key).To(Equal("task_double"))
			Expect(module.Tasks[0].Args).To(HaveLen(1))
			Expect(module.Tasks[0].Args[0].Name).To(Equal("value"))
			Expect(module.Tasks[0].Args[0].Type).To(Equal("f64"))
			Expect(*module.Tasks[0].Return).To(Equal("f64"))
		})

		It("should compile a task with stateful variables", func() {
			source := `
task accumulator(value f64) f64 {
	let sum $= 0.0
	sum = sum + value
	return sum
}
`
			module, err := c.Compile(source)
			Expect(err).ToNot(HaveOccurred())
			Expect(module).ToNot(BeNil())
			Expect(module.Tasks).To(HaveLen(1))
			Expect(module.Tasks[0].StatefulVars).To(HaveLen(1))
			Expect(module.Tasks[0].StatefulVars[0].Name).To(Equal("sum"))
			Expect(module.Tasks[0].StatefulVars[0].Type).To(Equal("f64"))
			Expect(module.Tasks[0].StatefulVars[0].Key).To(Equal(0))
		})

		It("should compile a function", func() {
			source := `
func add(a f64, b f64) f64 {
	return a + b
}
`
			module, err := c.Compile(source)
			Expect(err).ToNot(HaveOccurred())
			Expect(module).ToNot(BeNil())
			Expect(module.Functions).To(HaveLen(1))
			Expect(module.Functions[0].Name).To(Equal("add"))
			Expect(module.Functions[0].Key).To(Equal("func_add"))
			Expect(module.Functions[0].Params).To(HaveLen(2))
			Expect(*module.Functions[0].Return).To(Equal("f64"))
		})

		It("should compile a flow statement", func() {
			source := `
sensor -> double[value=2] -> output
`
			module, err := c.Compile(source)
			Expect(err).ToNot(HaveOccurred())
			Expect(module).ToNot(BeNil())
			// Flow should create nodes and edges
			Expect(module.Nodes).ToNot(BeEmpty())
			Expect(module.Edges).ToNot(BeEmpty())
		})

		It("should handle typed channel operations", func() {
			source := `
task reader() f64 {
	let value = <- sensor
	return value
}
`
			module, err := c.Compile(source)
			Expect(err).ToNot(HaveOccurred())
			Expect(module).ToNot(BeNil())
			// Should compile with typed channel read
			Expect(module.Tasks).To(HaveLen(1))
		})

		It("should serialize to JSON", func() {
			source := `
task identity(x f64) f64 {
	return x
}
`
			module, err := c.Compile(source)
			Expect(err).ToNot(HaveOccurred())
			
			// Should be able to marshal to JSON
			jsonBytes, err := json.Marshal(module)
			Expect(err).ToNot(HaveOccurred())
			Expect(jsonBytes).ToNot(BeEmpty())
			
			// Should be able to unmarshal from JSON
			var decoded analyzer.Module
			err = json.Unmarshal(jsonBytes, &decoded)
			Expect(err).ToNot(HaveOccurred())
			Expect(decoded.Tasks).To(HaveLen(1))
		})
	})

	Describe("Error Handling", func() {
		It("should return error for invalid syntax", func() {
			source := `task broken(`
			_, err := c.Compile(source)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parse error"))
		})

		It("should return error for semantic errors", func() {
			source := `
task test() f64 {
	return undefined_var
}
`
			_, err := c.Compile(source)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("analysis error"))
		})
	})
})