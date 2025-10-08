// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package util

import (
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/telem"
)

func DataTypeToIRType(dt telem.DataType) ir.Type {
	switch dt {
	case telem.Uint64T:
		return ir.U64{}
	case telem.Uint32T:
		return ir.U32{}
	case telem.Uint16T:
		return ir.U16{}
	case telem.Uint8T:
		return ir.U8{}
	case telem.Int64T:
		return ir.I64{}
	case telem.Int32T:
		return ir.I32{}
	case telem.Int16T:
		return ir.I16{}
	case telem.Int8T:
		return ir.I8{}
	case telem.Float64T:
		return ir.F64{}
	case telem.Float32T:
		return ir.F32{}
	case telem.StringT:
		return ir.String{}
	case telem.TimeStampT:
		return ir.TimeStamp{}
	default:
		// For unknown types, return nil (untyped)
		// The actual type should be determined by context
		return nil
	}
}

func IRTypeToDataType(t ir.Type) telem.DataType {
	switch t.(type) {
	case ir.F64:
		return telem.Int64T
	default:
		panic("bad")
	}
}
