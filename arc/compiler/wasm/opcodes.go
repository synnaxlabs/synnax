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

	"github.com/samber/lo"
)

// Opcode represents a WASM instruction opcode
//
//go:generate go run golang.org/x/tools/cmd/stringer -type=Opcode
type Opcode byte

type OPCodes []Opcode

func OPCodesFromBytes(data []byte) OPCodes {
	return lo.Map(data, func(b byte, _ int) Opcode {
		return Opcode(b)
	})
}

func (o OPCodes) String() string {
	return strings.Join(lo.Map(o, func(op Opcode, _ int) string {
		return op.String()
	}), " ")
}

// Control flow instructions
const (
	OpUnreachable Opcode = 0x00 // unreachable
	OpNop         Opcode = 0x01 // nop
	OpBlock       Opcode = 0x02 // block
	OpLoop        Opcode = 0x03 // loop
	OpIf          Opcode = 0x04 // if
	OpElse        Opcode = 0x05 // else
	OpEnd         Opcode = 0x0b // end
	OpBr          Opcode = 0x0c // br
	OpBrIf        Opcode = 0x0d // br_if
	OpBrTable     Opcode = 0x0e // br_table
	OpReturn      Opcode = 0x0f // return
)

// Call instructions
const (
	OpCall         Opcode = 0x10 // call
	OpCallIndirect Opcode = 0x11 // call_indirect
)

// Variable instructions
const (
	OpLocalGet  Opcode = 0x20 // local.get
	OpLocalSet  Opcode = 0x21 // local.set
	OpLocalTee  Opcode = 0x22 // local.tee
	OpGlobalGet Opcode = 0x23 // global.get
	OpGlobalSet Opcode = 0x24 // global.set
)

// Memory instructions
const (
	OpI32Load    Opcode = 0x28 // i32.load
	OpI64Load    Opcode = 0x29 // i64.load
	OpF32Load    Opcode = 0x2a // f32.load
	OpF64Load    Opcode = 0x2b // f64.load
	OpI32Load8S  Opcode = 0x2c // i32.load8_s
	OpI32Load8U  Opcode = 0x2d // i32.load8_u
	OpI32Load16S Opcode = 0x2e // i32.load16_s
	OpI32Load16U Opcode = 0x2f // i32.load16_u
	OpI32Store   Opcode = 0x36 // i32.store
	OpI64Store   Opcode = 0x37 // i64.store
	OpF32Store   Opcode = 0x38 // f32.store
	OpF64Store   Opcode = 0x39 // f64.store
	OpI32Store8  Opcode = 0x3a // i32.store8
	OpI32Store16 Opcode = 0x3b // i32.store16
	OpMemorySize Opcode = 0x3f // memory.size
	OpMemoryGrow Opcode = 0x40 // memory.grow
)

// Constant instructions
const (
	OpI32Const Opcode = 0x41 // i32.const
	OpI64Const Opcode = 0x42 // i64.const
	OpF32Const Opcode = 0x43 // f32.const
	OpF64Const Opcode = 0x44 // f64.const
)

// i32 comparison instructions
const (
	OpI32Eqz Opcode = 0x45 // i32.eqz
	OpI32Eq  Opcode = 0x46 // i32.eq
	OpI32Ne  Opcode = 0x47 // i32.ne
	OpI32LtS Opcode = 0x48 // i32.lt_s (signed)
	OpI32LtU Opcode = 0x49 // i32.lt_u (unsigned)
	OpI32GtS Opcode = 0x4a // i32.gt_s
	OpI32GtU Opcode = 0x4b // i32.gt_u
	OpI32LeS Opcode = 0x4c // i32.le_s
	OpI32LeU Opcode = 0x4d // i32.le_u
	OpI32GeS Opcode = 0x4e // i32.ge_s
	OpI32GeU Opcode = 0x4f // i32.ge_u
)

