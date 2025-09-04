// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import "fmt"

// ImportIndex tracks the indices of imported host functions.
type ImportIndex struct {
	// Channel operations
	ChannelRead         map[string]uint32 // type suffix -> function index
	ChannelWrite        map[string]uint32
	ChannelBlockingRead map[string]uint32

	// Series operations
	SeriesCreateEmpty map[string]uint32
	SeriesSetElement  map[string]uint32
	SeriesIndex       map[string]uint32
	SeriesLen         uint32
	SeriesSlice       uint32

	// Series arithmetic (per type)
	SeriesElementAdd map[string]uint32
	SeriesElementMul map[string]uint32
	SeriesSeriesAdd  map[string]uint32
	SeriesSeriesMul  map[string]uint32

	// State operations
	StateLoad  map[string]uint32
	StateStore map[string]uint32

	// String operations
	StringFromLiteral uint32
	StringLen         uint32

	// Built-in functions
	Now uint32

	// Error handling
	Panic uint32
}

// SetupTypedImports adds all typed host function imports to the module.
func SetupTypedImports(m *Module) *ImportIndex {
	idx := &ImportIndex{
		ChannelRead:         make(map[string]uint32),
		ChannelWrite:        make(map[string]uint32),
		ChannelBlockingRead: make(map[string]uint32),
		SeriesCreateEmpty:   make(map[string]uint32),
		SeriesSetElement:    make(map[string]uint32),
		SeriesIndex:         make(map[string]uint32),
		SeriesElementAdd:    make(map[string]uint32),
		SeriesElementMul:    make(map[string]uint32),
		SeriesSeriesAdd:     make(map[string]uint32),
		SeriesSeriesMul:     make(map[string]uint32),
		StateLoad:           make(map[string]uint32),
		StateStore:          make(map[string]uint32),
	}

	// Define the numeric types we support
	intTypes := []string{"i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64"}
	floatTypes := []string{"f32", "f64"}
	allNumericTypes := append(intTypes, floatTypes...)

	// AddSymbol channel read operations
	for _, typ := range allNumericTypes {
		funcName := fmt.Sprintf("channel_read_%s", typ)
		resultType := getWASMType(typ)
		idx.ChannelRead[typ] = m.AddImport("env", funcName, FunctionType{
			Params:  []ValueType{I32}, // channel ID
			Results: []ValueType{resultType},
		})
	}
	idx.ChannelRead["string"] = m.AddImport("env", "channel_read_string", FunctionType{
		Params:  []ValueType{I32},
		Results: []ValueType{I32}, // string handle
	})

	// AddSymbol channel write operations
	for _, typ := range allNumericTypes {
		funcName := fmt.Sprintf("channel_write_%s", typ)
		paramType := getWASMType(typ)
		idx.ChannelWrite[typ] = m.AddImport("env", funcName, FunctionType{
			Params:  []ValueType{I32, paramType}, // channel ID, value
			Results: []ValueType{},
		})
	}
	idx.ChannelWrite["string"] = m.AddImport("env", "channel_write_string", FunctionType{
		Params:  []ValueType{I32, I32}, // channel ID, string handle
		Results: []ValueType{},
	})

	// AddSymbol blocking read operations
	for _, typ := range allNumericTypes {
		funcName := fmt.Sprintf("channel_blocking_read_%s", typ)
		resultType := getWASMType(typ)
		idx.ChannelBlockingRead[typ] = m.AddImport("env", funcName, FunctionType{
			Params:  []ValueType{I32}, // channel ID
			Results: []ValueType{resultType},
		})
	}
	idx.ChannelBlockingRead["string"] = m.AddImport("env", "channel_blocking_read_string", FunctionType{
		Params:  []ValueType{I32},
		Results: []ValueType{I32}, // string handle
	})

	// AddSymbol series operations
	for _, typ := range allNumericTypes {
		// Create empty series
		funcName := fmt.Sprintf("series_create_empty_%s", typ)
		idx.SeriesCreateEmpty[typ] = m.AddImport("env", funcName, FunctionType{
			Params:  []ValueType{I32}, // length
			Results: []ValueType{I32}, // series handle
		})

		// Set element
		funcName = fmt.Sprintf("series_set_element_%s", typ)
		valueType := getWASMType(typ)
		idx.SeriesSetElement[typ] = m.AddImport("env", funcName, FunctionType{
			Params:  []ValueType{I32, I32, valueType}, // series, index, value
			Results: []ValueType{},
		})

		// Get element
		funcName = fmt.Sprintf("series_index_%s", typ)
		idx.SeriesIndex[typ] = m.AddImport("env", funcName, FunctionType{
			Params:  []ValueType{I32, I32}, // series, index
			Results: []ValueType{valueType},
		})

		// Arithmetic operations (only for numeric types)
		funcName = fmt.Sprintf("series_element_add_%s", typ)
		idx.SeriesElementAdd[typ] = m.AddImport("env", funcName, FunctionType{
			Params:  []ValueType{I32, valueType}, // series, scalar
			Results: []ValueType{I32},            // new series
		})

		funcName = fmt.Sprintf("series_element_mul_%s", typ)
		idx.SeriesElementMul[typ] = m.AddImport("env", funcName, FunctionType{
			Params:  []ValueType{I32, valueType}, // series, scalar
			Results: []ValueType{I32},            // new series
		})

		funcName = fmt.Sprintf("series_series_add_%s", typ)
		idx.SeriesSeriesAdd[typ] = m.AddImport("env", funcName, FunctionType{
			Params:  []ValueType{I32, I32}, // series1, series2
			Results: []ValueType{I32},      // new series
		})

		funcName = fmt.Sprintf("series_series_mul_%s", typ)
		idx.SeriesSeriesMul[typ] = m.AddImport("env", funcName, FunctionType{
			Params:  []ValueType{I32, I32}, // series1, series2
			Results: []ValueType{I32},      // new series
		})
	}

	// Type-agnostic series operations
	idx.SeriesLen = m.AddImport("env", "series_len", FunctionType{
		Params:  []ValueType{I32}, // series handle
		Results: []ValueType{I64}, // length
	})

	idx.SeriesSlice = m.AddImport("env", "series_slice", FunctionType{
		Params:  []ValueType{I32, I32, I32}, // series, start, end
		Results: []ValueType{I32},           // new series handle
	})

	// State persistence operations
	for _, typ := range allNumericTypes {
		// Load
		funcName := fmt.Sprintf("state_load_%s", typ)
		resultType := getWASMType(typ)
		idx.StateLoad[typ] = m.AddImport("env", funcName, FunctionType{
			Params:  []ValueType{I32, I32}, // task ID, var ID
			Results: []ValueType{resultType},
		})

		// Store
		funcName = fmt.Sprintf("state_store_%s", typ)
		valueType := getWASMType(typ)
		idx.StateStore[typ] = m.AddImport("env", funcName, FunctionType{
			Params:  []ValueType{I32, I32, valueType}, // task ID, var ID, value
			Results: []ValueType{},
		})
	}
	idx.StateLoad["string"] = m.AddImport("env", "state_load_string", FunctionType{
		Params:  []ValueType{I32, I32},
		Results: []ValueType{I32}, // string handle
	})
	idx.StateStore["string"] = m.AddImport("env", "state_store_string", FunctionType{
		Params:  []ValueType{I32, I32, I32}, // task ID, var ID, handle
		Results: []ValueType{},
	})

	// String operations
	idx.StringFromLiteral = m.AddImport("env", "string_from_literal", FunctionType{
		Params:  []ValueType{I32, I32}, // ptr, len
		Results: []ValueType{I32},      // string handle
	})

	idx.StringLen = m.AddImport("env", "string_len", FunctionType{
		Params:  []ValueType{I32}, // string handle
		Results: []ValueType{I32}, // length
	})

	// Built-in functions
	idx.Now = m.AddImport("env", "now", FunctionType{
		Params:  []ValueType{},
		Results: []ValueType{I64}, // timestamp
	})

	// Error handling
	idx.Panic = m.AddImport("env", "panic", FunctionType{
		Params:  []ValueType{I32, I32}, // ptr, len
		Results: []ValueType{},
	})

	return idx
}

