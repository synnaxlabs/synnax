// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import "github.com/synnaxlabs/arc/types"

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

func ConvertType(t types.Type) ValueType {
	switch t {
	case types.I8{}, types.I16{}, types.I32{}, types.U8{}, types.U16{}, types.U32{}:
		return I32
	case types.I64{}, types.U64{}:
		return I64
	case types.F32{}:
		return F32
	case types.F64{}:
		return F64
	default:
		return I32
	}
}
