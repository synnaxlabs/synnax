// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package module_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/module"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	. "github.com/synnaxlabs/x/testutil"
)

func assemble(source string, resolver symbol.Resolver) (*module.Module, error) {
	prog, err := parser.Parse(source)
	Expect(err).To(BeNil())
	result := analyzer.Analyze(prog, analyzer.Options{Resolver: resolver})
	Expect(result.Diagnostics).To(BeEmpty())
	wasmCode := MustSucceed(compiler.Compile(compiler.Config{Program: prog, Analysis: &result}))
	return module.Assemble(prog, result, wasmCode)
}

var _ = Describe("Graph Assembly", func() {
	Describe("Basic Flow Assembly", func() {
		It("Should assemble a basic task to task flow", func() {
			source := `
			task first{
				value f64
			} () f64 {
				return 1.0
			}
			task second{
				value f64
			} () {
			}
			first{value: 10.0} -> second{value: 20.0}
			`
			mod := MustSucceed(assemble(source, nil))
			Expect(mod.Tasks).To(HaveLen(2))
			firstTask := mod.Tasks[0]
			Expect(firstTask.Key).To(Equal("first"))
			Expect(firstTask.Returns).To(Equal("f64"))
			Expect(firstTask.Params).To(HaveLen(0))
			Expect(firstTask.Config).To(HaveLen(1))
			Expect(firstTask.Config["value"]).To(Equal("f64"))
			secondTask := mod.Tasks[1]
			Expect(secondTask.Key).To(Equal("second"))
			Expect(secondTask.Returns).To(Equal(""))
			Expect(secondTask.Params).To(HaveLen(0))
			Expect(secondTask.Config).To(HaveLen(1))
			Expect(secondTask.Config["value"]).To(Equal("f64"))

			Expect(mod.Nodes).To(HaveLen(2))
			firstNode := mod.Nodes[0]
			if firstNode.Key == "first_1" {
				Expect(firstNode.Key).To(Equal("first_1"))
				Expect(firstNode.Config["value"]).To(Equal("10.0"))
			}
			secondNode := mod.Nodes[1]
			if secondNode.Key == "second_1" {
				Expect(secondNode.Key).To(Equal("second_2"))
				Expect(secondNode.Config["value"]).To(Equal("20.0"))
			}
			Expect(mod.Edges).To(HaveLen(1))
		})

		It("Should assemble a basic channel to expression flow", func() {
			resolver := symbol.MapResolver{
				"ox_pt_1": symbol.Symbol{
					Name: "ox_pt_1",
					Type: types.Chan{ValueType: types.F64{}},
				},
			}
			source := `ox_pt_1 -> ox_pt_1 > 0`
			mod := MustSucceed(assemble(source, resolver))
			Expect(mod.Tasks).To(HaveLen(1))
		})

		It("Should correctly add channels to task", func() {
			source := `
			task first{
				value f64
			} () f64 {
				return ox_pt_1 + 1.0
			}
			task second{
				value f64
			} () {
			}
			first{value: 10.0} -> second{value: 20.0}
			`
			resolver := symbol.MapResolver{
				"ox_pt_1": symbol.Symbol{
					Name: "ox_pt_1",
					Kind: symbol.KindChannel,
					Type: types.Chan{ValueType: types.F64{}},
				},
			}
			mod := MustSucceed(assemble(source, resolver))
			Expect(mod.Tasks).To(HaveLen(2))
			firstTask := mod.Tasks[0]
			Expect(firstTask.Key).To(Equal("first"))
			Expect(firstTask.Returns).To(Equal("f64"))
			Expect(firstTask.Channels.Read).To(ContainElement("ox_pt_1"))
		})
	})
})
