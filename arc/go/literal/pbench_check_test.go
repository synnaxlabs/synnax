// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package literal_test

import (
	"testing"

	"github.com/synnaxlabs/arc/literal"
)

func BenchmarkCheckFmtStrParse(b *testing.B) {
	for _, body := range []string{
		"{x}",
		"value {x:%.2f} of {y + z}",
		"{a[0:2]} {b}",
		"{'pt-01':.1f}",
		"{x /* a */ + y:.1f}",
	} {
		b.Run(body, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if _, err := literal.FmtStrParse(body); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