// i64 comparison instructions
const (
	OpI64Eqz Opcode = 0x50 // i64.eqz
	OpI64Eq  Opcode = 0x51 // i64.eq
	OpI64Ne  Opcode = 0x52 // i64.ne
	OpI64LtS Opcode = 0x53 // i64.lt_s
	OpI64LtU Opcode = 0x54 // i64.lt_u
	OpI64GtS Opcode = 0x55 // i64.gt_s
	OpI64GtU Opcode = 0x56 // i64.gt_u
	OpI64LeS Opcode = 0x57 // i64.le_s
	OpI64LeU Opcode = 0x58 // i64.le_u
	OpI64GeS Opcode = 0x59 // i64.ge_s
	OpI64GeU Opcode = 0x5a // i64.ge_u
)

// f32 comparison instructions
const (
	OpF32Eq Opcode = 0x5b // f32.eq
	OpF32Ne Opcode = 0x5c // f32.ne
	OpF32Lt Opcode = 0x5d // f32.lt
	OpF32Gt Opcode = 0x5e // f32.gt
	OpF32Le Opcode = 0x5f // f32.le
	OpF32Ge Opcode = 0x60 // f32.ge
)

// f64 comparison instructions
const (
	OpF64Eq Opcode = 0x61 // f64.eq
	OpF64Ne Opcode = 0x62 // f64.ne
	OpF64Lt Opcode = 0x63 // f64.lt
	OpF64Gt Opcode = 0x64 // f64.gt
	OpF64Le Opcode = 0x65 // f64.le
	OpF64Ge Opcode = 0x66 // f64.ge
)

// i32 arithmetic instructions
const (
	OpI32Clz    Opcode = 0x67 // i32.clz (count leading zeros)
	OpI32Ctz    Opcode = 0x68 // i32.ctz (count trailing zeros)
	OpI32Popcnt Opcode = 0x69 // i32.popcnt (population count)
	OpI32Add    Opcode = 0x6a // i32.add
	OpI32Sub    Opcode = 0x6b // i32.sub
	OpI32Mul    Opcode = 0x6c // i32.mul
	OpI32DivS   Opcode = 0x6d // i32.div_s (signed)
	OpI32DivU   Opcode = 0x6e // i32.div_u (unsigned)
	OpI32RemS   Opcode = 0x6f // i32.rem_s (signed remainder/modulo)
	OpI32RemU   Opcode = 0x70 // i32.rem_u (unsigned remainder/modulo)
	OpI32And    Opcode = 0x71 // i32.and
	OpI32Or     Opcode = 0x72 // i32.or
	OpI32Xor    Opcode = 0x73 // i32.xor
	OpI32Shl    Opcode = 0x74 // i32.shl (shift left)
	OpI32ShrS   Opcode = 0x75 // i32.shr_s (shift right signed)
	OpI32ShrU   Opcode = 0x76 // i32.shr_u (shift right unsigned)
	OpI32Rotl   Opcode = 0x77 // i32.rotl (rotate left)
	OpI32Rotr   Opcode = 0x78 // i32.rotr (rotate right)
)

// i64 arithmetic instructions
const (
	OpI64Clz    Opcode = 0x79 // i64.clz
	OpI64Ctz    Opcode = 0x7a // i64.ctz
	OpI64Popcnt Opcode = 0x7b // i64.popcnt
	OpI64Add    Opcode = 0x7c // i64.add
	OpI64Sub    Opcode = 0x7d // i64.sub
	OpI64Mul    Opcode = 0x7e // i64.mul
	OpI64DivS   Opcode = 0x7f // i64.div_s
	OpI64DivU   Opcode = 0x80 // i64.div_u
	OpI64RemS   Opcode = 0x81 // i64.rem_s
	OpI64RemU   Opcode = 0x82 // i64.rem_u
	OpI64And    Opcode = 0x83 // i64.and
	OpI64Or     Opcode = 0x84 // i64.or
	OpI64Xor    Opcode = 0x85 // i64.xor
	OpI64Shl    Opcode = 0x86 // i64.shl
	OpI64ShrS   Opcode = 0x87 // i64.shr_s
	OpI64ShrU   Opcode = 0x88 // i64.shr_u
	OpI64Rotl   Opcode = 0x89 // i64.rotl
	OpI64Rotr   Opcode = 0x8a // i64.rotr
)

