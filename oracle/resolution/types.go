// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolution

// ValueKind identifies the type of an expression value.
type ValueKind int

const (
	ValueKindString ValueKind = iota
	ValueKindInt
	ValueKindFloat
	ValueKindBool
	ValueKindIdent
)

// Primitives is the set of built-in primitive type names recognized by Oracle.
var Primitives = map[string]bool{
	"uuid": true, "string": true, "bool": true,
	"int8": true, "int16": true, "int32": true, "int64": true,
	"uint8": true, "uint12": true, "uint16": true, "uint20": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
	"timestamp": true, "timespan": true, "time_range": true, "time_range_bounded": true,
	"json": true, "bytes": true, "data_type": true, "color": true,
}

// IsPrimitive returns true if the name is a built-in primitive type.
func IsPrimitive(name string) bool { return Primitives[name] }

// StringPrimitives identifies primitives that map to string-like types.
var StringPrimitives = map[string]bool{"string": true, "uuid": true}

// NumberPrimitives identifies primitives that map to number types.
var NumberPrimitives = map[string]bool{
	"int8": true, "int16": true, "int32": true, "int64": true,
	"uint8": true, "uint12": true, "uint16": true, "uint20": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
}

// IsStringPrimitive checks if the primitive is a string-like type.
func IsStringPrimitive(name string) bool { return StringPrimitives[name] }

// IsNumberPrimitive checks if the primitive is a number type.
func IsNumberPrimitive(name string) bool { return NumberPrimitives[name] }
