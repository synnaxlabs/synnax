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

// Reserved contains all Python reserved keywords.
var Reserved = set.New(
	"False", "None", "True", "and", "as",
	"assert", "async", "await", "break", "class",
	"continue", "def", "del", "elif", "else",
	"except", "finally", "for", "from", "global",
	"if", "import", "in", "is", "lambda",
	"nonlocal", "not", "or", "pass", "raise",
	"return", "try", "while", "with", "yield",
)

// Escape appends an underscore suffix to names that collide with Python reserved
// keywords.
func Escape(name string) string {
	if Reserved.Contains(name) {
		return name + "_"
	}
	return name
}