// f32 arithmetic instructions
const (
	OpF32Abs      Opcode = 0x8b // f32.abs
	OpF32Neg      Opcode = 0x8c // f32.neg
	OpF32Ceil     Opcode = 0x8d // f32.ceil
	OpF32Floor    Opcode = 0x8e // f32.floor
	OpF32Trunc    Opcode = 0x8f // f32.trunc
	OpF32Nearest  Opcode = 0x90 // f32.nearest
	OpF32Sqrt     Opcode = 0x91 // f32.sqrt
	OpF32Add      Opcode = 0x92 // f32.add
	OpF32Sub      Opcode = 0x93 // f32.sub
	OpF32Mul      Opcode = 0x94 // f32.mul
	OpF32Div      Opcode = 0x95 // f32.div
	OpF32Min      Opcode = 0x96 // f32.min
	OpF32Max      Opcode = 0x97 // f32.max
	OpF32Copysign Opcode = 0x98 // f32.copysign
)

// f64 arithmetic instructions
const (
	OpF64Abs      Opcode = 0x99 // f64.abs
	OpF64Neg      Opcode = 0x9a // f64.neg
	OpF64Ceil     Opcode = 0x9b // f64.ceil
	OpF64Floor    Opcode = 0x9c // f64.floor
	OpF64Trunc    Opcode = 0x9d // f64.trunc
	OpF64Nearest  Opcode = 0x9e // f64.nearest
	OpF64Sqrt     Opcode = 0x9f // f64.sqrt
	OpF64Add      Opcode = 0xa0 // f64.add
	OpF64Sub      Opcode = 0xa1 // f64.sub
	OpF64Mul      Opcode = 0xa2 // f64.mul
	OpF64Div      Opcode = 0xa3 // f64.div
	OpF64Min      Opcode = 0xa4 // f64.min
	OpF64Max      Opcode = 0xa5 // f64.max
	OpF64Copysign Opcode = 0xa6 // f64.copysign
)

// Conversion instructions
const (
	OpI32WrapI64        Opcode = 0xa7 // i32.wrap_i64
	OpI32TruncF32S      Opcode = 0xa8 // i32.trunc_f32_s
	OpI32TruncF32U      Opcode = 0xa9 // i32.trunc_f32_u
	OpI32TruncF64S      Opcode = 0xaa // i32.trunc_f64_s
	OpI32TruncF64U      Opcode = 0xab // i32.trunc_f64_u
	OpI64ExtendI32S     Opcode = 0xac // i64.extend_i32_s
	OpI64ExtendI32U     Opcode = 0xad // i64.extend_i32_u
	OpI64TruncF32S      Opcode = 0xae // i64.trunc_f32_s
	OpI64TruncF32U      Opcode = 0xaf // i64.trunc_f32_u
	OpI64TruncF64S      Opcode = 0xb0 // i64.trunc_f64_s
	OpI64TruncF64U      Opcode = 0xb1 // i64.trunc_f64_u
	OpF32ConvertI32S    Opcode = 0xb2 // f32.convert_i32_s
	OpF32ConvertI32U    Opcode = 0xb3 // f32.convert_i32_u
	OpF32ConvertI64S    Opcode = 0xb4 // f32.convert_i64_s
	OpF32ConvertI64U    Opcode = 0xb5 // f32.convert_i64_u
	OpF32DemoteF64      Opcode = 0xb6 // f32.demote_f64
	OpF64ConvertI32S    Opcode = 0xb7 // f64.convert_i32_s
	OpF64ConvertI32U    Opcode = 0xb8 // f64.convert_i32_u
	OpF64ConvertI64S    Opcode = 0xb9 // f64.convert_i64_s
	OpF64ConvertI64U    Opcode = 0xba // f64.convert_i64_u
	OpF64PromoteF32     Opcode = 0xbb // f64.promote_f32
	OpI32ReinterpretF32 Opcode = 0xbc // i32.reinterpret_f32
	OpI64ReinterpretF64 Opcode = 0xbd // i64.reinterpret_f64
	OpF32ReinterpretI32 Opcode = 0xbe // f32.reinterpret_i32
	OpF64ReinterpretI64 Opcode = 0xbf // f64.reinterpret_i64
)
