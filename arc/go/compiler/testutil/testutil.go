// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/types"
	"github.com/samber/lo"
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/resolve"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	arctypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
)

func functionScope(ctx context.Context) *symbol.Symbol {
	ginkgo.GinkgoHelper()
	root := symbol.NewRoot(nil, stl.NewSymbols())
	s := testutil.MustSucceed(
		root.Add(
			ctx,
			symbol.Symbol{
				Name: "func",
				Kind: symbol.KindFunction,
				Type: arctypes.I32(),
			},
		),
	)
	return testutil.MustSucceed(s.Add(ctx, symbol.Symbol{Kind: symbol.KindBlock}))
}

func NewContext(ctx context.Context) ccontext.Context[antlr.ParserRuleContext] {
	ginkgo.GinkgoHelper()
	scope := functionScope(ctx)
	return ccontext.NewRoot(
		ctx,
		scope,
		make(map[antlr.ParserRuleContext]arctypes.Type),
		resolve.NewResolver(),
	)
}

// FinalizeContext calls FinalizeAndPatch on the context's Resolver and returns
// the patched bytecodes from the writer. This must be called after compilation
// when the context has a Resolver.
func FinalizeContext(ctx ccontext.Context[antlr.ParserRuleContext]) []byte {
	if ctx.Resolver != nil {
		ctx.Resolver.FinalizeAndPatch(ctx.Module)
	}
	return ctx.Writer.Bytes()
}

// operandWriter writes op and its operands from rest, returning how many entries of
// rest it consumed.
type operandWriter func(w *wasm.Writer, op wasm.Opcode, rest []any) int

// operand returns rest[0], which must be a T.
func operand[T any](rest []any) T {
	v, ok := rest[0].(T)
	if !ok {
		panic(fmt.Sprintf("operand is a %T, not a %T", rest[0], v))
	}
	return v
}

func noOperand(w *wasm.Writer, op wasm.Opcode, _ []any) int {
	w.WriteOpcode(op)
	return 0
}

func i32Const(w *wasm.Writer, _ wasm.Opcode, rest []any) int {
	w.WriteI32Const(operand[int32](rest))
	return 1
}

func i64Const(w *wasm.Writer, _ wasm.Opcode, rest []any) int {
	w.WriteI64Const(operand[int64](rest))
	return 1
}

func f32Const(w *wasm.Writer, _ wasm.Opcode, rest []any) int {
	w.WriteF32Const(operand[float32](rest))
	return 1
}

func f64Const(w *wasm.Writer, _ wasm.Opcode, rest []any) int {
	w.WriteF64Const(operand[float64](rest))
	return 1
}

func localGet(w *wasm.Writer, _ wasm.Opcode, rest []any) int {
	w.WriteLocalGet(operand[int](rest))
	return 1
}

func localSet(w *wasm.Writer, _ wasm.Opcode, rest []any) int {
	w.WriteLocalSet(operand[int](rest))
	return 1
}

// call writes a fixed 5-byte LEB128 index to match WriteCallPlaceholder and PatchCall.
func call(w *wasm.Writer, op wasm.Opcode, rest []any) int {
	w.WriteOpcode(op)
	switch v := rest[0].(type) {
	case uint32:
		w.WriteLEB128Fixed5(uint64(v))
	case uint64:
		w.WriteLEB128Fixed5(v)
	case int:
		w.WriteLEB128Fixed5(uint64(v))
	default:
		panic(fmt.Sprintf("call index must be an integer, got %T", rest[0]))
	}
	return 1
}

// blockType returns the optional block type at the head of rest and how many entries
// it consumed. A byte names a value type.
func blockType(rest []any) (wasm.BlockType, int) {
	if len(rest) == 0 {
		return wasm.BlockTypeEmpty, 0
	}
	switch bt := rest[0].(type) {
	case wasm.BlockType:
		return bt, 1
	case byte:
		switch wasm.ValueType(bt) {
		case wasm.I32:
			return wasm.BlockTypeI32, 1
		case wasm.I64:
			return wasm.BlockTypeI64, 1
		case wasm.F32:
			return wasm.BlockTypeF32, 1
		case wasm.F64:
			return wasm.BlockTypeF64, 1
		default:
			return wasm.BlockTypeEmpty, 1
		}
	}
	return wasm.BlockTypeEmpty, 0
}

