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

	Describe("Literal Type Inference", func() {
		It("Should compile integer literal with f32 variable", func() {
			output := MustSucceed(compile(`
			func add_two(x f32) f32 {
				return x + 2
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			add_two := mod.ExportedFunction("add_two")
			Expect(add_two).ToNot(BeNil())

			// Call with 3.5, expect 5.5
			results := MustSucceed(add_two.Call(ctx, uint64(0x40600000))) // 3.5 as f32 bits
			Expect(results).To(HaveLen(1))
			// Result should be 5.5 as f32
			Expect(results[0]).To(Equal(uint64(0x40b00000)))
		})

		It("Should compile decimal literal with i32 variable", func() {
			output := MustSucceed(compile(`
			func compare(x i32) u8 {
				return x > 5.0
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			compare := mod.ExportedFunction("compare")
			Expect(compare).ToNot(BeNil())

			// Test with 10, should return 1 (true)
			results := MustSucceed(compare.Call(ctx, 10))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(1)))

			// Test with 3, should return 0 (false)
			results = MustSucceed(compare.Call(ctx, 3))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(0)))
		})

		It("Should compile expression with multiple literals and f32 variable", func() {
			output := MustSucceed(compile(`
			func celsius_to_fahrenheit(celsius f32) f32 {
				return celsius * 1.8 + 32
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			celsius_to_fahrenheit := mod.ExportedFunction("celsius_to_fahrenheit")
			Expect(celsius_to_fahrenheit).ToNot(BeNil())

			// Convert 0°C to °F, should be 32°F
			results := MustSucceed(celsius_to_fahrenheit.Call(ctx, uint64(0x00000000))) // 0.0 as f32
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(0x42000000))) // 32.0 as f32

			// Convert 100°C to °F, should be 212°F
			results = MustSucceed(celsius_to_fahrenheit.Call(ctx, uint64(0x42c80000))) // 100.0 as f32
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(0x43540000))) // 212.0 as f32
		})

		It("Should compile literals in variable declarations", func() {
			output := MustSucceed(compile(`
			func calculate(base f32) f32 {
				multiplier f32 := 2.5
				offset f32 := 10
				return base * multiplier + offset
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			calculate := mod.ExportedFunction("calculate")
			Expect(calculate).ToNot(BeNil())

			// calculate(4.0) = 4.0 * 2.5 + 10 = 20.0
			results := MustSucceed(calculate.Call(ctx, uint64(0x40800000))) // 4.0 as f32
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(0x41a00000))) // 20.0 as f32
		})

		It("Should compile literals with i64 variables", func() {
			output := MustSucceed(compile(`
			func increment(x i64) i64 {
				return x + 1
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			increment := mod.ExportedFunction("increment")
			Expect(increment).ToNot(BeNil())

			results := MustSucceed(increment.Call(ctx, 41))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(42)))
		})

		It("Should compile complex arithmetic with mixed literal types", func() {
			output := MustSucceed(compile(`
			func calculate(a f64, b f64) f64 {
				result f64 := 0
				result = a * 2 + b * 3.5 - 10
				return result
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			calculate := mod.ExportedFunction("calculate")
			Expect(calculate).ToNot(BeNil())

			// calculate(5.0, 2.0) = 5*2 + 2*3.5 - 10 = 10 + 7 - 10 = 7
			results := MustSucceed(calculate.Call(ctx,
				uint64(0x4014000000000000), // 5.0 as f64
				uint64(0x4000000000000000), // 2.0 as f64
			))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(0x401c000000000000))) // 7.0 as f64
		})

		It("Should compile literals in return statements", func() {
			output := MustSucceed(compile(`
			func get_constant() f32 {
				return 3.14159
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			get_constant := mod.ExportedFunction("get_constant")
			Expect(get_constant).ToNot(BeNil())

			results := MustSucceed(get_constant.Call(ctx))
			Expect(results).To(HaveLen(1))
			// Should be approximately 3.14159 as f32
			Expect(results[0]).To(Equal(uint64(0x40490fd0)))
		})

		It("Should compile literals with i32 variables in assignments", func() {
			output := MustSucceed(compile(`
			func process(x i32) i32 {
				result i32 := 0
				result = x + 5
				result = result * 2
				return result
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			process := mod.ExportedFunction("process")
			Expect(process).ToNot(BeNil())

			// process(3) = (3 + 5) * 2 = 16
			results := MustSucceed(process.Call(ctx, 3))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(16)))
		})

		It("Should default integer literals to i64 when unconstrained", func() {
			output := MustSucceed(compile(`
			func get_answer() i64 {
				x := 42
				return x
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			get_answer := mod.ExportedFunction("get_answer")
			Expect(get_answer).ToNot(BeNil())

			results := MustSucceed(get_answer.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(42)))
		})

		It("Should default float literals to f64 when unconstrained", func() {
			output := MustSucceed(compile(`
			func get_pi() f64 {
				x := 3.14
				return x
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			get_pi := mod.ExportedFunction("get_pi")
			Expect(get_pi).ToNot(BeNil())

			results := MustSucceed(get_pi.Call(ctx))
			Expect(results).To(HaveLen(1))
			// 3.14 as f64 bits
			Expect(results[0]).To(Equal(uint64(0x40091eb851eb851f)))
		})

		It("Should allow float literals in comparisons with i64", func() {
			output := MustSucceed(compile(`
			func is_positive(x i64) u8 {
				return x > 0.0
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			is_positive := mod.ExportedFunction("is_positive")
			Expect(is_positive).ToNot(BeNil())

			// Test positive value
			results := MustSucceed(is_positive.Call(ctx, 50))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(1)))

			// Test zero
			results = MustSucceed(is_positive.Call(ctx, 0))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(0)))

			// Test another positive
			results = MustSucceed(is_positive.Call(ctx, 100))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(1)))
		})

		It("Should allow mixed f32 and integer literal arithmetic", func() {
			output := MustSucceed(compile(`
			func scale_and_offset(value f32) f32 {
				scale f32 := 2
				offset f32 := 10
				return value * scale + offset
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			scale_and_offset := mod.ExportedFunction("scale_and_offset")
			Expect(scale_and_offset).ToNot(BeNil())

			// scale_and_offset(5.0) = 5.0 * 2 + 10 = 20.0
			results := MustSucceed(scale_and_offset.Call(ctx, uint64(0x40a00000))) // 5.0 as f32
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(0x41a00000))) // 20.0 as f32
		})

		It("Should execute complex literal inference with nested operations", func() {
			output := MustSucceed(compile(`
			func calculate(a i32, b i32) i32 {
				threshold i32 := 10
				multiplier i32 := 2
				if a > threshold {
					return a * multiplier + b
				}
				return b
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			calculate := mod.ExportedFunction("calculate")
			Expect(calculate).ToNot(BeNil())

			// Test a > threshold: calculate(15, 5) = 15 * 2 + 5 = 35
			results := MustSucceed(calculate.Call(ctx, 15, 5))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(35)))

			// Test a <= threshold: calculate(8, 5) = 5
			results = MustSucceed(calculate.Call(ctx, 8, 5))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(5)))
		})
	})
})
