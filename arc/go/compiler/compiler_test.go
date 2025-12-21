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
	"fmt"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/compiler/bindings"
	runtimebindings "github.com/synnaxlabs/arc/runtime/wasm/bindings"
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

		It("Should compile a function with an else statement", func() {
			output := MustSucceed(compile(`
			func add(a i64, b i64) i64 {
				if a > 0 {
					return a
				} else {
					return b
				}
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			add := mod.ExportedFunction("add")
			Expect(add).ToNot(BeNil())
			results := MustSucceed(add.Call(ctx, 10, 32))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(10)))
		})

		It("Should compile nested if-else where all branches return", func() {
			output := MustSucceed(compile(`
			func nested(a i64, b i64) i64 {
				if a > 0 {
					if b > 0 {
						return a + b
					} else {
						return a
					}
				} else {
					if b > 0 {
						return b
					} else {
						return 0
					}
				}
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			nested := mod.ExportedFunction("nested")
			Expect(nested).ToNot(BeNil())

			// Test a > 0, b > 0
			results := MustSucceed(nested.Call(ctx, 10, 5))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(15)))

			// Test a > 0, b <= 0
			results = MustSucceed(nested.Call(ctx, 10, 0))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(10)))

			// Test a <= 0, b > 0
			results = MustSucceed(nested.Call(ctx, 0, 5))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(5)))

			// Test a <= 0, b <= 0
			results = MustSucceed(nested.Call(ctx, 0, 0))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(0)))
		})

		It("Should compile if-else where only some branches return", func() {
			output := MustSucceed(compile(`
			func partial(a i64, b i64) i64 {
				x i64 := 0
				if a > 0 {
					if b > 0 {
						return a + b
					}
					x = a
				} else {
					x = b
				}
				return x
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			partial := mod.ExportedFunction("partial")
			Expect(partial).ToNot(BeNil())

			// Test early return
			results := MustSucceed(partial.Call(ctx, 10, 5))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(15)))

			// Test fall-through with a > 0, b <= 0
			results = MustSucceed(partial.Call(ctx, 10, 0))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(10)))

			// Test fall-through with a <= 0
			results = MustSucceed(partial.Call(ctx, 0, 7))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(7)))
		})

		It("Should compile deeply nested if-else with all returns", func() {
			output := MustSucceed(compile(`
			func deep(a i64, b i64, c i64) i64 {
				if a > 0 {
					if b > 0 {
						if c > 0 {
							return a + b + c
						} else {
							return a + b
						}
					} else {
						return a
					}
				} else {
					return 0
				}
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			deep := mod.ExportedFunction("deep")
			Expect(deep).ToNot(BeNil())

			// Test all positive
			results := MustSucceed(deep.Call(ctx, 1, 2, 3))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(6)))

			// Test c <= 0
			results = MustSucceed(deep.Call(ctx, 1, 2, 0))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(3)))

			// Test b <= 0
			results = MustSucceed(deep.Call(ctx, 1, 0, 3))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(1)))

			// Test a <= 0
			results = MustSucceed(deep.Call(ctx, 0, 2, 3))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(0)))
		})

		It("Should compile mixed nested returns with variables", func() {
			output := MustSucceed(compile(`
			func mixed(a i64, b i64) i64 {
				result i64 := 0
				if a > 10 {
					if b > 10 {
						return a * b
					} else {
						result = a
					}
				} else {
					if b > 10 {
						result = b
					} else {
						result = a + b
					}
				}
				return result + 1
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			mixed := mod.ExportedFunction("mixed")
			Expect(mixed).ToNot(BeNil())

			// Test early return
			results := MustSucceed(mixed.Call(ctx, 20, 30))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(600)))

			// Test a > 10, b <= 10
			results = MustSucceed(mixed.Call(ctx, 20, 5))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(21)))

			// Test a <= 10, b > 10
			results = MustSucceed(mixed.Call(ctx, 5, 20))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(21)))

			// Test a <= 10, b <= 10
			results = MustSucceed(mixed.Call(ctx, 5, 3))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(9)))
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
			func debug(x i64, y i64) (out i64) {
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
			func classifier(value i64) (high i64, low i64) {
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
			func router(x i64, y i64) (sum i64,	diff i64, both i64) {
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
			func converter(value i64) (asFloat f64,	asInt i32, original i64) {
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
			func divMod(a i64, b i64) (quotient i64, remainder i64) {
				quotient = a / b
				remainder = a % b
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			divMod := mod.ExportedFunction("divMod")
			Expect(divMod).ToNot(BeNil())
			MustSucceed(divMod.Call(ctx, 17, 5))

			mem := mod.Memory()
			dirtyFlags := MustBeOk(mem.ReadUint64Le(0x1000))
			Expect(dirtyFlags).To(Equal(uint64(3))) // Both outputs set (bits 0,1)
			quotient := MustBeOk(mem.ReadUint64Le(0x1008))
			Expect(quotient).To(Equal(uint64(3)))
			remainder := MustBeOk(mem.ReadUint64Le(0x1010))
			Expect(remainder).To(Equal(uint64(2)))
		})
	})

	Describe("Literal Type Inference", func() {
		It("Should compile integer literal with f32 variable", func() {
			output := MustSucceed(compile(`
			func addTwo(x f32) f32 {
				return x + 2
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			addTwo := mod.ExportedFunction("addTwo")
			Expect(addTwo).ToNot(BeNil())

			// Call with 3.5, expect 5.5
			results := MustSucceed(addTwo.Call(ctx, uint64(math.Float32bits(3.5))))
			Expect(results).To(HaveLen(1))
			// Result should be 5.5 as f32
			Expect(results[0]).To(Equal(uint64(math.Float32bits(5.5))))
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
			func celsiusToFahrenheit(celsius f32) f32 {
				return celsius * 1.8 + 32
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			celsiusToFahrenheit := mod.ExportedFunction("celsiusToFahrenheit")
			Expect(celsiusToFahrenheit).ToNot(BeNil())

			// Convert 0°C to °F, should be 32°F
			results := MustSucceed(celsiusToFahrenheit.Call(ctx, uint64(math.Float32bits(0.0))))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(math.Float32bits(32.0))))

			// Convert 100°C to °F, should be 212°F
			results = MustSucceed(celsiusToFahrenheit.Call(ctx, uint64(math.Float32bits(100.0))))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(math.Float32bits(212.0))))
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
			results := MustSucceed(calculate.Call(ctx, uint64(math.Float32bits(4.0))))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(math.Float32bits(20.0))))
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
				math.Float64bits(5.0),
				math.Float64bits(2.0),
			))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(math.Float64bits(7.0)))
		})

		It("Should compile literals in return statements", func() {
			output := MustSucceed(compile(`
			func getConstant() f32 {
				return 3.14159
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			getConstant := mod.ExportedFunction("getConstant")
			Expect(getConstant).ToNot(BeNil())

			results := MustSucceed(getConstant.Call(ctx))
			Expect(results).To(HaveLen(1))
			// Should be approximately 3.14159 as f32
			Expect(results[0]).To(Equal(uint64(math.Float32bits(3.14159))))
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
			func getAnswer() i64 {
				x := 42
				return x
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			getAnswer := mod.ExportedFunction("getAnswer")
			Expect(getAnswer).ToNot(BeNil())

			results := MustSucceed(getAnswer.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(42)))
		})

		It("Should default float literals to f64 when unconstrained", func() {
			output := MustSucceed(compile(`
			func getPi() f64 {
				x := 3.14
				return x
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			getPi := mod.ExportedFunction("getPi")
			Expect(getPi).ToNot(BeNil())

			results := MustSucceed(getPi.Call(ctx))
			Expect(results).To(HaveLen(1))
			// 3.14 as f64 bits
			Expect(results[0]).To(Equal(math.Float64bits(3.14)))
		})

		It("Should allow float literals in comparisons with i64", func() {
			output := MustSucceed(compile(`
			func isPositive(x i64) u8 {
				return x > 0.0
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			isPositive := mod.ExportedFunction("isPositive")
			Expect(isPositive).ToNot(BeNil())

			// Test positive value
			results := MustSucceed(isPositive.Call(ctx, 50))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(1)))

			// Test zero
			results = MustSucceed(isPositive.Call(ctx, 0))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(0)))

			// Test another positive
			results = MustSucceed(isPositive.Call(ctx, 100))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(1)))
		})

		It("Should allow mixed f32 and integer literal arithmetic", func() {
			output := MustSucceed(compile(`
			func scaleAndOffset(value f32) f32 {
				scale f32 := 2
				offset f32 := 10
				return value * scale + offset
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			scaleAndOffset := mod.ExportedFunction("scaleAndOffset")
			Expect(scaleAndOffset).ToNot(BeNil())

			// scaleAndOffset(5.0) = 5.0 * 2 + 10 = 20.0
			results := MustSucceed(scaleAndOffset.Call(ctx, uint64(math.Float32bits(5.0))))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(math.Float32bits(20.0))))
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

		It("Should correctly execute signed comparison with negative numbers", func() {
			output := MustSucceed(compile(`
			func test(a i32) u8 {
				return a > -10
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			test := mod.ExportedFunction("test")
			Expect(test).ToNot(BeNil())

			// Test: -5 > -10 should be true (signed comparison)
			negFive := int32(-5)
			results := MustSucceed(test.Call(ctx, uint64(uint32(negFive))))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(1))) // true

			// Test: -15 > -10 should be false
			negFifteen := int32(-15)
			results = MustSucceed(test.Call(ctx, uint64(uint32(negFifteen))))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(0))) // false
		})

		It("Should infer f32 from conditional return with integer constant and f32 value", func() {
			// This tests the regression for SY-3195
			// Integer constant (0) should be coerced to f32 when mixed with f32 returns
			output := MustSucceed(compile(`
			func conditionalReturn(condition u8, value f32) f32 {
				if (condition == 1) {
					return 0    // Integer constant
				}
				return value   // F32 value
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			conditionalReturn := mod.ExportedFunction("conditionalReturn")
			Expect(conditionalReturn).ToNot(BeNil())

			// Test case 1: condition == 1, should return 0.0 (as f32)
			results := MustSucceed(conditionalReturn.Call(ctx, 1, uint64(math.Float32bits(42.5))))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(math.Float32bits(0.0))))

			// Test case 2: condition != 1, should return the f32 value (42.5)
			results = MustSucceed(conditionalReturn.Call(ctx, 0, uint64(math.Float32bits(42.5))))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(math.Float32bits(42.5))))
		})
	})

	DescribeTable("PEMDAS", func(expr string, expected float64) {
		output := MustSucceed(compile(fmt.Sprintf(`
			func dog(b i64) f64 {
				return %s
			}`, expr), nil))
		mod := MustSucceed(r.Instantiate(ctx, output.WASM))
		dog := mod.ExportedFunction("dog")
		Expect(dog).ToNot(BeNil())
		results := MustSucceed(dog.Call(ctx, 2))
		Expect(results).To(HaveLen(1))
		res := results[0]
		Expect(math.Float64frombits(res)).To(Equal(expected))
	},
		// Basic operations
		Entry("Addition", "1.0 + 2.0", 3.0),
		Entry("Subtraction", "5.0 - 3.0", 2.0),
		Entry("Multiplication", "4.0 * 5.0", 20.0),
		Entry("Division", "10.0 / 2.0", 5.0),

		// Multiplication before Addition
		Entry("Multiplication before Addition: 2 + 3 * 4",
			"2.0 + 3.0 * 4.0",
			14.0,
		),
		Entry("Multiplication before Addition: 3 * 4 + 2",
			"3.0 * 4.0 + 2.0",
			14.0,
		),
		Entry("Multiple Multiplications with Addition: 2 * 3 + 4 * 5",
			"2.0 * 3.0 + 4.0 * 5.0",
			26.0,
		),
		Entry("Division before Addition", "10.0 / 2.0 + 3.0", 8.0),
		Entry("Division before Addition", "3.0 + 10.0 / 2.0", 8.0),

		// Multiplication before Subtraction
		Entry("Multiplication before Subtraction: 10 - 2 * 3",
			"10.0 - 2.0 * 3.0",
			4.0, // 10 - 6 = 4
		),
		Entry("Multiplication before Subtraction: 2 * 3 - 4",
			"2.0 * 3.0 - 4.0",
			2.0, // 6 - 4 = 2
		),

		// Division before Subtraction
		Entry("Division before Subtraction: 20 - 10 / 2",
			"20.0 - 10.0 / 2.0",
			15.0, // 20 - 5 = 15
		),
		Entry("Division before Subtraction: 10 / 2 - 3",
			"10.0 / 2.0 - 3.0",
			2.0, // 5 - 3 = 2
		),

		// Left-to-Right Associativity: Addition/Subtraction
		Entry("Left-to-Right Subtraction: 10 - 3 - 2",
			"10.0 - 3.0 - 2.0",
			5.0, // (10 - 3) - 2 = 7 - 2 = 5
		),
		Entry("Left-to-Right Addition: 1 + 2 + 3 + 4",
			"1.0 + 2.0 + 3.0 + 4.0",
			10.0, // ((1 + 2) + 3) + 4 = 10
		),
		Entry("Mixed Addition/Subtraction: 10 + 5 - 3 + 2",
			"10.0 + 5.0 - 3.0 + 2.0",
			14.0, // ((10 + 5) - 3) + 2 = 14
		),

		// Left-to-Right Associativity: Multiplication/Division
		Entry("Left-to-Right Division: 20 / 4 / 2",
			"20.0 / 4.0 / 2.0",
			2.5, // (20 / 4) / 2 = 5 / 2 = 2.5
		),
		Entry("Left-to-Right Multiplication: 2 * 3 * 4",
			"2.0 * 3.0 * 4.0",
			24.0, // (2 * 3) * 4 = 6 * 4 = 24
		),
		Entry("Mixed Multiplication/Division: 100 / 2 * 5 / 10",
			"100.0 / 2.0 * 5.0 / 10.0",
			25.0, // ((100 / 2) * 5) / 10 = (50 * 5) / 10 = 250 / 10 = 25
		),
		Entry("Division then Multiplication (original bug): 1 / 2 * 500",
			"1.0 / 2.0 * 500.0",
			250.0, // (1 / 2) * 500 = 0.5 * 500 = 250
		),

		// Parentheses Override
		Entry("Parentheses force addition before multiplication: (2 + 3) * 4",
			"(2.0 + 3.0) * 4.0",
			20.0, // 5 * 4 = 20
		),
		Entry("Parentheses force addition before division: 20 / (2 + 3)",
			"20.0 / (2.0 + 3.0)",
			4.0, // 20 / 5 = 4
		),
		Entry("Nested Parentheses: ((2 + 3) * 4) - 5",
			"((2.0 + 3.0) * 4.0) - 5.0",
			15.0, // (5 * 4) - 5 = 20 - 5 = 15
		),
		Entry("Multiple Parentheses Groups: (2 + 3) * (4 + 5)",
			"(2.0 + 3.0) * (4.0 + 5.0)",
			45.0, // 5 * 9 = 45
		),

		// Complex Mixed Operations
		Entry("Complex: 2 + 3 * 4 - 5",
			"2.0 + 3.0 * 4.0 - 5.0",
			9.0, // 2 + 12 - 5 = 9
		),
		Entry("Complex: 10 / 2 + 3 * 4 - 1",
			"10.0 / 2.0 + 3.0 * 4.0 - 1.0",
			16.0, // 5 + 12 - 1 = 16
		),
		Entry("Complex: 100 - 20 / 4 + 3 * 2",
			"100.0 - 20.0 / 4.0 + 3.0 * 2.0",
			101.0, // 100 - 5 + 6 = 101
		),
		Entry("Complex: 50 / 10 * 2 + 8 - 3",
			"50.0 / 10.0 * 2.0 + 8.0 - 3.0",
			15.0, // ((50 / 10) * 2) + 8 - 3 = (5 * 2) + 8 - 3 = 10 + 8 - 3 = 15
		),

		// Edge Cases
		Entry("All Additions: 1 + 2 + 3 + 4 + 5",
			"1.0 + 2.0 + 3.0 + 4.0 + 5.0",
			15.0,
		),
		Entry("All Multiplications: 2 * 3 * 4",
			"2.0 * 3.0 * 4.0",
			24.0,
		),
		Entry("Subtraction Chain: 100 - 10 - 5 - 3",
			"100.0 - 10.0 - 5.0 - 3.0",
			82.0, // ((100 - 10) - 5) - 3 = 82
		),
		Entry("Division Chain: 1000 / 10 / 5 / 2",
			"1000.0 / 10.0 / 5.0 / 2.0",
			10.0, // ((1000 / 10) / 5) / 2 = (100 / 5) / 2 = 20 / 2 = 10
		),

		// Realistic Calculations
		Entry("Average: (10 + 20 + 30) / 3",
			"(10.0 + 20.0 + 30.0) / 3.0",
			20.0,
		),
		Entry("Percentage: 200 * 15 / 100",
			"200.0 * 15.0 / 100.0",
			30.0, // (200 * 15) / 100 = 3000 / 100 = 30
		),
		Entry("Temperature Conversion Formula-like: 9 / 5 * 100 + 32",
			"9.0 / 5.0 * 100.0 + 32.0",
			212.0, // ((9 / 5) * 100) + 32 = (1.8 * 100) + 32 = 180 + 32 = 212
		),
	)

	Describe("Power Expression Execution", func() {
		BeforeEach(func() {
			// Setup bindings for math operations with actual runtime implementations
			b := bindings.NewBindings()
			// Note: Math functions don't require state, so we pass nil
			arcRuntime := runtimebindings.NewRuntime(nil, nil)
			runtimebindings.BindRuntime(arcRuntime, b)
			Expect(b.Bind(ctx, r)).To(Succeed())
		})

		It("Should execute i32 power: 2^3 = 8", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() i32 {
				return i32(2) ^ i32(3)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(8)))
		})

		It("Should execute i64 power: 2^10 = 1024", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() i64 {
				return 2 ^ 10
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(1024)))
		})

		It("Should execute u32 power: 3^4 = 81", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() u32 {
				return u32(3) ^ u32(4)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(81)))
		})

		It("Should execute u64 power: 5^3 = 125", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() u64 {
				return u64(5) ^ u64(3)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(125)))
		})

		It("Should execute f32 power: 2.0^3.0 = 8.0", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() f32 {
				return f32(2.0) ^ f32(3.0)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(math.Float32bits(8.0))))
		})

		It("Should execute f64 power: 2.5^2.0 = 6.25", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() f64 {
				return 2.5 ^ 2.0
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(math.Float64bits(6.25)))
		})

		It("Should execute right-associative power: 2^3^2 = 2^(3^2) = 2^9 = 512", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() i32 {
				return i32(2) ^ i32(3) ^ i32(2)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(512)))
		})

		It("Should execute power with higher precedence than addition: 2 + 3^2 = 2 + 9 = 11", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() i32 {
				return i32(2) + i32(3) ^ i32(2)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(11)))
		})

		It("Should execute power with parentheses: (2 + 3)^2 = 5^2 = 25", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() i32 {
				return (i32(2) + i32(3)) ^ i32(2)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(25)))
		})

		It("Should execute power with multiplication: 2 * 3^2 = 2 * 9 = 18", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() i32 {
				return i32(2) * i32(3) ^ i32(2)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(18)))
		})

		It("Should execute power with variable base and exponent", func() {
			output := MustSucceed(compileWithHostImports(`
			func power(base i32, exp i32) i32 {
				return base ^ exp
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			// Test 4^3 = 64
			results := MustSucceed(power.Call(ctx, 4, 3))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(64)))

			// Test 10^2 = 100
			results = MustSucceed(power.Call(ctx, 10, 2))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(100)))
		})

		It("Should execute power with zero exponent: 5^0 = 1", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() i32 {
				return i32(5) ^ i32(0)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(1)))
		})

		It("Should execute power with exponent one: 42^1 = 42", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() i32 {
				return i32(42) ^ i32(1)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			Expect(results[0]).To(Equal(uint64(42)))
		})

		It("Should execute negative base with even exponent: (-2)^4 = 16", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() i32 {
				return i32(-2) ^ i32(4)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			// -2^4 = 16 (even exponent, positive result)
			Expect(results[0]).To(Equal(uint64(16)))
		})

		It("Should execute negative base with odd exponent: (-2)^3 = -8", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() i32 {
				return i32(-2) ^ i32(3)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			// -2^3 = -8 (odd exponent, negative result)
			negEight := int32(-8)
			Expect(results[0]).To(Equal(uint64(uint32(negEight))))
		})

		It("Should execute fractional f64 power: 27.0^(1.0/3.0) ≈ 3.0", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() f64 {
				return 27.0 ^ (1.0 / 3.0)
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			// Cube root of 27 is 3.0
			result := math.Float64frombits(results[0])
			Expect(result).To(BeNumerically("~", 3.0, 0.0001))
		})

		It("Should execute negative fractional f64 power: 0.5^(-1.0) = 2.0", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() f64 {
				return 0.5 ^ -1.0
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			// 0.5^(-1) = 1/0.5 = 2.0
			Expect(results[0]).To(Equal(math.Float64bits(2.0)))
		})
	})

	Describe("Power operator with literal type inference (SY-3207)", func() {
		BeforeEach(func() {
			// Setup bindings for math operations with actual runtime implementations
			b := bindings.NewBindings()
			arcRuntime := runtimebindings.NewRuntime(nil, nil)
			runtimebindings.BindRuntime(arcRuntime, b)
			Expect(b.Bind(ctx, r)).To(Succeed())
		})

		It("Should execute f32 variable with integer literal: x^2", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() f32 {
				x f32 := 3.0
				return x ^ 2
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			// 3.0^2 = 9.0
			Expect(results[0]).To(Equal(uint64(math.Float32bits(9.0))))
		})

		It("Should execute f64 variable with integer literal: x^3", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() f64 {
				x f64 := 2.0
				return x ^ 3
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			// 2.0^3 = 8.0
			Expect(results[0]).To(Equal(math.Float64bits(8.0)))
		})

		It("Should execute i32 variable with integer literal: x^2", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() i32 {
				x i32 := 5
				return x ^ 2
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			// 5^2 = 25
			Expect(int32(results[0])).To(Equal(int32(25)))
		})

		It("Should execute f32 with float literal exponent: x^2.5", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() f32 {
				x f32 := 4.0
				return x ^ 2.5
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			// 4.0^2.5 = 32.0
			result := math.Float32frombits(uint32(results[0]))
			Expect(result).To(BeNumerically("~", 32.0, 0.0001))
		})

		It("Should execute complex expression with power and literals: 2 * x^2 + 3", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() f32 {
				x f32 := 3.0
				return 2 * x ^ 2 + 3
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			// 2 * 3.0^2 + 3 = 2 * 9.0 + 3 = 21.0
			result := math.Float32frombits(uint32(results[0]))
			Expect(result).To(BeNumerically("~", 21.0, 0.0001))
		})

		It("Should execute chained power with literals: x^2^3", func() {
			output := MustSucceed(compileWithHostImports(`
			func power() f32 {
				x f32 := 2.0
				return x ^ 2 ^ 3
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			power := mod.ExportedFunction("power")
			Expect(power).ToNot(BeNil())

			results := MustSucceed(power.Call(ctx))
			Expect(results).To(HaveLen(1))
			// 2.0^(2^3) = 2.0^8 = 256.0
			result := math.Float32frombits(uint32(results[0]))
			Expect(result).To(BeNumerically("~", 256.0, 0.0001))
		})
	})

	Describe("Unit Literals", func() {
		It("Should compile unit literal with scale conversion", func() {
			output := MustSucceed(compile(`
			func getMs() f64 {
				return 300ms
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			getMs := mod.ExportedFunction("getMs")
			Expect(getMs).ToNot(BeNil())

			results := MustSucceed(getMs.Call(ctx))
			Expect(results).To(HaveLen(1))
			result := math.Float64frombits(results[0])
			Expect(result).To(BeNumerically("~", 300000000.0, 1)) // 300ms = 300 million ns
		})

		It("Should compile kilometer literal", func() {
			output := MustSucceed(compile(`
			func getKm() f64 {
				return 5km
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			getKm := mod.ExportedFunction("getKm")
			results := MustSucceed(getKm.Call(ctx))
			result := math.Float64frombits(results[0])
			Expect(result).To(Equal(5000.0))
		})

		It("Should compile psi literal", func() {
			output := MustSucceed(compile(`
			func getPsi() f64 {
				return 100psi
			}
			`, nil))

			mod := MustSucceed(r.Instantiate(ctx, output.WASM))
			getPsi := mod.ExportedFunction("getPsi")
			results := MustSucceed(getPsi.Call(ctx))
			result := math.Float64frombits(results[0])
			Expect(result).To(BeNumerically("~", 689476.0, 1.0))
		})
	})
})
