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
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/onsi/gomega/types"
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/resolve"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channel"
	stlerrors "github.com/synnaxlabs/arc/stl/errors"
	"github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stateful"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/symbol"
	arctypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
)

func FunctionScope(ctx context.Context) *symbol.Scope {
	symbols := &symbol.Scope{}
	s := testutil.MustSucceed(symbols.Add(ctx, symbol.Symbol{Name: "func", Kind: symbol.KindFunction, Type: arctypes.I32()}))
	return testutil.MustSucceed(s.Add(ctx, symbol.Symbol{Kind: symbol.KindBlock}))
}

// NewStdlibResolver returns a symbol resolver built from actual STL modules.
// Each module provides its own symbol definitions — this is the single source
// of truth for host function type signatures.
func NewStdlibResolver() symbol.Resolver {
	return stl.CompoundResolver(
		channel.NewModule(nil, nil),
		stateful.NewModule(nil, nil),
		series.NewModule(nil),
		stlstrings.NewModule(nil),
		time.NewModule(),
		math.NewModule(),
		stlerrors.NewModule(),
	)
}

func NewContext(ctx context.Context) ccontext.Context[antlr.ParserRuleContext] {
	return ccontext.CreateRoot(ctx, FunctionScope(ctx), make(map[antlr.ParserRuleContext]arctypes.Type), resolve.NewResolver(NewStdlibResolver()))
}

// FinalizeContext calls FinalizeAndPatch on the context's Resolver and returns
// the patched bytecodes from the writer. This must be called after compilation
// when the context has a Resolver.
func FinalizeContext(ctx ccontext.Context[antlr.ParserRuleContext]) []byte {
	if ctx.Resolver != nil {
		if err := ctx.Resolver.FinalizeAndPatch(ctx.Module); err != nil {
			panic(fmt.Sprintf("FinalizeAndPatch failed: %v", err))
		}
	}
	return ctx.Writer.Bytes()
}

// WASM builds WASM bytecode from a variadic slice of opcodes and operands
func WASM(instructions ...any) []byte {
	encoder := wasm.NewWriter()

	for i := 0; i < len(instructions); i++ {
		switch instr := instructions[i].(type) {
		case wasm.Opcode:
			switch instr {
			case wasm.OpI32Const:
				encoder.WriteI32Const(instructions[i+1].(int32))
				i++ // Skip the operand
			case wasm.OpI64Const:
				encoder.WriteI64Const(instructions[i+1].(int64))
				i++ // Skip the operand
			case wasm.OpF32Const:
				encoder.WriteF32Const(instructions[i+1].(float32))
				i++ // Skip the operand
			case wasm.OpF64Const:
				encoder.WriteF64Const(instructions[i+1].(float64))
				i++ // Skip the operand
			case wasm.OpLocalGet:
				encoder.WriteLocalGet(instructions[i+1].(int))
				i++ // Skip the operand
			case wasm.OpLocalSet:
				encoder.WriteLocalSet(instructions[i+1].(int))
				i++ // Skip the operand
			case wasm.OpCall:
				// Use fixed 5-byte LEB128 to match WriteCallPlaceholder+PatchCall encoding
				encoder.WriteOpcode(wasm.OpCall)
				switch v := instructions[i+1].(type) {
				case uint32:
					encoder.WriteLEB128Fixed5(uint64(v))
				case uint64:
					encoder.WriteLEB128Fixed5(v)
				case int:
					encoder.WriteLEB128Fixed5(uint64(v))
				}
				i++ // Skip the operand
			case wasm.OpIf:
				// Check if there's a block type following
				if i+1 < len(instructions) {
					// First check if it's a BlockType interface
					if bt, ok := instructions[i+1].(wasm.BlockType); ok {
						encoder.WriteIf(bt)
						i++ // Skip the block type
					} else if bt, ok := instructions[i+1].(byte); ok {
						// It's a block type byte (for backward compatibility)
						switch wasm.ValueType(bt) {
						case wasm.I32:
							encoder.WriteIf(wasm.BlockTypeI32)
						case wasm.I64:
							encoder.WriteIf(wasm.BlockTypeI64)
						case wasm.F32:
							encoder.WriteIf(wasm.BlockTypeF32)
						case wasm.F64:
							encoder.WriteIf(wasm.BlockTypeF64)
						default:
							encoder.WriteIf(wasm.BlockTypeEmpty)
						}
						i++ // Skip the block type
					} else {
						encoder.WriteIf(wasm.BlockTypeEmpty)
					}
				} else {
					encoder.WriteIf(wasm.BlockTypeEmpty)
				}
			default:
				// Simple opcode with no operands
				encoder.WriteOpcode(instr)
			}
		}
	}

	return encoder.Bytes()
}

