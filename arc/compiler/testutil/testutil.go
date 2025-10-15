// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/antlr4-go/antlr/v4"
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

func FunctionScope(ctx context.Context, t ir.Function) *symbol.Scope {
	symbols := &symbol.Scope{}
	s := MustSucceed(symbols.Add(ctx, symbol.Symbol{Name: "func", Kind: symbol.KindFunction, Type: types.I32()}))
	return MustSucceed(s.Add(ctx, symbol.Symbol{Kind: symbol.KindBlock}))
}

func NewContext(ctx context.Context) ccontext.Context[antlr.ParserRuleContext] {
	return NewContextWithFunctionType(ctx, ir.Function{})
}

func NewContextWithFunctionType(ctx context.Context, t ir.Function) ccontext.Context[antlr.ParserRuleContext] {
	return ccontext.CreateRoot(ctx, FunctionScope(ctx, t), false)
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
				// Handle both uint32 and uint64 for compatibility
				switch v := instructions[i+1].(type) {
				case uint32:
					encoder.WriteCall(v)
				case uint64:
					encoder.WriteCall(uint32(v))
				case int:
					encoder.WriteCall(uint32(v))
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
