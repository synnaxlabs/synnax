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
	"fmt"
	"strings"
	"testing"

	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// program returns a plausible arc source of the given line count. Hash is computed on
// every retrieve, create, and dispatch, so it must stay cheap into the thousands of
// lines.
func program(lines int) string {
	var b strings.Builder
	for i := range lines {
		fmt.Fprintf(&b, "valve_%d -> chamber_%d\n", i, i)
	}
	return b.String()
}

func bigGraphArc(nodes int) arc.Arc {
	g := graph.Graph{Inputs: map[string]msgpack.EncodedJSON{}}
	for i := range nodes {
		key := fmt.Sprintf("node_%d", i)
		g.Nodes = append(g.Nodes, graph.Node{Key: key})
		g.Inputs[key] = msgpack.EncodedJSON{"type": "on", "channel": i}
		if i > 0 {
			g.Edges = append(g.Edges, graph.Edge{
				Key: fmt.Sprintf("e%d", i),
				Edge: ir.Edge{
					Source: ir.Handle{Node: fmt.Sprintf("node_%d", i-1), Param: "out"},
					Target: ir.Handle{Node: key, Param: "in"},
				},
			})
		}
	}
	return arc.Arc{Mode: arc.ModeGraph, Graph: g}
}

func benchHash(b *testing.B, a arc.Arc) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := arc.Hash(a); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHash(b *testing.B) {
	b.Run("text-10-lines", func(b *testing.B) {
		benchHash(b, textArc(program(10)))
	})
	b.Run("text-1000-lines", func(b *testing.B) {
		benchHash(b, textArc(program(1000)))
	})
	b.Run("graph-10-nodes", func(b *testing.B) {
		benchHash(b, bigGraphArc(10))
	})
	b.Run("graph-100-nodes", func(b *testing.B) {
		benchHash(b, bigGraphArc(100))
	})
}
