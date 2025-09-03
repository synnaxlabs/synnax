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
	"bytes"
	"encoding/binary"
	"math"
)

// Encoder handles low-level WASM instruction encoding
type Encoder struct {
	buf bytes.Buffer
}

// NewEncoder creates a new WASM encoder
func NewEncoder() *Encoder {
	return &Encoder{}
}

// WriteOpcode writes a single WASM opcode
func (e *Encoder) WriteOpcode(op Opcode) { e.buf.WriteByte(byte(op)) }

// WriteBytes writes raw bytes
func (e *Encoder) WriteBytes(data []byte) { e.buf.Write(data) }

// WriteI32Const writes an i32.const instruction
func (e *Encoder) WriteI32Const(val int32) {
	e.WriteOpcode(OpI32Const)
	e.WriteLEB128Signed(int64(val))
}

// WriteI64Const writes an i64.const instruction
func (e *Encoder) WriteI64Const(val int64) {
	e.WriteOpcode(OpI64Const)
	e.WriteLEB128Signed(val)
}

// WriteF32Const writes a f32.const instruction
func (e *Encoder) WriteF32Const(val float32) {
	e.WriteOpcode(OpF32Const)
	bits := math.Float32bits(val)
	binary.Write(&e.buf, binary.LittleEndian, bits)
}

// WriteF64Const writes an f64.const instruction
func (e *Encoder) WriteF64Const(val float64) {
	e.WriteOpcode(OpF64Const)
	bits := math.Float64bits(val)
	binary.Write(&e.buf, binary.LittleEndian, bits)
}

// WriteLocalGet writes a local.get instruction
func (e *Encoder) WriteLocalGet(idx uint32) {
	e.WriteOpcode(OpLocalGet)
	e.WriteLEB128Unsigned(uint64(idx))
}

// WriteLocalSet writes a local.set instruction
func (e *Encoder) WriteLocalSet(idx uint32) {
	e.WriteOpcode(OpLocalSet)
	e.WriteLEB128Unsigned(uint64(idx))
}

// WriteLocalTee writes a local.tee instruction (set but keep on stack)
func (e *Encoder) WriteLocalTee(idx uint32) {
	e.WriteOpcode(OpLocalTee)
	e.WriteLEB128Unsigned(uint64(idx))
}

// WriteGlobalGet writes a global.get instruction
func (e *Encoder) WriteGlobalGet(idx uint32) {
	e.WriteOpcode(OpGlobalGet)
	e.WriteLEB128Unsigned(uint64(idx))
}

// WriteGlobalSet writes a global.set instruction
func (e *Encoder) WriteGlobalSet(idx uint32) {
	e.WriteOpcode(OpGlobalSet)
	e.WriteLEB128Unsigned(uint64(idx))
}

// WriteCall writes a call instruction
func (e *Encoder) WriteCall(funcIdx uint32) {
	e.WriteOpcode(OpCall)
	e.WriteLEB128Unsigned(uint64(funcIdx))
}

// WriteReturn writes a return instruction
func (e *Encoder) WriteReturn() {
	e.WriteOpcode(OpReturn)
}

// WriteEnd writes an end instruction
func (e *Encoder) WriteEnd() {
	e.WriteOpcode(OpEnd)
}

// WriteIf writes an if instruction with optional result type
func (e *Encoder) WriteIf(resultType BlockType) {
	e.WriteOpcode(OpIf)
	e.writeBlockType(resultType)
}

// WriteElse writes an else instruction
func (e *Encoder) WriteElse() {
	e.WriteOpcode(OpElse)
}

// WriteBlock writes a block instruction
func (e *Encoder) WriteBlock(resultType BlockType) {
	e.WriteOpcode(OpBlock)
	e.writeBlockType(resultType)
}

// WriteLoop writes a loop instruction
func (e *Encoder) WriteLoop(resultType BlockType) {
	e.WriteOpcode(OpLoop)
	e.writeBlockType(resultType)
}

// WriteBr writes a br (branch) instruction
func (e *Encoder) WriteBr(labelIdx uint32) {
	e.WriteOpcode(OpBr)
	e.WriteLEB128Unsigned(uint64(labelIdx))
}

// WriteBrIf writes a br_if (conditional branch) instruction
func (e *Encoder) WriteBrIf(labelIdx uint32) {
	e.WriteOpcode(OpBrIf)
	e.WriteLEB128Unsigned(uint64(labelIdx))
}

// === Arithmetic Instructions ===

// WriteBinaryOp writes a simple binary operation (no immediates)
func (e *Encoder) WriteBinaryOp(op Opcode) {
	e.WriteOpcode(op)
}

// WriteUnaryOp writes a simple unary operation (no immediates)
func (e *Encoder) WriteUnaryOp(op Opcode) {
	e.WriteOpcode(op)
}

// WriteMemoryOp writes a memory operation with alignment and offset
func (e *Encoder) WriteMemoryOp(op Opcode, align, offset uint32) {
	e.WriteOpcode(op)
	e.WriteLEB128Unsigned(uint64(align))
	e.WriteLEB128Unsigned(uint64(offset))
}

// === Helper Methods ===

// writeBlockType writes a block type (for if/block/loop)
func (e *Encoder) writeBlockType(bt BlockType) {
	switch bt := bt.(type) {
	case EmptyBlockType:
		e.buf.WriteByte(0x40) // empty type
	case ValueBlockType:
		e.buf.WriteByte(byte(bt.Type))
	default:
		// For more complex block types (multi-value), we'd need type indices
		e.buf.WriteByte(0x40) // default to empty
	}
}

// === LEB128 Encoding ===

// WriteLEB128Unsigned writes an unsigned LEB128 encoded integer
func (e *Encoder) WriteLEB128Unsigned(val uint64) {
	for {
		b := byte(val & 0x7f)
		val >>= 7
		if val != 0 {
			b |= 0x80
		}
		e.buf.WriteByte(b)
		if val == 0 {
			break
		}
	}
}

// WriteLEB128Signed writes a signed LEB128 encoded integer
func (e *Encoder) WriteLEB128Signed(val int64) {
	for {
		b := byte(val & 0x7f)
		val >>= 7

		// Sign bit of byte is second high order bit
		signBit := (b & 0x40) != 0

		if (val == 0 && !signBit) || (val == -1 && signBit) {
			e.buf.WriteByte(b)
			break
		}

		e.buf.WriteByte(b | 0x80)
	}
}

// === Output Methods ===

// Bytes returns the accumulated bytecode
func (e *Encoder) Bytes() []byte {
	return e.buf.Bytes()
}

// Len returns the current bytecode length
func (e *Encoder) Len() int {
	return e.buf.Len()
}

// Reset clears the bytecode buffer
func (e *Encoder) Reset() {
	e.buf.Reset()
}

// === Block Types ===

// BlockType represents the type signature of a block
type BlockType interface {
	isBlockType()
}

// EmptyBlockType represents a block with no result
type EmptyBlockType struct{}

func (EmptyBlockType) isBlockType() {}

// ValueBlockType represents a block with a single result type
type ValueBlockType struct {
	Type ValueType
}

func (ValueBlockType) isBlockType() {}

// Helper constructors for block types
var (
	BlockTypeEmpty = EmptyBlockType{}
	BlockTypeI32   = ValueBlockType{I32}
	BlockTypeI64   = ValueBlockType{I64}
	BlockTypeF32   = ValueBlockType{F32}
	BlockTypeF64   = ValueBlockType{F64}
)
