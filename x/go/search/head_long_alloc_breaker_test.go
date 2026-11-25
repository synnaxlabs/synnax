// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package search_test

import (
	"testing"

	"github.com/synnaxlabs/x/search"
)

// BenchmarkBreakerLongDescription indexes names with a 1,024-character description as
// secondary text.
func BenchmarkBreakerLongDescription(b *testing.B) {
	names := makeNames(1000)
	descs := make([]string, len(names))
	for i := range descs {
		descs[i] = breakerDescription(i)
	}
	b.ReportAllocs()
	for b.Loop() {
		idx := search.New[int]()
		for i, n := range names {
			idx.SetText(i, []string{n}, []string{descs[i]})
		}
	}
}
