// Copyright 2026 Synnax Labs, Inc.
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
	"io"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
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
	var b strings.Builder
	s.StringBuilder(&b)
	return b.String()
}

func mustPrintf(w io.Writer, format string, a ...any) int {
	return lo.Must1(fmt.Fprintf(w, format, a...))
}

func (s *StackTrace) StringBuilder(b *strings.Builder) {
	if s == nil || s.ReportableStackTrace == nil {
		return
	}
	for _, f := range s.Frames {
		funcName := f.Function
		if funcName == "" {
			funcName = f.Symbol
		}
		if funcName == "" {
			funcName = "<unknown>"
		}
		mustPrintf(b, "%s", funcName)

		if f.Module != "" {
			mustPrintf(b, " [%s]", f.Module)
		}
		mustPrintf(b, "\n\t")

		if f.Filename != "" {
			mustPrintf(b, "%s", f.Filename)
		} else if f.AbsPath != "" {
			mustPrintf(b, "%s", f.AbsPath)
		} else {
			mustPrintf(b, "<unknown file>")
		}

		if f.Lineno > 0 {
			mustPrintf(b, ":%d", f.Lineno)
			if f.Colno > 0 {
				mustPrintf(b, ":%d", f.Colno)
			}
		}
		lo.Must(fmt.Fprintln(b))
		if f.InstructionAddr != "" || f.SymbolAddr != "" {
			mustPrintf(
				b,
				"\t@ %s (symbol: %s, image: %s)\n",
				f.InstructionAddr,
				f.SymbolAddr,
				f.ImageAddr,
			)
		}
	}
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
