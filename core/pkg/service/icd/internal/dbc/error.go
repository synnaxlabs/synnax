// Copyright 2019 Einride AB. Modified by Synnax Labs, Inc.
//
// Use of this source code is governed by the MIT license in the LICENSE file in this
// directory.

package dbc

import (
	"fmt"
	"text/scanner"
)

// Error is a failure to parse a DBC file.
type Error struct {
	// Pos is where in the file parsing failed.
	Pos scanner.Position
	// Reason says why parsing failed.
	Reason string
}

var _ error = (*Error)(nil)

// Error implements error. It names the line and column of the failure.
func (e *Error) Error() string {
	return fmt.Sprintf("line %d, column %d: %s", e.Pos.Line, e.Pos.Column, e.Reason)
}
