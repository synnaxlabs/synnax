// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"strings"

	"github.com/synnaxlabs/x/errors"

	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/types"
)

// MapType converts a Slate type to a WASM value type
func MapType(t types.Type) wasm.ValueType {
	switch t {
	case types.I8{}, types.I16{}, types.I32{}, types.U8{}, types.U16{}, types.U32{}:
		return wasm.I32
	case types.I64{}, types.U64{}, types.TimeStamp{}, types.TimeSpan{}:
		return wasm.I64
	case types.F32{}:
		return wasm.F32
	case types.F64{}:
		return wasm.F64
	default:
		typeStr := t.String()
		if strings.HasPrefix(typeStr, "chan ") ||
			strings.HasPrefix(typeStr, "<-chan ") ||
			strings.HasPrefix(typeStr, "->chan ") ||
			strings.HasPrefix(typeStr, "series ") ||
			typeStr == "string" {
			return wasm.I32 // Handle
		}
		return wasm.I32
	}
}

// GetBinaryOpcode returns the WASM opcode for a binary operation
func GetBinaryOpcode(op string, t types.Type) (wasm.Opcode, error) {
	isFloat := types.IsFloat(t)
	is64bit := types.Is64Bit(t)
	switch op {
	case "+":
		if isFloat {
			if is64bit {
				return wasm.OpF64Add, nil
			}
			return wasm.OpF32Add, nil
		}
		if is64bit {
			return wasm.OpI64Add, nil
		}
		return wasm.OpI32Add, nil

	case "-":
		if isFloat {
			if is64bit {
				return wasm.OpF64Sub, nil
			}
			return wasm.OpF32Sub, nil
		}
		if is64bit {
			return wasm.OpI64Sub, nil
		}
		return wasm.OpI32Sub, nil

	case "*":
		if isFloat {
			if is64bit {
				return wasm.OpF64Mul, nil
			}
			return wasm.OpF32Mul, nil
		}
		if is64bit {
			return wasm.OpI64Mul, nil
		}
		return wasm.OpI32Mul, nil

	case "/":
		if isFloat {
			if is64bit {
				return wasm.OpF64Div, nil
			}
			return wasm.OpF32Div, nil
		}
		// Integer division - need to check if signed or unsigned
		if types.IsUnsignedInteger(t) {
			if is64bit {
				return wasm.OpI64DivU, nil
			}
			return wasm.OpI32DivU, nil
		}
		if is64bit {
			return wasm.OpI64DivS, nil
		}
		return wasm.OpI32DivS, nil

	case "%":
		// Modulo - integers only
		if isFloat {
			// Float modulo would need a host function call
			return 0, errors.New("float modulo not yet implemented")
		}
		if strings.HasPrefix(t.String(), "u") {
			if is64bit {
				return wasm.OpI64RemU, nil
			}
			return wasm.OpI32RemU, nil
		}
		if is64bit {
			return wasm.OpI64RemS, nil
		}
		return wasm.OpI32RemS, nil

	case "==":
		if isFloat {
			if is64bit {
				return wasm.OpF64Eq, nil
			}
			return wasm.OpF32Eq, nil
		}
		if is64bit {
			return wasm.OpI64Eq, nil
		}
		return wasm.OpI32Eq, nil

	case "!=":
		if isFloat {
			if is64bit {
				return wasm.OpF64Ne, nil
			}
			return wasm.OpF32Ne, nil
		}
		if is64bit {
			return wasm.OpI64Ne, nil
		}
		return wasm.OpI32Ne, nil

	case "<":
		if isFloat {
			if is64bit {
				return wasm.OpF64Lt, nil
			}
			return wasm.OpF32Lt, nil
		}
		if types.IsUnsignedInteger(t) {
			if is64bit {
				return wasm.OpI64LtU, nil
			}
			return wasm.OpI32LtU, nil
		}
		if is64bit {
			return wasm.OpI64LtS, nil
		}
		return wasm.OpI32LtS, nil

	case ">":
		if isFloat {
			if is64bit {
				return wasm.OpF64Gt, nil
			}
			return wasm.OpF32Gt, nil
		}
		if types.IsUnsignedInteger(t) {
			if is64bit {
				return wasm.OpI64GtU, nil
			}
			return wasm.OpI32GtU, nil
		}
		if is64bit {
			return wasm.OpI64GtS, nil
		}
		return wasm.OpI32GtS, nil

	case "<=":
		if isFloat {
			if is64bit {
				return wasm.OpF64Le, nil
			}
			return wasm.OpF32Le, nil
		}
		if types.IsUnsignedInteger(t) {
			if is64bit {
				return wasm.OpI64LeU, nil
			}
			return wasm.OpI32LeU, nil
		}
		if is64bit {
			return wasm.OpI64LeS, nil
		}
		return wasm.OpI32LeS, nil

	case ">=":
		if isFloat {
			if is64bit {
				return wasm.OpF64Ge, nil
			}
			return wasm.OpF32Ge, nil
		}
		if types.IsUnsignedInteger(t) {
			if is64bit {
				return wasm.OpI64GeU, nil
			}
			return wasm.OpI32GeU, nil
		}
		if is64bit {
			return wasm.OpI64GeS, nil
		}
		return wasm.OpI32GeS, nil
	case "^":
		// Exponentiation - needs host function call
		return 0, errors.New("exponentiation not yet implemented")
	default:
		return 0, errors.Newf("unknown operator: %s", op)
	}
}
