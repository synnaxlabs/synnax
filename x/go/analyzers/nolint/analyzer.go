// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package nolint

import (
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzer reports each //nolint directive.
var Analyzer = &analysis.Analyzer{
	Name: "nolint",
	Doc: `forbids //nolint directives.

A directive hides the finding that points at a defect. Fix the code, or leave the linter
disabled in .golangci.yaml.`,
	Run: run,
}

// directive matches the comment text golangci-lint reads as a //nolint directive.
var directive = regexp.MustCompile(`^nolint( |:|$)`)

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, group := range file.Comments {
			for _, c := range group.List {
				if directive.MatchString(strings.TrimLeft(c.Text, "/ ")) {
					pass.Reportf(c.Pos(), "//nolint directive; fix the finding instead")
				}
			}
		}
	}
	return nil, nil
}
