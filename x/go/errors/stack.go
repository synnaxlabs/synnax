// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
)

// GetStackTrace attempts to pull the stack trace from the given error. If the
// stack trace is not available, returns an empty stack trace.
func GetStackTrace(err error) *StackTrace {
	return &StackTrace{ReportableStackTrace: errors.GetReportableStackTrace(err)}
}

// StackTrace provides information about the stack trace of an error.
type StackTrace struct{ *errors.ReportableStackTrace }

// String implements fmt.Stringer.
func (s *StackTrace) String() string {
	if s == nil || s.ReportableStackTrace == nil {
		return ""
	}
	var b strings.Builder
	for _, f := range s.Frames {
		funcName := f.Function
		if funcName == "" {
			funcName = f.Symbol
		}
		if funcName == "" {
			funcName = "<unknown>"
		}
		_, _ = fmt.Fprintf(&b, "%s", funcName)

		if f.Module != "" {
			_, _ = fmt.Fprintf(&b, " [%s]", f.Module)
		}
		_, _ = fmt.Fprint(&b, "\n\t")

		if f.Filename != "" {
			_, _ = fmt.Fprintf(&b, "%s", f.Filename)
		} else if f.AbsPath != "" {
			_, _ = fmt.Fprintf(&b, "%s", f.AbsPath)
		} else {
			_, _ = fmt.Fprintf(&b, "<unknown file>")
		}

		if f.Lineno > 0 {
			_, _ = fmt.Fprintf(&b, ":%d", f.Lineno)
			if f.Colno > 0 {
				_, _ = fmt.Fprintf(&b, ":%d", f.Colno)
			}
		}
		_, _ = fmt.Fprintln(&b)
		if f.InstructionAddr != "" || f.SymbolAddr != "" {
			_, _ = fmt.Fprintf(
				&b,
				"\t@ %s (symbol: %s, image: %s)\n",
				f.InstructionAddr,
				f.SymbolAddr,
				f.ImageAddr,
			)
		}
	}

	return b.String()
}

// WithStack annotates err with a stack trace at the point WithStack was called.
func WithStack(err error) error {
	return WithStackDepth(err, 1)
}

// WithStackDepth annotates an error with a stack trace starting from the given call depth.
// Zero identifies the caller of WithStackDepth itself. See the documentation
// of WithStack() for more details.
func WithStackDepth(err error, depth int) error {
	return errors.WithStackDepth(err, depth+1)
}
