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
)

// BenchmarkResolveNames measures an Arc resolver resolving N names, one by one, beside
// a table of 1,000 calculated channels. Each op resolves all N.
func BenchmarkResolveNames(b *testing.B) {
	env := newBenchEnv(b, 1_000)
	b.Cleanup(func() { env.close(b) })
	for _, n := range []int{1, 10, 100} {
		names := make([]string, n)
		for i := range names {
			names[i] = fmt.Sprintf("bench_calc_%d", i)
		}
		b.Run(fmt.Sprintf("names=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				r := env.svc.NewArcSymbolResolver(env.db)
				for _, name := range names {
					if _, err := r.Resolve(b.Context(), name); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
