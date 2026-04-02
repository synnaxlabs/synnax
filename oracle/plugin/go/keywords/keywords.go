// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package keywords

import "github.com/synnaxlabs/x/set"

// Reserved contains Go reserved keywords and predeclared identifiers that cannot
// be used as variable names in generated code.
var Reserved = set.New(
	// Keywords
	"break", "case", "chan", "const", "continue",
	"default", "defer", "else", "fallthrough", "for",
	"func", "go", "goto", "if", "import",
	"interface", "map", "package", "range", "return",
	"select", "struct", "switch", "type", "var",
	// Predeclared types
	"bool", "byte", "complex64", "complex128", "error",
	"float32", "float64", "int", "int8", "int16",
	"int32", "int64", "rune", "string", "uint",
	"uint8", "uint16", "uint32", "uint64", "uintptr",
	"any", "comparable",
	// Predeclared constants
	"true", "false", "iota", "nil",
	// Predeclared functions
	"append", "cap", "clear", "close", "complex",
	"copy", "delete", "imag", "len", "make",
	"max", "min", "new", "panic", "print",
	"println", "real", "recover",
)

// Escape appends "Val" to names that collide with Go reserved words.
func Escape(name string) string {
	if Reserved.Contains(name) {
		return name + "Val"
	}
	return name
}
