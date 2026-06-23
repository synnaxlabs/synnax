// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crdt_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/synnaxlabs/x/crdt"
)

// benchSizes are document lengths in characters. They bracket the realistic range for an
// Arc source file (low thousands) and push past it to expose the scaling curve.
var benchSizes = []int{100, 1000, 5000, 10000}

// buildDoc types n characters into a fresh document one at a time, appending each.
func buildDoc(n int) *crdt.Text {
	t := crdt.New(1)
	for i := range n {
		t.Insert(i, "a")
	}
	return t
}

// BenchmarkBuild measures typing a document of n characters from empty, one keystroke at
// a time. Each Insert re-traverses the whole document, so the build is O(n^2); dividing
// ns/op by n gives the average per-keystroke cost, which rises with n.
func BenchmarkBuild(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				t := crdt.New(1)
				for i := range n {
					t.Insert(i, "a")
				}
			}
		})
	}
}

// BenchmarkBuildAndRender measures the real editor loop: every keystroke inserts a
// character and then materializes the whole string (what the binding diffs against).
// Both halves are O(n) per keystroke, so the loop is O(n^2).
func BenchmarkBuildAndRender(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				t := crdt.New(1)
				for i := range n {
					t.Insert(i, "a")
					_ = t.String()
				}
			}
		})
	}
}

// BenchmarkApplyRemote measures a replica integrating n insert operations produced by
// another replica, delivered in order (the receiver's hot path while a peer types).
func BenchmarkApplyRemote(b *testing.B) {
	for _, n := range benchSizes {
		src := crdt.New(2)
		ops := src.Insert(0, strings.Repeat("a", n))
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				r := crdt.New(1)
				r.ApplyInsert(ops...)
			}
		})
	}
}

// BenchmarkBootstrap measures a joining client reconstructing a document of n characters
// from a server snapshot.
func BenchmarkBootstrap(b *testing.B) {
	for _, n := range benchSizes {
		ins, del := buildDoc(n).Snapshot()
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				crdt.New(1).Load(ins, del)
			}
		})
	}
}
