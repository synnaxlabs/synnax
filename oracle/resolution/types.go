// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolution

import "github.com/synnaxlabs/x/set"

// ValueKind identifies the type of an expression value.
type ValueKind int

const (
	ValueKindString ValueKind = iota
	ValueKindInt
	ValueKindFloat
	ValueKindBool
	ValueKindIdent
	ValueKindArray
	ValueKindStruct
)

// primitives is the set of built-in primitive type names recognized by Oracle.
var primitives = set.New(
	"uuid", "string", "bool",
	"int8", "int16", "int32", "int64",
	"uint8", "uint12", "uint16", "uint20", "uint32", "uint64",
	"float32", "float64",
	"record", "bytes", "any", "nil",
)

// IsPrimitive returns true if the name is a built-in primitive type.
func IsPrimitive(name string) bool { return primitives.Contains(name) }

var stringPrimitives = set.New("string", "uuid")

var numberPrimitives = set.New(
	"int8", "int16", "int32", "int64",
	"uint8", "uint12", "uint16", "uint20", "uint32", "uint64",
	"float32", "float64",
)

// constraints is the set of built-in constraint-only type names recognized by Oracle.
// These are valid as type parameter constraints but cannot be used as field types.
var constraints = set.New("comparable", "numeric")

// IsConstraint returns true if the name is a built-in constraint-only type.
func IsConstraint(name string) bool { return constraints.Contains(name) }

// IsStringPrimitive checks if the primitive is a string-like type.
func IsStringPrimitive(name string) bool { return stringPrimitives.Contains(name) }

// IsNumberPrimitive checks if the primitive is a number type.
func IsNumberPrimitive(name string) bool { return numberPrimitives.Contains(name) }
