// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package selector holds the rule for the selectors that name shapes in the drawing of
// a symbol.
package selector

import "regexp"

const (
	escapeUnit = `\\[0-9a-fA-F]{1,6} ?|\\[^0-9a-fA-F\r\n\f]`
	identStart = `(?:[A-Za-z_]|[^\x00-\x7F]|` + escapeUnit + `)`
	identUnit  = `(?:[\w\-]|[^\x00-\x7F]|` + escapeUnit + `)`
	// ident never starts with a digit, or with a hyphen and a digit.
	ident = `(?:--|-?` + identStart + `)` + identUnit + `*`
)

var name = regexp.MustCompile(
	`^(?:#` + ident + `|(?:\.` + ident + `)+|` +
		`\[data-region-id="(?:[^"\\]|\\.)*"\])$`,
)

// IsName reports whether sel names shapes by #id, .class.class, or
// [data-region-id="..."], with each id and class a valid CSS identifier.
func IsName(sel string) bool { return name.MatchString(sel) }