func ifBlock(w *wasm.Writer, _ wasm.Opcode, rest []any) int {
	bt, n := blockType(rest)
	w.WriteIf(bt)
	return n
}

func block(w *wasm.Writer, _ wasm.Opcode, rest []any) int {
	bt, n := blockType(rest)
	w.WriteBlock(bt)
	return n
}

func loop(w *wasm.Writer, _ wasm.Opcode, rest []any) int {
	bt, n := blockType(rest)
	w.WriteLoop(bt)
	return n
}

func br(w *wasm.Writer, _ wasm.Opcode, rest []any) int {
	w.WriteBr(operand[uint32](rest))
	return 1
}

func brIf(w *wasm.Writer, _ wasm.Opcode, rest []any) int {
	w.WriteBrIf(operand[uint32](rest))
	return 1
}

// operandWriters holds the writer for every opcode the assembler accepts.
var operandWriters = map[wasm.Opcode]operandWriter{
	wasm.OpUnreachable:       noOperand,
	wasm.OpNop:               noOperand,
	wasm.OpBlock:             block,
	wasm.OpLoop:              loop,
	wasm.OpIf:                ifBlock,
	wasm.OpElse:              noOperand,
	wasm.OpEnd:               noOperand,
	wasm.OpBr:                br,
	wasm.OpBrIf:              brIf,
	wasm.OpBrTable:           noOperand,
	wasm.OpReturn:            noOperand,
	wasm.OpCall:              call,
	wasm.OpCallIndirect:      noOperand,
	wasm.OpDrop:              noOperand,
	wasm.OpSelect:            noOperand,
	wasm.OpLocalGet:          localGet,
	wasm.OpLocalSet:          localSet,
	wasm.OpLocalTee:          noOperand,
	wasm.OpGlobalGet:         noOperand,
	wasm.OpGlobalSet:         noOperand,
	wasm.OpI32Load:           noOperand,
	wasm.OpI64Load:           noOperand,
	wasm.OpF32Load:           noOperand,
	wasm.OpF64Load:           noOperand,
	wasm.OpI32Load8S:         noOperand,
	wasm.OpI32Load8U:         noOperand,
	wasm.OpI32Load16S:        noOperand,
	wasm.OpI32Load16U:        noOperand,
	wasm.OpI32Store:          noOperand,
	wasm.OpI64Store:          noOperand,
	wasm.OpF32Store:          noOperand,
	wasm.OpF64Store:          noOperand,
	wasm.OpI32Store8:         noOperand,
	wasm.OpI32Store16:        noOperand,
	wasm.OpMemorySize:        noOperand,
	wasm.OpMemoryGrow:        noOperand,
	wasm.OpI32Const:          i32Const,
	wasm.OpI64Const:          i64Const,
	wasm.OpF32Const:          f32Const,
	wasm.OpF64Const:          f64Const,
	wasm.OpI32Eqz:            noOperand,
	wasm.OpI32Eq:             noOperand,
	wasm.OpI32Ne:             noOperand,
	wasm.OpI32LtS:            noOperand,
	wasm.OpI32LtU:            noOperand,
	wasm.OpI32GtS:            noOperand,
	wasm.OpI32GtU:            noOperand,
	wasm.OpI32LeS:            noOperand,
	wasm.OpI32LeU:            noOperand,
	wasm.OpI32GeS:            noOperand,
	wasm.OpI32GeU:            noOperand,
	wasm.OpI64Eqz:            noOperand,
	wasm.OpI64Eq:             noOperand,
	wasm.OpI64Ne:             noOperand,
	wasm.OpI64LtS:            noOperand,
	wasm.OpI64LtU:            noOperand,
	wasm.OpI64GtS:            noOperand,
	wasm.OpI64GtU:            noOperand,
	wasm.OpI64LeS:            noOperand,
	wasm.OpI64LeU:            noOperand,
	wasm.OpI64GeS:            noOperand,
	wasm.OpI64GeU:            noOperand,
	wasm.OpF32Eq:             noOperand,
	wasm.OpF32Ne:             noOperand,
	wasm.OpF32Lt:             noOperand,
	wasm.OpF32Gt:             noOperand,
	wasm.OpF32Le:             noOperand,
	wasm.OpF32Ge:             noOperand,
	wasm.OpF64Eq:             noOperand,
	wasm.OpF64Ne:             noOperand,
	wasm.OpF64Lt:             noOperand,
	wasm.OpF64Gt:             noOperand,
	wasm.OpF64Le:             noOperand,
	wasm.OpF64Ge:             noOperand,
	wasm.OpI32Clz:            noOperand,
	wasm.OpI32Ctz:            noOperand,
	wasm.OpI32Popcnt:         noOperand,
	wasm.OpI32Add:            noOperand,
	wasm.OpI32Sub:            noOperand,
	wasm.OpI32Mul:            noOperand,
	wasm.OpI32DivS:           noOperand,
	wasm.OpI32DivU:           noOperand,
	wasm.OpI32RemS:           noOperand,
	wasm.OpI32RemU:           noOperand,
	wasm.OpI32And:            noOperand,
	wasm.OpI32Or:             noOperand,
	wasm.OpI32Xor:            noOperand,
	wasm.OpI32Shl:            noOperand,
	wasm.OpI32ShrS:           noOperand,
	wasm.OpI32ShrU:           noOperand,
	wasm.OpI32Rotl:           noOperand,
	wasm.OpI32Rotr:           noOperand,
	wasm.OpI64Clz:            noOperand,
	wasm.OpI64Ctz:            noOperand,
	wasm.OpI64Popcnt:         noOperand,
	wasm.OpI64Add:            noOperand,
	wasm.OpI64Sub:            noOperand,
	wasm.OpI64Mul:            noOperand,
	wasm.OpI64DivS:           noOperand,
	wasm.OpI64DivU:           noOperand,
	wasm.OpI64RemS:           noOperand,
	wasm.OpI64RemU:           noOperand,
	wasm.OpI64And:            noOperand,
	wasm.OpI64Or:             noOperand,
	wasm.OpI64Xor:            noOperand,
	wasm.OpI64Shl:            noOperand,
	wasm.OpI64ShrS:           noOperand,
	wasm.OpI64ShrU:           noOperand,
	wasm.OpI64Rotl:           noOperand,
	wasm.OpI64Rotr:           noOperand,
	wasm.OpF32Abs:            noOperand,
	wasm.OpF32Neg:            noOperand,
	wasm.OpF32Ceil:           noOperand,
	wasm.OpF32Floor:          noOperand,
	wasm.OpF32Trunc:          noOperand,
	wasm.OpF32Nearest:        noOperand,
	wasm.OpF32Sqrt:           noOperand,
	wasm.OpF32Add:            noOperand,
	wasm.OpF32Sub:            noOperand,
	wasm.OpF32Mul:            noOperand,
	wasm.OpF32Div:            noOperand,
	wasm.OpF32Min:            noOperand,
	wasm.OpF32Max:            noOperand,
	wasm.OpF32Copysign:       noOperand,
	wasm.OpF64Abs:            noOperand,
	wasm.OpF64Neg:            noOperand,
	wasm.OpF64Ceil:           noOperand,
	wasm.OpF64Floor:          noOperand,
	wasm.OpF64Trunc:          noOperand,
	wasm.OpF64Nearest:        noOperand,
	wasm.OpF64Sqrt:           noOperand,
	wasm.OpF64Add:            noOperand,
	wasm.OpF64Sub:            noOperand,
	wasm.OpF64Mul:            noOperand,
	wasm.OpF64Div:            noOperand,
	wasm.OpF64Min:            noOperand,
	wasm.OpF64Max:            noOperand,
	wasm.OpF64Copysign:       noOperand,
	wasm.OpI32WrapI64:        noOperand,
	wasm.OpI32TruncF32S:      noOperand,
	wasm.OpI32TruncF32U:      noOperand,
	wasm.OpI32TruncF64S:      noOperand,
	wasm.OpI32TruncF64U:      noOperand,
	wasm.OpI64ExtendI32S:     noOperand,
	wasm.OpI64ExtendI32U:     noOperand,
	wasm.OpI64TruncF32S:      noOperand,
	wasm.OpI64TruncF32U:      noOperand,
	wasm.OpI64TruncF64S:      noOperand,
	wasm.OpI64TruncF64U:      noOperand,
	wasm.OpF32ConvertI32S:    noOperand,
	wasm.OpF32ConvertI32U:    noOperand,
	wasm.OpF32ConvertI64S:    noOperand,
	wasm.OpF32ConvertI64U:    noOperand,
	wasm.OpF32DemoteF64:      noOperand,
	wasm.OpF64ConvertI32S:    noOperand,
	wasm.OpF64ConvertI32U:    noOperand,
	wasm.OpF64ConvertI64S:    noOperand,
	wasm.OpF64ConvertI64U:    noOperand,
	wasm.OpF64PromoteF32:     noOperand,
	wasm.OpI32ReinterpretF32: noOperand,
	wasm.OpI64ReinterpretF64: noOperand,
	wasm.OpF32ReinterpretI32: noOperand,
	wasm.OpF64ReinterpretI64: noOperand,
}