// getWASMType returns the WASM type for a Slate type suffix.
func getWASMType(typeSuffix string) ValueType {
	switch typeSuffix {
	case "i8", "i16", "i32", "u8", "u16", "u32":
		return I32
	case "i64", "u64":
		return I64
	case "f32":
		return F32
	case "f64":
		return F64
	default:
		return I32 // Default for handles
	}
}

// GetChannelRead returns the import index for a channel read function
func (idx *ImportIndex) GetChannelRead(t interface{}) uint32 {
	suffix := getTypeSuffix(t)
	if funcIdx, ok := idx.ChannelRead[suffix]; ok {
		return funcIdx
	}
	panic(fmt.Sprintf("no channel read function for type %v", t))
}

// GetChannelWrite returns the import index for a channel write function
func (idx *ImportIndex) GetChannelWrite(t interface{}) uint32 {
	suffix := getTypeSuffix(t)
	if funcIdx, ok := idx.ChannelWrite[suffix]; ok {
		return funcIdx
	}
	panic(fmt.Sprintf("no channel write function for type %v", t))
}

// GetStateLoad returns the import index for a state load function
func (idx *ImportIndex) GetStateLoad(t interface{}) uint32 {
	suffix := getTypeSuffix(t)
	if funcIdx, ok := idx.StateLoad[suffix]; ok {
		return funcIdx
	}
	panic(fmt.Sprintf("no state load function for type %v", t))
}

// GetStateStore returns the import index for a state store function
func (idx *ImportIndex) GetStateStore(t interface{}) uint32 {
	suffix := getTypeSuffix(t)
	if funcIdx, ok := idx.StateStore[suffix]; ok {
		return funcIdx
	}
	panic(fmt.Sprintf("no state store function for type %v", t))
}

// getTypeSuffix returns the type suffix string for import lookups
func getTypeSuffix(t interface{}) string {
	// If it's already a string, use it directly
	if s, ok := t.(string); ok {
		return s
	}
	
	// Otherwise get the string representation and extract the type name
	typeStr := fmt.Sprintf("%T", t)
	
	// Remove package prefix if present
	if idx := len("types."); idx < len(typeStr) && typeStr[:idx] == "types." {
		typeStr = typeStr[idx:]
	}
	
	// Remove struct suffix
	if len(typeStr) > 2 && typeStr[len(typeStr)-2:] == "{}" {
		typeStr = typeStr[:len(typeStr)-2]
	}
	
	// Convert to lowercase
	switch typeStr {
	case "I8":
		return "i8"
	case "I16":
		return "i16"
	case "I32":
		return "i32"
	case "I64":
		return "i64"
	case "U8":
		return "u8"
	case "U16":
		return "u16"
	case "U32":
		return "u32"
	case "U64":
		return "u64"
	case "F32":
		return "f32"
	case "F64":
		return "f64"
	case "String":
		return "string"
	case "TimeStamp", "TimeSpan":
		return "i64"
	default:
		// For channels, extract the element type
		if len(typeStr) > 5 && typeStr[:5] == "Chan[" {
			// Extract element type
			return "i32" // Placeholder - would need proper parsing
		}
		return "i32" // Default
	}
}
