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
	"fmt"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/telem"
)

func IRTypeToDataType(t ir.Type) telem.DataType {
	switch t.(type) {
	case ir.F64:
		return telem.Float64T
	case ir.F32:
		return telem.Float32T
	case ir.U64:
		return telem.Uint64T
	case ir.U32:
		return telem.Uint32T
	case ir.U16:
		return telem.Uint16T
	case ir.U8:
		return telem.Uint8T
	case ir.String:
		return telem.StringT
	case ir.TimeStamp:
		return telem.TimeStampT
	case ir.I64:
		return telem.Int64T
	case ir.I32:
		return telem.Int32T
	case ir.I16:
		return telem.Int16T
	case ir.I8:
		return telem.Int8T
	default:
		panic(fmt.Sprintf("IRTypeToDataType: unsupported type %T (%v)", t, t))
	}
}
