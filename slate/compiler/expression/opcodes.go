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
	"fmt"
	"strings"

	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/types"
)

// MapType converts a Slate type to a WASM value type
func MapType(t types.Type) wasm.ValueType {
	typeStr := t.String()
	switch typeStr {
	// Small integers all use i32 in WASM
	case "i8", "i16", "i32", "u8", "u16", "u32":
		return wasm.I32
	// 64-bit integers
	case "i64", "u64", "timestamp", "timespan":
		return wasm.I64
	// Floats
	case "f32":
		return wasm.F32
	case "f64":
		return wasm.F64
	// Channels, strings, series all use handles (i32)
	default:
		if strings.HasPrefix(typeStr, "chan ") ||
			strings.HasPrefix(typeStr, "<-chan ") ||
			strings.HasPrefix(typeStr, "->chan ") ||
			strings.HasPrefix(typeStr, "series ") ||
			typeStr == "string" {
			return wasm.I32 // Handle
		}
		// Unknown type - default to i32
		return wasm.I32
	}
}

// GetBinaryOpcode returns the WASM opcode for a binary operation
func GetBinaryOpcode(op string, t types.Type) (wasm.Opcode, error) {
	typeStr := t.String()
	// Determine base type for the operation
	var isFloat bool
	var is64bit bool
	switch typeStr {
	case "f32":
		isFloat = true
		is64bit = false
	case "f64":
		isFloat = true
		is64bit = true
	case "i64", "u64", "timestamp", "timespan":
		is64bit = true
		isFloat = false
	default:
		is64bit = false
		isFloat = false
	}
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
				return 0xa2, nil // f64.mul
			}
			return 0x94, nil // f32.mul
		}
		if is64bit {
			return 0x7e, nil // i64.mul
		}
		return 0x6c, nil // i32.mul

	case "/":
		if isFloat {
			if is64bit {
				return 0xa3, nil // f64.div
			}
			return 0x95, nil // f32.div
		}
		// Integer division - need to check if signed or unsigned
		if strings.HasPrefix(typeStr, "u") {
			if is64bit {
				return 0x80, nil // i64.div_u
			}
			return 0x6e, nil // i32.div_u
		}
		if is64bit {
			return 0x7f, nil // i64.div_s
		}
		return 0x6d, nil // i32.div_s

	case "%":
		// Modulo - integers only
		if isFloat {
			// Float modulo would need a host function call
			return 0, fmt.Errorf("float modulo not yet implemented")
		}
		if strings.HasPrefix(typeStr, "u") {
			if is64bit {
				return 0x82, nil // i64.rem_u
			}
			return 0x70, nil // i32.rem_u
		}
		if is64bit {
			return 0x81, nil // i64.rem_s
		}
		return 0x6f, nil // i32.rem_s

	case "==":
		if isFloat {
			if is64bit {
				return 0x61, nil // f64.eq
			}
			return 0x5b, nil // f32.eq
		}
		if is64bit {
			return 0x51, nil // i64.eq
		}
		return 0x46, nil // i32.eq

	case "!=":
		if isFloat {
			if is64bit {
				return 0x62, nil // f64.ne
			}
			return 0x5c, nil // f32.ne
		}
		if is64bit {
			return 0x52, nil // i64.ne
		}
		return 0x47, nil // i32.ne

	case "<":
		if isFloat {
			if is64bit {
				return 0x63, nil // f64.lt
			}
			return 0x5d, nil // f32.lt
		}
		if strings.HasPrefix(typeStr, "u") {
			if is64bit {
				return 0x53, nil // i64.lt_u
			}
			return 0x49, nil // i32.lt_u
		}
		if is64bit {
			return 0x54, nil // i64.lt_s
		}
		return 0x48, nil // i32.lt_s

	case ">":
		if isFloat {
			if is64bit {
				return 0x64, nil // f64.gt
			}
			return 0x5e, nil // f32.gt
		}
		if strings.HasPrefix(typeStr, "u") {
			if is64bit {
				return 0x55, nil // i64.gt_u
			}
			return 0x4b, nil // i32.gt_u
		}
		if is64bit {
			return 0x56, nil // i64.gt_s
		}
		return 0x4a, nil // i32.gt_s

	case "<=":
		if isFloat {
			if is64bit {
				return 0x65, nil // f64.le
			}
			return 0x5f, nil // f32.le
		}
		if strings.HasPrefix(typeStr, "u") {
			if is64bit {
				return 0x57, nil // i64.le_u
			}
			return 0x4d, nil // i32.le_u
		}
		if is64bit {
			return 0x58, nil // i64.le_s
		}
		return 0x4c, nil // i32.le_s

	case ">=":
		if isFloat {
			if is64bit {
				return 0x66, nil // f64.ge
			}
			return 0x60, nil // f32.ge
		}
		if strings.HasPrefix(typeStr, "u") {
			if is64bit {
				return 0x59, nil // i64.ge_u
			}
			return 0x4f, nil // i32.ge_u
		}
		if is64bit {
			return 0x5a, nil // i64.ge_s
		}
		return 0x4e, nil // i32.ge_s

	case "&&", "||":
		// Logical operations - need special handling
		// These should normalize to 0 or 1 (u8)
		return 0, fmt.Errorf("logical operations need special handling")

	case "^":
		// Exponentiation - needs host function call
		return 0, fmt.Errorf("exponentiation not yet implemented")

	default:
		return 0, fmt.Errorf("unknown operator: %s", op)
	}
}

// GetUnaryOpcode returns the WASM opcode for a unary operation
func GetUnaryOpcode(op string, t types.Type) (byte, error) {
	typeStr := t.String()

	switch op {
	case "-":
		// Negation: 0 - value
		// This is handled by pushing 0 then subtracting
		// Return the subtract opcode
		if typeStr == "f32" {
			return 0x93, nil // f32.sub
		} else if typeStr == "f64" {
			return 0xa1, nil // f64.sub
		} else if typeStr == "i64" || typeStr == "u64" {
			return 0x7d, nil // i64.sub
		} else {
			return 0x6b, nil // i32.sub
		}

	case "!":
		// Logical NOT: value == 0
		// Use eqz instruction
		if typeStr == "i64" || typeStr == "u64" {
			return 0x50, nil // i64.eqz
		}
		return 0x45, nil // i32.eqz

	default:
		return 0, fmt.Errorf("unknown unary operator: %s", op)
	}
}
