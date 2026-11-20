// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment_test

import (
	"fmt"
	"testing"

	"github.com/synnaxlabs/cesium/internal/segment"
	xfs "github.com/synnaxlabs/x/io/fs"
)

// BenchmarkWrite measures a write of one series to an open fixed-density segment in
// memory, which a commit then puts back so the file stays one block long.
func BenchmarkWrite(b *testing.B) {
	for _, size := range []int{512, 16 << 10, 1 << 20} {
		b.Run(fmt.Sprintf("bytes=%d", size), func(b *testing.B) {
			budget, err := segment.NewBudget(1)
			if err != nil {
				b.Fatal(err)
			}
			h, err := budget.Create(xfs.NewMem(), "1.seg")
			if err != nil {
				b.Fatal(err)
			}
			w := segment.NewWriter(h, cfg)
			data := pattern(size)
			b.SetBytes(int64(size))
			b.ReportAllocs()
			for b.Loop() {
				if err = w.Write(data); err != nil {
					b.Fatal(err)
				}
				w.PutBack()
			}
		})
	}
}
