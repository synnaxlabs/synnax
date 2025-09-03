// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

// WASM binary format constants
const (
	MagicNumber   = 0x0061736d // \0asm (little endian)
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