func assemble(instructions ...any) []byte {
	var out bytes.Buffer
	encoder := wasm.NewWriter()
	for i := 0; i < len(instructions); i++ {
		switch instr := instructions[i].(type) {
		case wasm.Opcode:
			write, ok := operandWriters[instr]
			if !ok {
				panic(fmt.Sprintf("assemble: no operand writer for opcode %s", instr))
			}
			i += write(encoder, instr, instructions[i+1:])
		case wasm.TruncSatOpcode:
			encoder.WriteTruncSat(instr)
		case []byte:
			out.Write(encoder.Bytes())
			out.Write(instr)
			encoder = wasm.NewWriter()
		}
	}
	out.Write(encoder.Bytes())
	return out.Bytes()
}

// opcodeMatcher is a custom Gomega matcher for comparing opcode sequences.
type opcodeMatcher struct {
	expected wasm.OPCodes
}

// MatchOpcodes returns a Gomega matcher that compares two opcode sequences for
// equality. It provides clear, readable failure messages that show the expected vs
// actual opcodes.
//
// Example usage:
//
//	bytecode := compiler.Compile(expr)
//	Expect(bytecode).To(MatchOpcodes(wasm.OpI32Const, wasm.OpI32Add))
func MatchOpcodes(expected ...any) types.GomegaMatcher {
	var opcodes wasm.OPCodes

	// Handle different input types
	switch len(expected) {
	case 0:
		return &opcodeMatcher{expected: opcodes}
	case 1:
		// Check if it's already a byte slice
		if bytes, ok := expected[0].([]byte); ok {
			opcodes = opcodesFromBytes(bytes)
			return &opcodeMatcher{expected: opcodes}
		}
		// Check if it's already OPCodes
		if ops, ok := expected[0].(wasm.OPCodes); ok {
			return &opcodeMatcher{expected: ops}
		}
	}

	// Convert variadic opcodes to OPCodes
	bytes := assemble(expected...)
	opcodes = opcodesFromBytes(bytes)
	return &opcodeMatcher{expected: opcodes}
}

