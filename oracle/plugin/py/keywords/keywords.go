// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package keywords

import (
	_ "embed"
	"strings"

	"github.com/synnaxlabs/x/set"
)

var reserved = set.New(
	"False", "None", "True", "and", "as",
	"assert", "async", "await", "break", "class",
	"continue", "def", "del", "elif", "else",
	"except", "finally", "for", "from", "global",
	"if", "import", "in", "is", "lambda",
	"nonlocal", "not", "or", "pass", "raise",
	"return", "try", "while", "with", "yield",
)

// modelAttributes lists the public attributes of a Pydantic BaseModel, one per line. A
// test in the Python client checks it against the installed Pydantic.
//
//go:embed model_attributes.txt
var modelAttributes string

var attributes = set.New(strings.Fields(modelAttributes)...)

// EscapeField appends an underscore suffix to a model field name that collides with a
// Python reserved keyword or with an attribute of a Pydantic BaseModel.
func EscapeField(name string) string {
	if attributes.Contains(name) {
		return name + "_"
	}
	return Escape(name)
}

// Escape appends an underscore suffix to names that collide with Python reserved
// keywords.
func Escape(name string) string {
	if reserved.Contains(name) {
		return name + "_"
	}
	return name
}
