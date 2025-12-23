// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package bindings manages the contract between compiled Arc WebAssembly modules and
// the host runtime. It tracks import indices for host functions that provide channel
// operations, series manipulation, state persistence, and other runtime services.
//
// The ImportIndex maintains type-specific mappings for all host functions, ensuring
// type-safe communication between Arc compiled code and the runtime environment.
package bindings

import (
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// lookupImport is a generic helper for looking up import indices by type
func (idx *ImportIndex) lookupImport(
	m map[string]uint32,
	t types.Type,
	funcName string,
) (uint32, error) {
	suffix := t.Unwrap().String()
	if funcIdx, ok := m[suffix]; ok {
		return funcIdx, nil
	}
	return 0, errors.Newf("no %s function for type %v", funcName, t)
}

// GetChannelRead returns the import index for a channel read function
func (idx *ImportIndex) GetChannelRead(t types.Type) (uint32, error) {
	return idx.lookupImport(idx.ChannelRead, t, "channel read")
}

// GetChannelWrite returns the import index for a channel write function
func (idx *ImportIndex) GetChannelWrite(t types.Type) (uint32, error) {
	return idx.lookupImport(idx.ChannelWrite, t, "channel write")
}

// GetChannelBlockingRead returns the import index for a blocking channel read function
func (idx *ImportIndex) GetChannelBlockingRead(t types.Type) (uint32, error) {
	return idx.lookupImport(idx.ChannelBlockingRead, t, "channel blocking read")
}

// GetSeriesCreateEmpty returns the import index for creating an empty series
func (idx *ImportIndex) GetSeriesCreateEmpty(t types.Type) (uint32, error) {
	return idx.lookupImport(idx.SeriesCreateEmpty, t, "series create")
}

// GetSeriesIndex returns the import index for series indexing
func (idx *ImportIndex) GetSeriesIndex(t types.Type) (uint32, error) {
	return idx.lookupImport(idx.SeriesIndex, t, "series index")
}

// GetSeriesSetElement returns the import index for setting a series element
func (idx *ImportIndex) GetSeriesSetElement(t types.Type) (uint32, error) {
	return idx.lookupImport(idx.SeriesSetElement, t, "series set element")
}

// GetSeriesArithmetic returns the import index for series arithmetic operations
func (idx *ImportIndex) GetSeriesArithmetic(op string, t types.Type, isScalar bool) (uint32, error) {
	suffix := t.Unwrap().String()

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
	suffix := t.Unwrap().String()

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
	return idx.lookupImport(idx.StateLoad, t, "state load")
}

// GetStateStore returns the import index for a state store function
func (idx *ImportIndex) GetStateStore(t types.Type) (uint32, error) {
	return idx.lookupImport(idx.StateStore, t, "state store")
}

// GetStateLoadSeries returns the import index for loading a series from state.
// Unlike GetStateLoad, this takes the element type directly since series state
// operations are keyed by element type (e.g., "f64" not "series f64").
func (idx *ImportIndex) GetStateLoadSeries(elemType types.Type) (uint32, error) {
	suffix := elemType.String()
	if funcIdx, ok := idx.StateLoadSeries[suffix]; ok {
		return funcIdx, nil
	}
	return 0, errors.Newf("no series state load function for element type %v", elemType)
}

// GetStateStoreSeries returns the import index for storing a series to state.
// Unlike GetStateStore, this takes the element type directly since series state
// operations are keyed by element type (e.g., "f64" not "series f64").
func (idx *ImportIndex) GetStateStoreSeries(elemType types.Type) (uint32, error) {
	suffix := elemType.String()
	if funcIdx, ok := idx.StateStoreSeries[suffix]; ok {
		return funcIdx, nil
	}
	return 0, errors.Newf("no series state store function for element type %v", elemType)
}
