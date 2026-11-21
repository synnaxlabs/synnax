// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"fmt"
	"testing"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
)

// BenchmarkSearch measures a channel search as the table grows, over channels with no
// description: an exact name and a word that every channel holds.
func BenchmarkSearch(b *testing.B) {
	for _, count := range []int{1_000, 10_000} {
		env := newBenchEnv(b, count)
		for _, term := range []string{"bench_calc_42", "calc"} {
			b.Run(fmt.Sprintf("channels=%d/term=%s", count, term), func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					var found []channel.Channel
					if err := env.svc.NewRetrieve().
						Search(term).
						Limit(10).
						Entries(&found).
						Exec(b.Context(), env.db); err != nil {
						b.Fatalf("failed to search: %v", err)
					}
					if len(found) == 0 {
						b.Fatalf("found no channel for %q", term)
					}
				}
			})
		}
		env.close(b)
	}
}
