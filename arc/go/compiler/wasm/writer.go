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

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/types"
	xbinary "github.com/synnaxlabs/x/binary"
)

// Writer handles low-level WASM instruction encoding
type Writer struct {
	buf bytes.Buffer
}

// NewWriter creates a new WASM encoder
func NewWriter() *Writer {
	return &Writer{}
}

// WriteOpcode writes a single WASM opcode
func (e *Writer) WriteOpcode(op Opcode) { e.buf.WriteByte(byte(op)) }

// WriteBytes writes raw bytes
func (e *Writer) WriteBytes(data []byte) { e.buf.Write(data) }

// WriteI32Const writes an i32.const instruction
func (e *Writer) WriteI32Const(val int32) {
	e.WriteOpcode(OpI32Const)
	e.WriteLEB128Signed(int64(val))
}

// WriteI64Const writes an i64.const instruction
func (e *Writer) WriteI64Const(val int64) {
	e.WriteOpcode(OpI64Const)
	e.WriteLEB128Signed(val)
}

// WriteF32Const writes a f32.const instruction
func (e *Writer) WriteF32Const(val float32) {
	e.WriteOpcode(OpF32Const)
	lo.Must0(binary.Write(&e.buf, binary.LittleEndian, math.Float32bits(val)))
}

// WriteF64Const writes an f64.const instruction
func (e *Writer) WriteF64Const(val float64) {
	e.WriteOpcode(OpF64Const)
	lo.Must0(binary.Write(&e.buf, binary.LittleEndian, math.Float64bits(val)))
}

// WriteLocalGet writes a local.get instruction
func (e *Writer) WriteLocalGet(idx int) {
	e.WriteOpcode(OpLocalGet)
	e.WriteLEB128Unsigned(uint64(idx))
}

// WriteLocalSet writes a local.set instruction
func (e *Writer) WriteLocalSet(idx int) {
	e.WriteOpcode(OpLocalSet)
	e.WriteLEB128Unsigned(uint64(idx))
}

// WriteLocalTee writes a local.tee instruction (set but keep on stack)
func (e *Writer) WriteLocalTee(idx int) {
	e.WriteOpcode(OpLocalTee)
	e.WriteLEB128Unsigned(uint64(idx))
}

// WriteGlobalGet writes a global.get instruction
func (e *Writer) WriteGlobalGet(idx int) {
	e.WriteOpcode(OpGlobalGet)
	e.WriteLEB128Unsigned(uint64(idx))
}

// WriteGlobalSet writes a global.set instruction
func (e *Writer) WriteGlobalSet(idx int) {
	e.WriteOpcode(OpGlobalSet)
	e.WriteLEB128Unsigned(uint64(idx))
}

// WriteCall writes a call instruction
func (e *Writer) WriteCall(funcIdx uint32) {
	e.WriteOpcode(OpCall)
	e.WriteLEB128Unsigned(uint64(funcIdx))
}

// WriteReturn writes a return instruction
func (e *Writer) WriteReturn() {
	e.WriteOpcode(OpReturn)
}

// WriteUnreachable writes an unreachable instruction
func (e *Writer) WriteUnreachable() {
	e.WriteOpcode(OpUnreachable)
}

// WriteEnd writes an end instruction
func (e *Writer) WriteEnd() {
	e.WriteOpcode(OpEnd)
}

// WriteIf writes an if instruction with optional result type
func (e *Writer) WriteIf(resultType BlockType) {
	e.WriteOpcode(OpIf)
	e.writeBlockType(resultType)
}

// WriteElse writes an else instruction
func (e *Writer) WriteElse() {
	e.WriteOpcode(OpElse)
}

// WriteBlock writes a block instruction
func (e *Writer) WriteBlock(resultType BlockType) {
	e.WriteOpcode(OpBlock)
	e.writeBlockType(resultType)
}

// WriteLoop writes a loop instruction
func (e *Writer) WriteLoop(resultType BlockType) {
	e.WriteOpcode(OpLoop)
	e.writeBlockType(resultType)
}

// WriteBr writes a br (branch) instruction
func (e *Writer) WriteBr(labelIdx uint32) {
	e.WriteOpcode(OpBr)
	e.WriteLEB128Unsigned(uint64(labelIdx))
}

// WriteBrIf writes a br_if (conditional branch) instruction
func (e *Writer) WriteBrIf(labelIdx uint32) {
	e.WriteOpcode(OpBrIf)
	e.WriteLEB128Unsigned(uint64(labelIdx))
}

func (e *Writer) WriteBinaryOpInferred(op string, resultType types.Type) error {
	opcode, err := binaryOpcode(op, resultType)
	if err != nil {
		return err
	}
	e.WriteBinaryOp(opcode)
	return nil
}

// WriteBinaryOp writes a simple binary operation (no immediates)
func (e *Writer) WriteBinaryOp(op Opcode) {
	e.WriteOpcode(op)
}

// WriteUnaryOp writes a simple unary operation (no immediates)
func (e *Writer) WriteUnaryOp(op Opcode) {
	e.WriteOpcode(op)
}

// WriteI32Eqz writes an i32.eqz instruction (returns 1 if value is 0, else 0)
func (e *Writer) WriteI32Eqz() {
	e.WriteOpcode(OpI32Eqz)
}

// WriteMemoryOp writes a memory operation with alignment and offset
func (e *Writer) WriteMemoryOp(op Opcode, align, offset uint32) {
	e.WriteOpcode(op)
	e.WriteLEB128Unsigned(uint64(align))
	e.WriteLEB128Unsigned(uint64(offset))
}

// writeBlockType writes a block type (for if/block/loop)
func (e *Writer) writeBlockType(bt BlockType) {
	if bt.empty {
		e.buf.WriteByte(0x40)
		return
	}
	e.buf.WriteByte(byte(bt.valueType))
}

// WriteLEB128Unsigned writes an unsigned LEB128 encoded integer
func (e *Writer) WriteLEB128Unsigned(val uint64) {
	xbinary.WriteLEB128Unsigned(&e.buf, val)
}

// WriteLEB128Signed writes a signed LEB128 encoded integer
func (e *Writer) WriteLEB128Signed(val int64) {
	xbinary.WriteLEB128Signed(&e.buf, val)
}

// Bytes returns the accumulated bytecode
func (e *Writer) Bytes() []byte {
	return e.buf.Bytes()
}

// Len returns the current bytecode length
func (e *Writer) Len() int {
	return e.buf.Len()
}

// Reset clears the bytecode buffer
func (e *Writer) Reset() {
	e.buf.Reset()
}

// BlockType represents the type signature of a block
type BlockType struct {
	empty     bool
	valueType ValueType
}

var (
	BlockTypeEmpty = BlockType{empty: true}
	BlockTypeI32   = BlockType{valueType: I32}
	BlockTypeI64   = BlockType{valueType: I64}
	BlockTypeF32   = BlockType{valueType: F32}
	BlockTypeF64   = BlockType{valueType: F64}
)
