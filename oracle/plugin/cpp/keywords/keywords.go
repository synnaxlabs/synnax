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

// Reserved contains all C++ reserved keywords.
var Reserved = set.New[string](
	"alignas", "alignof", "and", "and_eq", "asm",
	"auto", "bitand", "bitor", "bool", "break",
	"case", "catch", "char", "char16_t", "char32_t",
	"class", "compl", "const", "constexpr", "const_cast",
	"continue", "decltype", "default", "delete", "do",
	"double", "dynamic_cast", "else", "enum", "explicit",
	"export", "extern", "false", "float", "for",
	"friend", "goto", "if", "inline", "int",
	"long", "mutable", "namespace", "new", "noexcept",
	"not", "not_eq", "nullptr", "operator", "or",
	"or_eq", "private", "protected", "public", "register",
	"reinterpret_cast", "return", "short", "signed",
	"sizeof", "static", "static_assert", "static_cast",
	"struct", "switch", "template", "this", "thread_local",
	"throw", "true", "try", "typedef", "typeid",
	"typename", "union", "unsigned", "using", "virtual",
	"void", "volatile", "wchar_t", "while", "xor",
	"xor_eq",
)

// Escape appends an underscore suffix to names that collide with C++ reserved keywords.
func Escape(name string) string {
	if Reserved.Contains(name) {
		return name + "_"
	}
	return name
}
