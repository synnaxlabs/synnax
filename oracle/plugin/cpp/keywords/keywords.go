// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package keywords

// Reserved contains all C++ reserved keywords.
var Reserved = map[string]bool{
	"alignas": true, "alignof": true, "and": true, "and_eq": true, "asm": true,
	"auto": true, "bitand": true, "bitor": true, "bool": true, "break": true,
	"case": true, "catch": true, "char": true, "char16_t": true, "char32_t": true,
	"class": true, "compl": true, "const": true, "constexpr": true, "const_cast": true,
	"continue": true, "decltype": true, "default": true, "delete": true, "do": true,
	"double": true, "dynamic_cast": true, "else": true, "enum": true, "explicit": true,
	"export": true, "extern": true, "false": true, "float": true, "for": true,
	"friend": true, "goto": true, "if": true, "inline": true, "int": true,
	"long": true, "mutable": true, "namespace": true, "new": true, "noexcept": true,
	"not": true, "not_eq": true, "nullptr": true, "operator": true, "or": true,
	"or_eq": true, "private": true, "protected": true, "public": true, "register": true,
	"reinterpret_cast": true, "return": true, "short": true, "signed": true,
	"sizeof": true, "static": true, "static_assert": true, "static_cast": true,
	"struct": true, "switch": true, "template": true, "this": true, "thread_local": true,
	"throw": true, "true": true, "try": true, "typedef": true, "typeid": true,
	"typename": true, "union": true, "unsigned": true, "using": true, "virtual": true,
	"void": true, "volatile": true, "wchar_t": true, "while": true, "xor": true,
	"xor_eq": true,
}

// Escape appends an underscore suffix to names that collide with C++ reserved keywords.
func Escape(name string) string {
	if Reserved[name] {
		return name + "_"
	}
	return name
}
