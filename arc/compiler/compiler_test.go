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
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/compiler/bindings"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
)

func compile(source string, resolver symbol.Resolver) (compiler.Output, error) {
	prog := MustSucceed(text.Parse(text.Text{Raw: source}))
	inter, diag := text.Analyze(ctx, prog, resolver)
	Expect(diag.Ok()).To(BeTrue())
	return compiler.Compile(ctx, inter, compiler.DisableHostImport())
}

func compileWithHostImports(source string, resolver symbol.Resolver) (compiler.Output, error) {
	prog := MustSucceed(text.Parse(text.Text{Raw: source}))
	inter, diag := text.Analyze(ctx, prog, resolver)
	Expect(diag.Ok()).To(BeTrue())
	return compiler.Compile(ctx, inter)
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
			output := MustSucceed(compile(`
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
			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
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
			output := MustSucceed(compile(`
			func add(a i64, b i64) i64 {
				return a + b
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			add := mod.ExportedFunction("add")
			Expect(add).ToNot(BeNil())

			results := MustSucceed(add.Call(ctx, 10, 32))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(42)))
		})
	})

	Describe("Function with Config Execution", func() {
		It("Should execute a simple compiled addition function with config", func() {
			output := MustSucceed(compile(`
			func add{
				a i64
			} (b i64) i64 {
				return a + b
			}
			`, nil))
			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
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
			mockRuntime := bindings.NewBindings()

			// Setup channel data
			channelData := map[uint32]int32{0: 42}

			// Define channel read implementation
			mockRuntime.ChannelReadI32 = func(ctx context.Context, channelID uint32) int32 {
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
					Type: types.Chan(types.I32()),
				},
			})

			// Compile with host imports enabled
			output := MustSucceed(compileWithHostImports(`
			func readAndDouble() i32 {
				return sensor * 2
			}
			`, resolver))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			readAndDouble := mod.ExportedFunction("readAndDouble")
			Expect(readAndDouble).ToNot(BeNil())

			results := MustSucceed(readAndDouble.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(84))) // 42 * 2
		})
	})

	Describe("Named Output Routing", func() {
		It("Should compile a debug multi-param function", func() {
			output := MustSucceed(compile(`
			func debug(x i64, y i64) {
				out i64
			} {
				out = x + y
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			debug := mod.ExportedFunction("debug")
			Expect(debug).ToNot(BeNil())

			MustSucceed(debug.Call(ctx, 10, 3))

			mem := mod.Memory()
			dirtyFlags, ok := mem.ReadUint64Le(0x1000)
			Expect(ok).To(BeTrue())
			Expect(dirtyFlags).To(Equal(uint64(1)))

			outValue, ok := mem.ReadUint64Le(0x1008)
			Expect(ok).To(BeTrue())
			Expect(outValue).To(Equal(uint64(13))) // Should be 10 + 3
		})

		It("Should compile a basic multi-output function", func() {
			output := MustSucceed(compile(`
			func classifier(value i64) {
				high i64
				low i64
			} {
				if value > 50 {
					high = value
				} else {
					low = value
				}
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			classifier := mod.ExportedFunction("classifier")
			Expect(classifier).ToNot(BeNil())

			// Call with value > 50 - should set high output
			MustSucceed(classifier.Call(ctx, 75))

			// Read memory to check outputs
			// Memory layout: [dirty_flags:i64][high:i64][low:i64]
			mem := mod.Memory()
			dirtyFlags, ok := mem.ReadUint64Le(0x1000)
			Expect(ok).To(BeTrue())
			Expect(dirtyFlags).To(Equal(uint64(1))) // Bit 0 set (high output)

			highValue, ok := mem.ReadUint64Le(0x1008)
			Expect(ok).To(BeTrue())
			Expect(highValue).To(Equal(uint64(75)))

			// Call with value <= 50 - should set low output
			MustSucceed(classifier.Call(ctx, 25))

			dirtyFlags, ok = mem.ReadUint64Le(0x1000)
			Expect(ok).To(BeTrue())
			Expect(dirtyFlags).To(Equal(uint64(2))) // Bit 1 set (low output)

			lowValue, ok := mem.ReadUint64Le(0x1010)
			Expect(ok).To(BeTrue())
			Expect(lowValue).To(Equal(uint64(25)))
		})

		It("Should handle conditional output routing", func() {
			output := MustSucceed(compile(`
			func router(x i64, y i64) {
				sum i64
				diff i64
				both i64
			} {
				if x > y {
					diff = x - y
				} else if x < y {
					diff = y - x
				} else {
					both = x + y
				}
				sum = x + y
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			router := mod.ExportedFunction("router")
			Expect(router).ToNot(BeNil())

			// Test x > y case
			MustSucceed(router.Call(ctx, 10, 3))

			mem := mod.Memory()
			dirtyFlags, ok := mem.ReadUint64Le(0x1000)
			Expect(ok).To(BeTrue())
			Expect(dirtyFlags & 1).To(Equal(uint64(1))) // sum set
			Expect(dirtyFlags & 2).To(Equal(uint64(2))) // diff set
			Expect(dirtyFlags & 4).To(Equal(uint64(0))) // both not set

			sumValue, ok := mem.ReadUint64Le(0x1008)
			Expect(ok).To(BeTrue())
			Expect(sumValue).To(Equal(uint64(13)))

			diffValue, ok := mem.ReadUint64Le(0x1010)
			Expect(ok).To(BeTrue())
			Expect(diffValue).To(Equal(uint64(7)))

			// Test x == y case
			MustSucceed(router.Call(ctx, 5, 5))

			dirtyFlags, ok = mem.ReadUint64Le(0x1000)
			Expect(ok).To(BeTrue())
			Expect(dirtyFlags & 1).To(Equal(uint64(1))) // sum set
			Expect(dirtyFlags & 2).To(Equal(uint64(0))) // diff not set
			Expect(dirtyFlags & 4).To(Equal(uint64(4))) // both set

			bothValue, ok := mem.ReadUint64Le(0x1018)
			Expect(ok).To(BeTrue())
			Expect(bothValue).To(Equal(uint64(10)))
		})

		It("Should support mixed output types", func() {
			output := MustSucceed(compile(`
			func converter(value i64) {
				asFloat f64
				asInt i32
				original i64
			} {
				asFloat = 3.14
				asInt = i32(42)
				original = value
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			converter := mod.ExportedFunction("converter")
			Expect(converter).ToNot(BeNil())

			MustSucceed(converter.Call(ctx, 100))

			mem := mod.Memory()
			dirtyFlags, ok := mem.ReadUint64Le(0x1000)
			Expect(ok).To(BeTrue())
			Expect(dirtyFlags).To(Equal(uint64(7))) // All 3 outputs set (bits 0,1,2)

			// Memory layout: [dirty_flags:i64][asFloat:f64][asInt:i32][original:i64]
			floatValue, ok := mem.ReadFloat64Le(0x1008)
			Expect(ok).To(BeTrue())
			Expect(floatValue).To(BeNumerically("~", 3.14, 0.01))

			intValue, ok := mem.ReadUint32Le(0x1010)
			Expect(ok).To(BeTrue())
			Expect(intValue).To(Equal(uint32(42)))

			originalValue, ok := mem.ReadUint64Le(0x1014)
			Expect(ok).To(BeTrue())
			Expect(originalValue).To(Equal(uint64(100)))
		})

		It("Should support multi-output functions", func() {
			output := MustSucceed(compile(`
			func divmod(a i64, b i64) {
				quotient i64
				remainder i64
			} {
				quotient = a / b
				remainder = a % b
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			divmod := mod.ExportedFunction("divmod")
			Expect(divmod).ToNot(BeNil())

			MustSucceed(divmod.Call(ctx, 17, 5))

			mem := mod.Memory()
			dirtyFlags, ok := mem.ReadUint64Le(0x1000)
			Expect(ok).To(BeTrue())
			Expect(dirtyFlags).To(Equal(uint64(3))) // Both outputs set (bits 0,1)

			quotient, ok := mem.ReadUint64Le(0x1008)
			Expect(ok).To(BeTrue())
			Expect(quotient).To(Equal(uint64(3)))

			remainder, ok := mem.ReadUint64Le(0x1010)
			Expect(ok).To(BeTrue())
			Expect(remainder).To(Equal(uint64(2)))
		})
	})
})
