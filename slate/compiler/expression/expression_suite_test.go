// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression_test

import (
	"encoding/hex"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/compiler/expression"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/types"
	. "github.com/synnaxlabs/x/testutil"
)

func expectExpression(expression string, expectedType types.Type, expectedOpcodes ...any) {
	bytecode, exprType := compileExpression(expression)
	expected := WASM(expectedOpcodes...)
	Expect(bytecode).To(Equal(expected))
	Expect(exprType).To(Equal(expectedType))
}

func compileExpression(source string) ([]byte, types.Type) {
	expr := MustSucceed(parser.ParseExpression(source))
	module := wasm.NewModule()
	symbols := &symbol.Scope{}
	ctx := compiler.NewContext(module, symbols)
	ctx.EnterFunction("test", nil)
	compiler := expression.NewCompiler(ctx)
	exprType := MustSucceed(compiler.Compile(expr))
	return compiler.Bytes(), exprType
}

func hexToBytes(hexStr string) []byte {
	cleanHex := strings.ReplaceAll(hexStr, " ", "")
	bytes := MustSucceed(hex.DecodeString(cleanHex))
	return bytes
}

// WASM builds WASM bytecode from a variadic slice of opcodes and operands
func WASM(instructions ...any) []byte {
	encoder := wasm.NewEncoder()

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
				encoder.WriteLocalGet(instructions[i+1].(uint32))
				i++ // Skip the operand
			case wasm.OpIf:
				// Check if there's a block type following
				if i+1 < len(instructions) {
					if bt, ok := instructions[i+1].(byte); ok {
						// It's a block type byte
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

func TestExpression(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expression Suite")
}
