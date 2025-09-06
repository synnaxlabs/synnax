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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/compiler/runtime/mock"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
)

func compile(source string, resolver symbol.Resolver) ([]byte, error) {
	prog := MustSucceed(parser.Parse(source))
	analysis := analyzer.Analyze(prog, analyzer.Options{Resolver: resolver})
	Expect(analysis.Ok()).To(BeTrue())
	return compiler.Compile(compiler.Config{
		Program:            prog,
		Analysis:           &analysis,
		DisableHostImports: true,
	})
}

func compileWithHostImports(source string, resolver symbol.Resolver) ([]byte, error) {
	prog := MustSucceed(parser.Parse(source))
	analysis := analyzer.Analyze(prog, analyzer.Options{Resolver: resolver})
	Expect(analysis.Ok()).To(BeTrue())
	return compiler.Compile(compiler.Config{
		Program:            prog,
		Analysis:           &analysis,
		DisableHostImports: false,
	})
}

var _ = Describe("Compiler", func() {
	var r wazero.Runtime
	BeforeEach(func() {
		r = wazero.NewRuntime(ctx)
	})
	AfterEach(func() {
		Expect(r.Close(ctx)).To(Succeed())
	})
	Describe("Function Execution", func() {
		It("should execute a function with conditional returns", func() {
			wasmBytes := MustSucceed(compile(`
			func dog(b i64) i64 {
				a i64 := 2
				if b == a {
					c := 1
					return c
				} else if b > a {
					d := 2
					return d
				}
				return b
			}
			`, nil))
			mod := MustSucceed(r.Instantiate(ctx, wasmBytes))
			dog := mod.ExportedFunction("dog")
			Expect(dog).ToNot(BeNil())
			// Test case 1: b == 2 should return 1
			results := MustSucceed(dog.Call(ctx, 2))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(1)))

			// Test case 2: b > 2 (e.g., 5) should return 2
			results = MustSucceed(dog.Call(ctx, 5))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(2)))

			// Test case 3: b < 2 (e.g., 1) should return 0 (default/unspecified path)
			results = MustSucceed(dog.Call(ctx, 1))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(1)))
		})

		It("should execute a simple addition function", func() {
			wasmBytes := MustSucceed(compile(`
			func add(a i64, b i64) i64 {
				return a + b
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, wasmBytes))
			add := mod.ExportedFunction("add")
			Expect(add).ToNot(BeNil())

			results := MustSucceed(add.Call(ctx, 10, 32))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(42)))
		})
	})

	Describe("Task Execution", func() {
		It("Should execute a simple compiled addition task", func() {
			wasmBytes := MustSucceed(compile(`
			task add{
				a i64
			} (b i64) i64 {
				return a + b
			}
			`, nil))
			mod := MustSucceed(r.Instantiate(ctx, wasmBytes))
			add := mod.ExportedFunction("add")
			Expect(add).ToNot(BeNil())
			results := MustSucceed(add.Call(ctx, 10, 32))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(42)))
		})
	})

	Describe("Channel Operations", func() {
		It("Should execute a function with channel read operations", func() {
			// Create mock runtime with channel implementations
			mockRuntime := mock.New()

			// Setup channel data
			channelData := map[uint32]int32{0: 42}

			// Define channel read implementation
			mockRuntime.ChannelRead["i32"] = func(ctx context.Context, channelID uint32) int32 {
				if val, ok := channelData[channelID]; ok {
					return val
				}
				return 0
			}

			// Bind the mock runtime
			Expect(mockRuntime.Bind(ctx, r)).To(Succeed())

			resolver := symbol.MapResolver(map[string]symbol.Symbol{
				"sensor": {
					Name: "sensor",
					Kind: symbol.KindChannel,
					Type: types.Chan{ValueType: types.I32{}},
				},
			})

			// Compile with host imports enabled
			wasmBytes := MustSucceed(compileWithHostImports(`
			func readAndDouble() i32 {
				return sensor * 2
			}
			`, resolver))

			mod := MustSucceed(r.Instantiate(ctx, wasmBytes))
			readAndDouble := mod.ExportedFunction("readAndDouble")
			Expect(readAndDouble).ToNot(BeNil())

			results := MustSucceed(readAndDouble.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(84))) // 42 * 2
		})
	})

	Describe("Flow Expression", func() {
		It("Should correctly compile and execute a flow expression", func() {

			// Create mock runtime with channel implementations
			mockRuntime := mock.New()

			// Setup channel data
			channelData := map[uint32]int32{12: 32}

			// Define channel read implementation
			mockRuntime.ChannelRead["i32"] = func(ctx context.Context, channelID uint32) int32 {
				if val, ok := channelData[channelID]; ok {
					return val
				}
				return 0
			}

			// Bind the mock runtime
			Expect(mockRuntime.Bind(ctx, r)).To(Succeed())
			printType := types.NewTask()
			printType.Config.Put("message", types.String{})

			resolver := symbol.MapResolver{
				"ox_pt_1": symbol.Symbol{
					Name: "ox_pt_1",
					Kind: symbol.KindChannel,
					Type: types.Chan{ValueType: types.I32{}},
					ID:   12,
				},
				"print": symbol.Symbol{
					Name: "print",
					Kind: symbol.KindTask,
					Type: printType,
				},
			}

			// Compile with host imports enabled
			wasmBytes := MustSucceed(compileWithHostImports(`ox_pt_1 > 10 -> print{message: "dog"}`, resolver))

			mod := MustSucceed(r.Instantiate(ctx, wasmBytes))
			readAndDouble := mod.ExportedFunction("__expr_0")
			Expect(readAndDouble).ToNot(BeNil())

			results := MustSucceed(readAndDouble.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(1))) // 42 * 2
		})
	})
})
