// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import (
	"strings"

	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// WASM binary format constants.
const (
	sectionCustom = 0x00
	sectionType   = 0x01
	sectionImport = 0x02
	sectionFunc   = 0x03
	sectionMemory = 0x05
	sectionGlobal = 0x06
	sectionExport = 0x07
	sectionCode   = 0x0a
	sectionData   = 0x0b
	funcType      = 0x60
)

// ValueType represents WASM value types.
type ValueType byte

const (
	I32 ValueType = 0x7f
	I64 ValueType = 0x7e
	F32 ValueType = 0x7d
	F64 ValueType = 0x7c
)

// ExportKind represents what kind of export.
type ExportKind byte

const (
	ExportKindFunc   ExportKind = 0x00
	ExportKindTable  ExportKind = 0x01
	ExportKindMemory ExportKind = 0x02
	ExportKindGlobal ExportKind = 0x03
)

var (
	// The WASM magic number that must be present at the start of all files.
	magicNumber = []byte{0x00, 0x61, 0x73, 0x6d}
	// version is the version of WASM the arc compiler compiles to (V1).
	version = []byte{0x01, 0x00, 0x00, 0x00}
)

func ConvertType(t types.Type) ValueType {
	if t.Kind == types.KindF64 {
		return F64
	}
	if t.Is64Bit() {
		return I64
	}
	if t.Kind == types.KindF32 {
		return F32
	}
	return I32
}

// binaryOps holds the opcodes of one binary operator for each value type. Operators
// that ignore sign repeat the signed opcodes in the unsigned fields.
type binaryOps struct {
	f32, f64, i32S, i64S, i32U, i64U Opcode
}

func signless(f32, f64, i32, i64 Opcode) binaryOps {
	return binaryOps{f32: f32, f64: f64, i32S: i32, i64S: i64, i32U: i32, i64U: i64}
}

func lookupBinaryOps(op string) (binaryOps, bool) {
	switch op {
	case "+":
		return signless(OpF32Add, OpF64Add, OpI32Add, OpI64Add), true
	case "-":
		return signless(OpF32Sub, OpF64Sub, OpI32Sub, OpI64Sub), true
	case "*":
		return signless(OpF32Mul, OpF64Mul, OpI32Mul, OpI64Mul), true
	case "/":
		return binaryOps{
			OpF32Div,
			OpF64Div,
			OpI32DivS,
			OpI64DivS,
			OpI32DivU,
			OpI64DivU,
		}, true
	case "%":
		return binaryOps{
			i32S: OpI32RemS,
			i64S: OpI64RemS,
			i32U: OpI32RemU,
			i64U: OpI64RemU,
		}, true
	case "==":
		return signless(OpF32Eq, OpF64Eq, OpI32Eq, OpI64Eq), true
	case "!=":
		return signless(OpF32Ne, OpF64Ne, OpI32Ne, OpI64Ne), true
	case "<":
		return binaryOps{OpF32Lt, OpF64Lt, OpI32LtS, OpI64LtS, OpI32LtU, OpI64LtU}, true
	case ">":
		return binaryOps{OpF32Gt, OpF64Gt, OpI32GtS, OpI64GtS, OpI32GtU, OpI64GtU}, true
	case "<=":
		return binaryOps{OpF32Le, OpF64Le, OpI32LeS, OpI64LeS, OpI32LeU, OpI64LeU}, true
	case ">=":
		return binaryOps{OpF32Ge, OpF64Ge, OpI32GeS, OpI64GeS, OpI32GeU, OpI64GeU}, true
	default:
		return binaryOps{}, false
	}
}

func binaryOpcode(op string, t types.Type) (Opcode, error) {
	ops, ok := lookupBinaryOps(op)
	if !ok {
		return 0, errors.Newf("unknown operator: %s", op)
	}
	is64bit := t.Is64Bit()
	if t.IsFloat() {
		// Float modulo would need a host function call.
		if op == "%" {
			return 0, errors.New("float modulo not yet implemented")
		}
		return pickOpcode(is64bit, ops.f32, ops.f64), nil
	}
	unsigned := t.IsUnsignedInteger()
	if op == "%" {
		unsigned = strings.HasPrefix(t.String(), "u")
	}
	if unsigned {
		return pickOpcode(is64bit, ops.i32U, ops.i64U), nil
	}
	return pickOpcode(is64bit, ops.i32S, ops.i64S), nil
}

func pickOpcode(is64bit bool, op32, op64 Opcode) Opcode {
	if is64bit {
		return op64
	}
	return op32
}
