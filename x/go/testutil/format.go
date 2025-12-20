// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"strings"

	"github.com/onsi/gomega/format"
	"github.com/synnaxlabs/x/errors"
)

func init() {
	format.RegisterCustomFormatter(formatErrorWithStack)
}

// formatErrorWithStack is a custom Gomega formatter that extracts and displays
// stack traces from cockroachdb/errors when formatting errors in test assertions.
//
// This allows test failures to show not just where the assertion failed, but also
// where the error was originally created, making debugging significantly easier.
func formatErrorWithStack(value any) (string, bool) {
	err, ok := value.(error)
	if !ok {
		return "", false
	}
	var b strings.Builder
	if stackTrace := errors.GetStackTrace(err); stackTrace != nil {
		b.WriteString("\nError Origin Stack Trace:\n\n")
		stackTrace.StringBuilder(&b)
	} else {
		b.WriteString(err.Error())
	}
	return b.String(), true
}
