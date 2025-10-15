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

	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

func IRTypeToDataType(t types.Type) telem.DataType {
	switch t.(type) {
	case types.F64:
		return telem.Float64T
	case types.F32:
		return telem.Float32T
	case types.U64:
		return telem.Uint64T
	case types.U32:
		return telem.Uint32T
	case types.U16:
		return telem.Uint16T
	case types.U8:
		return telem.Uint8T
	case types.String:
		return telem.StringT
	case types.TimeStamp:
		return telem.TimeStampT
	case types.I64:
		return telem.Int64T
	case types.I32:
		return telem.Int32T
	case types.I16:
		return telem.Int16T
	case types.I8:
		return telem.Int8T
	default:
		panic(fmt.Sprintf("IRTypeToDataType: unsupported type %T (%v)", t, t))
	}
}
