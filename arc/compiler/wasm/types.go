// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/maps"
)

// WASM binary format constants
const (
	Version       = 0x00000001 // version 1 (little endian)
	SectionCustom = 0x00
	SectionType   = 0x01
	SectionImport = 0x02
	SectionFunc   = 0x03
	SectionMemory = 0x05
	SectionExport = 0x07
	SectionCode   = 0x0a
)

// ValueType represents WASM value types
type ValueType byte

const (
	I32 ValueType = 0x7f
	I64 ValueType = 0x7e
	F32 ValueType = 0x7d
	F64 ValueType = 0x7c
)

// ExportKind represents what kind of export
type ExportKind byte

const (
	ExportFunc   ExportKind = 0x00
	ExportTable  ExportKind = 0x01
	ExportMemory ExportKind = 0x02
	ExportGlobal ExportKind = 0x03
)

var MagicNumber = []byte{0x00, 0x61, 0x73, 0x6d}

func ConvertType(t ir.Type) ValueType {
	// Type variables should never reach the compiler - they must be resolved during analysis
	if ir.IsTypeVariable(t) {
		panic(errors.Newf("compiler received unresolved type variable: %v", t))
	}

	switch t {
	case ir.I8{}, ir.I16{}, ir.I32{}, ir.U8{}, ir.U16{}, ir.U32{}:
		return I32
	case ir.I64{}, ir.U64{}:
		return I64
	case ir.F32{}:
		return F32
	case ir.F64{}:
		return F64
	default:
		return I32
	}
}

func ConvertParams(params maps.Ordered[string, ir.Type]) []ValueType {
	wasmParams := make([]ValueType, 0, params.Count())
	for _, paramType := range params.Iter() {
		wasmParams = append(wasmParams, ConvertType(paramType))
	}
	return wasmParams
}

func binaryOpcode(op string, t ir.Type) (Opcode, error) {
	isFloat := ir.IsFloat(t)
	is64bit := ir.Is64Bit(t)
	switch op {
	case "+":
		if isFloat {
			if is64bit {
				return OpF64Add, nil
			}
			return OpF32Add, nil
		}
		if is64bit {
			return OpI64Add, nil
		}
		return OpI32Add, nil

	case "-":
		if isFloat {
			if is64bit {
				return OpF64Sub, nil
			}
			return OpF32Sub, nil
		}
		if is64bit {
			return OpI64Sub, nil
		}
		return OpI32Sub, nil

	case "*":
		if isFloat {
			if is64bit {
				return OpF64Mul, nil
			}
			return OpF32Mul, nil
		}
		if is64bit {
			return OpI64Mul, nil
		}
		return OpI32Mul, nil

	case "/":
		if isFloat {
			if is64bit {
				return OpF64Div, nil
			}
			return OpF32Div, nil
		}
		// Integer division - need to check if signed or unsigned
		if ir.IsUnsignedInteger(t) {
			if is64bit {
				return OpI64DivU, nil
			}
			return OpI32DivU, nil
		}
		if is64bit {
			return OpI64DivS, nil
		}
		return OpI32DivS, nil

	case "%":
		// Modulo - integers only
		if isFloat {
			// Float modulo would need a host function call
			return 0, errors.New("float modulo not yet implemented")
		}
		if strings.HasPrefix(t.String(), "u") {
			if is64bit {
				return OpI64RemU, nil
			}
			return OpI32RemU, nil
		}
		if is64bit {
			return OpI64RemS, nil
		}
		return OpI32RemS, nil

	case "==":
		if isFloat {
			if is64bit {
				return OpF64Eq, nil
			}
			return OpF32Eq, nil
		}
		if is64bit {
			return OpI64Eq, nil
		}
		return OpI32Eq, nil

	case "!=":
		if isFloat {
			if is64bit {
				return OpF64Ne, nil
			}
			return OpF32Ne, nil
		}
		if is64bit {
			return OpI64Ne, nil
		}
		return OpI32Ne, nil

	case "<":
		if isFloat {
			if is64bit {
				return OpF64Lt, nil
			}
			return OpF32Lt, nil
		}
		if ir.IsUnsignedInteger(t) {
			if is64bit {
				return OpI64LtU, nil
			}
			return OpI32LtU, nil
		}
		if is64bit {
			return OpI64LtS, nil
		}
		return OpI32LtS, nil

	case ">":
		if isFloat {
			if is64bit {
				return OpF64Gt, nil
			}
			return OpF32Gt, nil
		}
		if ir.IsUnsignedInteger(t) {
			if is64bit {
				return OpI64GtU, nil
			}
			return OpI32GtU, nil
		}
		if is64bit {
			return OpI64GtS, nil
		}
		return OpI32GtS, nil

	case "<=":
		if isFloat {
			if is64bit {
				return OpF64Le, nil
			}
			return OpF32Le, nil
		}
		if ir.IsUnsignedInteger(t) {
			if is64bit {
				return OpI64LeU, nil
			}
			return OpI32LeU, nil
		}
		if is64bit {
			return OpI64LeS, nil
		}
		return OpI32LeS, nil

	case ">=":
		if isFloat {
			if is64bit {
				return OpF64Ge, nil
			}
			return OpF32Ge, nil
		}
		if ir.IsUnsignedInteger(t) {
			if is64bit {
				return OpI64GeU, nil
			}
			return OpI32GeU, nil
		}
		if is64bit {
			return OpI64GeS, nil
		}
		return OpI32GeS, nil
	case "^":
		// Exponentiation - needs host function call
		return 0, errors.New("exponentiation not yet implemented")
	default:
		return 0, errors.Newf("unknown operator: %s", op)
	}
}
