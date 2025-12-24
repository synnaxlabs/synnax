// Copyright 2025 Synnax Labs, Inc.
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
	logicalOps    = map[string]op.Binary{orSymbolName: op.OrU8, andSymbolName: op.AndU8}
	unaryOps      = map[string]op.Unary{notSymbolName: op.NotU8}
	typedUnaryOps = map[string]map[telem.DataType]op.Unary{
		negSymbolName: {
			telem.Float64T: op.NegateF64,
			telem.Float32T: op.NegateF32,
			telem.Int64T:   op.NegateI64,
			telem.Int32T:   op.NegateI32,
			telem.Int16T:   op.NegateI16,
			telem.Int8T:    op.NegateI8,
		},
	}
	typedOps = map[string]map[telem.DataType]op.Binary{
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
		addSymbolName: {
			telem.Float64T: op.AddF64,
			telem.Float32T: op.AddF32,
			telem.Int64T:   op.AddI64,
			telem.Int32T:   op.AddI32,
			telem.Int16T:   op.AddI16,
			telem.Int8T:    op.AddI8,
			telem.Uint64T:  op.AddU64,
			telem.Uint32T:  op.AddU32,
			telem.Uint16T:  op.AddU16,
			telem.Uint8T:   op.AddU8,
		},
		subSymbolName: {
			telem.Float64T: op.SubtractF64,
			telem.Float32T: op.SubtractF32,
			telem.Int64T:   op.SubtractI64,
			telem.Int32T:   op.SubtractI32,
			telem.Int16T:   op.SubtractI16,
			telem.Int8T:    op.SubtractI8,
			telem.Uint64T:  op.SubtractU64,
			telem.Uint32T:  op.SubtractU32,
			telem.Uint16T:  op.SubtractU16,
			telem.Uint8T:   op.SubtractU8,
		},
		mulSymbolName: {
			telem.Float64T: op.MultiplyF64,
			telem.Float32T: op.MultiplyF32,
			telem.Int64T:   op.MultiplyI64,
			telem.Int32T:   op.MultiplyI32,
			telem.Int16T:   op.MultiplyI16,
			telem.Int8T:    op.MultiplyI8,
			telem.Uint64T:  op.MultiplyU64,
			telem.Uint32T:  op.MultiplyU32,
			telem.Uint16T:  op.MultiplyU16,
			telem.Uint8T:   op.MultiplyU8,
		},
		divSymbolName: {
			telem.Float64T: op.DivideF64,
			telem.Float32T: op.DivideF32,
			telem.Int64T:   op.DivideI64,
			telem.Int32T:   op.DivideI32,
			telem.Int16T:   op.DivideI16,
			telem.Int8T:    op.DivideI8,
			telem.Uint64T:  op.DivideU64,
			telem.Uint32T:  op.DivideU32,
			telem.Uint16T:  op.DivideU16,
			telem.Uint8T:   op.DivideU8,
		},
		modSymbolName: {
			telem.Float64T: op.ModuloF64,
			telem.Float32T: op.ModuloF32,
			telem.Int64T:   op.ModuloI64,
			telem.Int32T:   op.ModuloI32,
			telem.Int16T:   op.ModuloI16,
			telem.Int8T:    op.ModuloI8,
			telem.Uint64T:  op.ModuloU64,
			telem.Uint32T:  op.ModuloU32,
			telem.Uint16T:  op.ModuloU16,
			telem.Uint8T:   op.ModuloU8,
		},
	}
)