func (m *opcodeMatcher) Match(actual any) (success bool, err error) {
	var actualOpcodes wasm.OPCodes

	switch v := actual.(type) {
	case []byte:
		actualOpcodes = opcodesFromBytes(v)
	case wasm.OPCodes:
		actualOpcodes = v
	default:
		return false, errors.Newf(
			"MatchOpcodes expects []byte or wasm.OPCodes, got %T",
			actual,
		)
	}

	if len(actualOpcodes) != len(m.expected) {
		return false, nil
	}

	for i := range actualOpcodes {
		if actualOpcodes[i] != m.expected[i] {
			return false, nil
		}
	}

	return true, nil
}

func (m *opcodeMatcher) FailureMessage(actual any) string {
	var actualOpcodes wasm.OPCodes

	switch v := actual.(type) {
	case []byte:
		actualOpcodes = opcodesFromBytes(v)
	case wasm.OPCodes:
		actualOpcodes = v
	default:
		return fmt.Sprintf("Expected []byte or wasm.OPCodes, got %T", actual)
	}

	var b strings.Builder
	b.WriteString("Opcodes did not match:\n")

	// Show side-by-side comparison
	maxLen := max(len(actualOpcodes), len(m.expected))

	fmt.Fprintf(&b, "\n%-4s  %-30s  %-30s\n", "Idx", "Expected", "Actual")
	b.WriteString(strings.Repeat("-", 70) + "\n")

	for i := range maxLen {
		var expectedStr, actualStr string
		var marker string

		if i < len(m.expected) {
			expectedStr = m.expected[i].String()
		} else {
			expectedStr = "<missing>"
		}

		if i < len(actualOpcodes) {
			actualStr = actualOpcodes[i].String()
		} else {
			actualStr = "<missing>"
		}

		if expectedStr != actualStr {
			marker = " ✗"
		} else {
			marker = " ✓"
		}

		_, _ = fmt.Fprintf(
			&b,
			"%-4d  %-30s  %-30s%s\n",
			i,
			expectedStr,
			actualStr,
			marker,
		)
	}

	_, _ = fmt.Fprintf(&b, "\nExpected: %s\n", m.expected.String())
	_, _ = fmt.Fprintf(&b, "Actual:   %s\n", actualOpcodes.String())

	return b.String()
}

