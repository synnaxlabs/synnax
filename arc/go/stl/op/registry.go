// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package op

import (
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/op"
)

var (
	logicalOps = map[string]op.Binary{orSymbolName: op.OrU8, andSymbolName: op.AndU8}
	unaryOps   = map[string]op.Unary{notSymbolName: op.NotU8}
	typedOps   = map[string]map[telem.DataType]op.Binary{
		geSymbolName: {
			telem.Float64T: op.GreaterThanOrEqualF64,
			telem.Float32T: op.GreaterThanOrEqualF32,
			telem.Int64T:   op.GreaterThanOrEqualI64,
			telem.Int32T:   op.GreaterThanOrEqualI32,
			telem.Int16T:   op.GreaterThanOrEqualI16,
			telem.Int8T:    op.GreaterThanOrEqualI8,
			telem.Uint64T:  op.GreaterThanOrEqualU64,
			telem.Uint32T:  op.GreaterThanOrEqualU32,
			telem.Uint16T:  op.GreaterThanOrEqualU16,
			telem.Uint8T:   op.GreaterThanOrEqualU8,
		},
		gtSymbolName: {
			telem.Float64T: op.GreaterThanF64,
			telem.Float32T: op.GreaterThanF32,
			telem.Int64T:   op.GreaterThanI64,
			telem.Int32T:   op.GreaterThanI32,
			telem.Int16T:   op.GreaterThanI16,
			telem.Int8T:    op.GreaterThanI8,
			telem.Uint64T:  op.GreaterThanU64,
			telem.Uint32T:  op.GreaterThanU32,
			telem.Uint16T:  op.GreaterThanU16,
			telem.Uint8T:   op.GreaterThanU8,
		},
		leSymbolName: {
			telem.Float64T: op.LessThanOrEqualF64,
			telem.Float32T: op.LessThanOrEqualF32,
			telem.Int64T:   op.LessThanOrEqualI64,
			telem.Int32T:   op.LessThanOrEqualI32,
			telem.Int16T:   op.LessThanOrEqualI16,
			telem.Int8T:    op.LessThanOrEqualI8,
			telem.Uint64T:  op.LessThanOrEqualU64,
			telem.Uint32T:  op.LessThanOrEqualU32,
			telem.Uint16T:  op.LessThanOrEqualU16,
			telem.Uint8T:   op.LessThanOrEqualU8,
		},
		ltSymbolName: {
			telem.Float64T: op.LessThanF64,
			telem.Float32T: op.LessThanF32,
			telem.Int64T:   op.LessThanI64,
			telem.Int32T:   op.LessThanI32,
			telem.Int16T:   op.LessThanI16,
			telem.Int8T:    op.LessThanI8,
			telem.Uint64T:  op.LessThanU64,
			telem.Uint32T:  op.LessThanU32,
			telem.Uint16T:  op.LessThanU16,
			telem.Uint8T:   op.LessThanU8,
		},
		eqSymbolName: {
			telem.Float64T: op.EqualF64,
			telem.Float32T: op.EqualF32,
			telem.Int64T:   op.EqualI64,
			telem.Int32T:   op.EqualI32,
			telem.Int16T:   op.EqualI16,
			telem.Int8T:    op.EqualI8,
			telem.Uint64T:  op.EqualU64,
			telem.Uint32T:  op.EqualU32,
			telem.Uint16T:  op.EqualU16,
			telem.Uint8T:   op.EqualU8,
		},
		neSymbolName: {
			telem.Float64T: op.NotEqualF64,
			telem.Float32T: op.NotEqualF32,
			telem.Int64T:   op.NotEqualI64,
			telem.Int32T:   op.NotEqualI32,
			telem.Int16T:   op.NotEqualI16,
			telem.Int8T:    op.NotEqualI8,
			telem.Uint64T:  op.NotEqualU64,
			telem.Uint32T:  op.NotEqualU32,
			telem.Uint16T:  op.NotEqualU16,
			telem.Uint8T:   op.NotEqualU8,
		},
	}
)
