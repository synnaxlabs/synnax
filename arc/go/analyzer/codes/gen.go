// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build ignore

// Writes the table of kinds to codes.gen.json, which the docs site reads.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

type kind struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

var severities = map[protocol.DiagnosticSeverity]string{
	protocol.DiagnosticSeverityError:   "error",
	protocol.DiagnosticSeverityWarning: "warning",
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func main() {
	kinds := make(map[diagnostics.ErrorCode]kind, len(codes.All))
	for _, k := range codes.All {
		if _, ok := kinds[k.Code]; ok {
			fail("two kinds have the code %s", k.Code)
		}
		severity, ok := severities[k.Severity]
		if !ok {
			fail("%s has a severity that the docs do not show", k.Code)
		}
		kinds[k.Code] = kind{Severity: severity, Message: k.Message}
	}
	b, err := json.MarshalIndent(kinds, "", "  ")
	if err != nil {
		fail("failed to encode the kinds: %v", err)
	}
	if err = os.WriteFile("codes.gen.json", append(b, '\n'), 0o644); err != nil {
		fail("failed to write the kinds: %v", err)
	}
}