// opcodeMatcher is a custom Gomega matcher for comparing opcode sequences
type opcodeMatcher struct {
	expected wasm.OPCodes
}

// MatchOpcodes returns a Gomega matcher that compares two opcode sequences for equality.
// It provides clear, readable failure messages that show the expected vs actual opcodes.
//
// Example usage:
//
//	bytecode := compiler.Compile(expr)
//	Expect(bytecode).To(MatchOpcodes(wasm.OpI32Const, wasm.OpI32Add))
//
// Or with the WASM helper:
//
//	Expect(bytecode).To(MatchOpcodes(WASM(wasm.OpI32Const, int32(10), wasm.OpI32Add)))
func MatchOpcodes(expected ...any) types.GomegaMatcher {
	var opcodes wasm.OPCodes

	// Handle different input types
	switch len(expected) {
	case 0:
		return &opcodeMatcher{expected: opcodes}
	case 1:
		// Check if it's already a byte slice (from WASM helper)
		if bytes, ok := expected[0].([]byte); ok {
			opcodes = wasm.OPCodesFromBytes(bytes)
			return &opcodeMatcher{expected: opcodes}
		}
		// Check if it's already OPCodes
		if ops, ok := expected[0].(wasm.OPCodes); ok {
			return &opcodeMatcher{expected: ops}
		}
	}

	// Convert variadic opcodes to OPCodes
	bytes := WASM(expected...)
	opcodes = wasm.OPCodesFromBytes(bytes)
	return &opcodeMatcher{expected: opcodes}
}

func (m *opcodeMatcher) Match(actual any) (success bool, err error) {
	var actualOpcodes wasm.OPCodes

	switch v := actual.(type) {
	case []byte:
		actualOpcodes = wasm.OPCodesFromBytes(v)
	case wasm.OPCodes:
		actualOpcodes = v
	default:
		return false, errors.Newf("MatchOpcodes expects []byte or wasm.OPCodes, got %T", actual)
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
		actualOpcodes = wasm.OPCodesFromBytes(v)
	case wasm.OPCodes:
		actualOpcodes = v
	default:
		return fmt.Sprintf("Expected []byte or wasm.OPCodes, got %T", actual)
	}

	var b strings.Builder
	b.WriteString("Opcodes did not match:\n")

	// Show side-by-side comparison
	maxLen := len(m.expected)
	if len(actualOpcodes) > maxLen {
		maxLen = len(actualOpcodes)
	}

	b.WriteString(fmt.Sprintf("\n%-4s  %-30s  %-30s\n", "Idx", "Expected", "Actual"))
	b.WriteString(strings.Repeat("-", 70) + "\n")

	for i := 0; i < maxLen; i++ {
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

		b.WriteString(fmt.Sprintf("%-4d  %-30s  %-30s%s\n", i, expectedStr, actualStr, marker))
	}

	b.WriteString(fmt.Sprintf("\nExpected: %s\n", m.expected.String()))
	b.WriteString(fmt.Sprintf("Actual:   %s\n", actualOpcodes.String()))

	return b.String()
}

func (m *opcodeMatcher) NegatedFailureMessage(actual any) string {
	var actualOpcodes wasm.OPCodes

	switch v := actual.(type) {
	case []byte:
		actualOpcodes = wasm.OPCodesFromBytes(v)
	case wasm.OPCodes:
		actualOpcodes = v
	default:
		return fmt.Sprintf("Expected not to match, but got invalid type %T", actual)
	}

	return fmt.Sprintf("Expected opcodes not to match, but they did:\n  %s", actualOpcodes.String())
}