func (m *opcodeMatcher) NegatedFailureMessage(actual any) string {
	var actualOpcodes wasm.OPCodes

	switch v := actual.(type) {
	case []byte:
		actualOpcodes = opcodesFromBytes(v)
	case wasm.OPCodes:
		actualOpcodes = v
	default:
		return fmt.Sprintf("Expected not to match, but got invalid type %T", actual)
	}

	return "Expected opcodes not to match, but they did:\n  " + actualOpcodes.String()
}

func opcodesFromBytes(data []byte) wasm.OPCodes {
	return lo.Map(data, func(b byte, _ int) wasm.Opcode { return wasm.Opcode(b) })
}

// Compile compiles the module that m generates with wazero, failing the spec when the
// module is not valid WASM. The compiled module closes when the spec ends.
func Compile(ctx context.Context, m *wasm.Module) wazero.CompiledModule {
	ginkgo.GinkgoHelper()
	r := wazero.NewRuntime(ctx)
	ginkgo.DeferCleanup(func(ctx context.Context) error { return r.Close(ctx) })
	return testutil.MustSucceed(r.CompileModule(ctx, m.Generate()))
}

// ImportNames returns the names of the functions that m imports, in import order.
func ImportNames(ctx context.Context, m *wasm.Module) []string {
	ginkgo.GinkgoHelper()
	imports := Compile(ctx, m).ImportedFunctions()
	names := make([]string, len(imports))
	for i, imp := range imports {
		_, names[i], _ = imp.Import()
	}
	return names
}

// LoopBudget returns the instructions that each loop header and function entry run
// against the loop budget global, which a root compiler context declares at index 0.
func LoopBudget() []byte {
	w := wasm.NewWriter()
	w.WriteLoopBudget(0)
	return w.Bytes()
}
