// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"testing"

	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
)

// BenchmarkAnalyze measures the analysis of one text program and one graph, each from a
// new root, with the standard library built for each root ("fresh") or built once and
// shared ("shared").
func BenchmarkAnalyze(b *testing.B) {
	ctx := b.Context()
	parsedText, diag := text.Parse(arc.Text{Raw: sharedLSPSource})
	if diag != nil && !diag.Ok() {
		b.Fatalf("failed to parse the text program: %s", diag)
	}
	parsedGraph, err := graph.Parse(sharedGraph(), parser.Config{})
	if err != nil {
		b.Fatalf("failed to parse the graph: %v", err)
	}
	shared := stl.NewSymbols()
	ambients := []struct {
		name    string
		ambient func() []*symbol.Symbol
	}{
		{name: "fresh", ambient: stl.NewSymbols},
		{name: "shared", ambient: func() []*symbol.Symbol { return shared }},
	}
	for _, a := range ambients {
		b.Run("text/"+a.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if _, diag := text.Analyze(
					ctx,
					parsedText,
					newRepositoryRoot(a.ambient()),
				); !diag.Ok() {
					b.Fatalf("failed to analyze the text program: %s", diag)
				}
			}
		})
		b.Run("graph/"+a.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				root := newRepositoryRoot(a.ambient())
				symbol.AutoImportModules(root)
				if _, diag := graph.Analyze(ctx, parsedGraph, root); !diag.Ok() {
					b.Fatalf("failed to analyze the graph: %s", diag)
				}
			}
		})
	}
}
