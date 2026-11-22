// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index_test

import (
	"fmt"
	"testing"

	"github.com/synnaxlabs/gorp/internal/index"
)

// BenchmarkMultiPopulate measures the bulk load of a Multi with 100,000 keys, each
// holding perKey of 1,000 shared values, from one reused buffer as a table scan feeds
// it.
func BenchmarkMultiPopulate(b *testing.B) {
	const n, pool = 100_000, 1000
	for _, tracked := range []bool{false, true} {
		for _, perKey := range []int{0, 1, 20} {
			name := fmt.Sprintf("tracked=%t/values=%d", tracked, perKey)
			b.Run(name, func(b *testing.B) {
				buf := make([]int32, perKey)
				b.ReportAllocs()
				for b.Loop() {
					m := index.NewMulti[*scope, int32, int32](tracked)
					insert, finish := m.Populate()
					for k := range int32(n) {
						for j := range buf {
							buf[j] = (k + int32(j)) % pool
						}
						insert(k, buf)
					}
					finish(nil)
				}
			})
		}
	}
}
