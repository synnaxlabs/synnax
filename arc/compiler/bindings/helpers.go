// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bindings

import (
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// GetChannelRead returns the import index for a channel read function
func (idx *ImportIndex) GetChannelRead(t types.Type) (uint32, error) {
	suffix := getTypeSuffix(t)
	if funcIdx, ok := idx.ChannelRead[suffix]; ok {
		return funcIdx, nil
	}
	return 0, errors.Newf("no channel read function for type %v", t)
}

// GetChannelWrite returns the import index for a channel write function
func (idx *ImportIndex) GetChannelWrite(t types.Type) (uint32, error) {
	suffix := getTypeSuffix(t)
	if funcIdx, ok := idx.ChannelWrite[suffix]; ok {
		return funcIdx, nil
	}
	return 0, errors.Newf("no channel write function for type %v", t)
}

// GetChannelBlockingRead returns the import index for a blocking channel read function
func (idx *ImportIndex) GetChannelBlockingRead(t types.Type) (uint32, error) {
	suffix := getTypeSuffix(t)
	if funcIdx, ok := idx.ChannelBlockingRead[suffix]; ok {
		return funcIdx, nil
	}
	return 0, errors.Newf("no channel blocking read function for type %v", t)
}

// GetSeriesCreateEmpty returns the import index for creating an empty series
func (idx *ImportIndex) GetSeriesCreateEmpty(t types.Type) (uint32, error) {
	suffix := getTypeSuffix(t)
	if funcIdx, ok := idx.SeriesCreateEmpty[suffix]; ok {
		return funcIdx, nil
	}
	return 0, errors.Newf("no series create function for type %v", t)
}

// GetSeriesIndex returns the import index for series indexing
func (idx *ImportIndex) GetSeriesIndex(t types.Type) (uint32, error) {
	suffix := getTypeSuffix(t)
	if funcIdx, ok := idx.SeriesIndex[suffix]; ok {
		return funcIdx, nil
	}
	return 0, errors.Newf("no series index function for type %v", t)
}

// GetSeriesArithmetic returns the import index for series arithmetic operations
func (idx *ImportIndex) GetSeriesArithmetic(op string, t types.Type, isScalar bool) (uint32, error) {
	suffix := getTypeSuffix(t)

	var m map[string]uint32
	if isScalar {
		switch op {
		case "+":
			m = idx.SeriesElementAdd
		case "-":
			m = idx.SeriesElementSub
		case "*":
			m = idx.SeriesElementMul
		case "/":
			m = idx.SeriesElementDiv
		default:
			return 0, errors.Newf("unknown arithmetic operator: %s", op)
		}
	} else {
		switch op {
		case "+":
			m = idx.SeriesSeriesAdd
		case "-":
			m = idx.SeriesSeriesSub
		case "*":
			m = idx.SeriesSeriesMul
		case "/":
			m = idx.SeriesSeriesDiv
		default:
			return 0, errors.Newf("unknown arithmetic operator: %s", op)
		}
	}

	if funcIdx, ok := m[suffix]; ok {
		return funcIdx, nil
	}
	return 0, errors.Newf("no series %s function for type %v", op, t)
}

// GetSeriesComparison returns the import index for series comparison operations
func (idx *ImportIndex) GetSeriesComparison(op string, t types.Type) (uint32, error) {
	suffix := getTypeSuffix(t)

	var m map[string]uint32
	switch op {
	case ">":
		m = idx.SeriesCompareGT
	case "<":
		m = idx.SeriesCompareLT
	case ">=":
		m = idx.SeriesCompareGE
	case "<=":
		m = idx.SeriesCompareLE
	case "==":
		m = idx.SeriesCompareEQ
	case "!=":
		m = idx.SeriesCompareNE
	default:
		return 0, errors.Newf("unknown comparison operator: %s", op)
	}

	if funcIdx, ok := m[suffix]; ok {
		return funcIdx, nil
	}
	return 0, errors.Newf("no series comparison %s function for type %v", op, t)
}

// GetStateLoad returns the import index for a state load function
func (idx *ImportIndex) GetStateLoad(t types.Type) (uint32, error) {
	suffix := getTypeSuffix(t)
	if funcIdx, ok := idx.StateLoad[suffix]; ok {
		return funcIdx, nil
	}
	return 0, errors.Newf("no state load function for type %v", t)
}

// GetStateStore returns the import index for a state store function
func (idx *ImportIndex) GetStateStore(t types.Type) (uint32, error) {
	suffix := getTypeSuffix(t)
	if funcIdx, ok := idx.StateStore[suffix]; ok {
		return funcIdx, nil
	}
	return 0, errors.Newf("no state store function for type %v", t)
}

// getTypeSuffix extracts the type suffix for import lookups
func getTypeSuffix(t types.Type) string {
	switch t := t.(type) {
	case types.I8:
		return "i8"
	case types.I16:
		return "i16"
	case types.I32:
		return "i32"
	case types.I64:
		return "i64"
	case types.U8:
		return "u8"
	case types.U16:
		return "u16"
	case types.U32:
		return "u32"
	case types.U64:
		return "u64"
	case types.F32:
		return "f32"
	case types.F64:
		return "f64"
	case types.String:
		return "string"
	case types.TimeStamp, types.TimeSpan:
		return "i64"
	case types.Series:
		// For series, we need the element type
		return getTypeSuffix(t.ValueType)
	case types.Chan:
		// For channels, we need the element type
		return getTypeSuffix(t.ValueType)
	default:
		// Default fallback
		return "i32"
	}
}
